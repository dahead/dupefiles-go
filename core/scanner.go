package core

import (
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"time"
)

const sizeThreshold = 2 * 1024 * 1024 * 1024 // 2GB

type ResultList struct {
	HashSum   string
	FileGuids []string
}

type Scanner struct {
	idx *Index
}

func NewScanner(idx *Index) *Scanner {
	return &Scanner{idx: idx}
}

// ScanBySize groups files by size
func (s *Scanner) ScanBySize() (map[int64][]*FileItem, error) {
	sizeGroups := make(map[int64][]*FileItem)
	fmt.Println("Scanning for size equivalent files...")
	for _, file := range s.idx.files {
		sizeGroups[file.Size] = append(sizeGroups[file.Size], file)
	}
	return sizeGroups, nil
}

func (s *Scanner) ScanForDuplicates() ([]ResultList, error) {
	// Step 1: Group files by size
	sizeGroups, err := s.ScanBySize()
	if err != nil {
		return nil, err
	}

	// Step 2: Calculate hashes for files in each size group
	finalHashGroups, err := s.ScanByHash(sizeGroups)
	if err != nil {
		return nil, err
	}

	// Step 3: Find actual duplicates by comparing file contents
	fmt.Println("Verifying potential duplicates...")
	var results []ResultList
	var resultsMu sync.Mutex

	var wg sync.WaitGroup
	resultsChan := make(chan ResultList, len(finalHashGroups))
	semaphore := make(chan struct{}, runtime.NumCPU()) // Limit concurrent hash groups

	for hash, filesInHashGroup := range finalHashGroups {
		if len(filesInHashGroup) < 2 {
			continue
		}

		wg.Add(1)
		go func(h string, files []*FileItem) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := s.findDuplicatesInHashGroup(h, files)
			if result != nil {
				if err := s.addDuplicatesToIndex(result); err != nil {
					fmt.Printf("Warning: %v\n", err)
				}
				resultsChan <- *result
			}
		}(hash, filesInHashGroup)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		resultsMu.Lock()
		results = append(results, result)
		resultsMu.Unlock()
	}

	return results, nil
}

func (s *Scanner) ScanByHash(sizeGroups map[int64][]*FileItem) (map[string][]*FileItem, error) {
	hashGroups, hashesToUpdate, err := s.calculateHashGroups(sizeGroups)
	if err != nil {
		return nil, err
	}

	if err := s.updateHashesInIndex(hashesToUpdate); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	return hashGroups, nil
}

func (s *Scanner) calculateHashGroups(sizeGroups map[int64][]*FileItem) (map[string][]*FileItem, []struct{ guid, hash string }, error) {
	finalHashGroups := make(map[string][]*FileItem)
	var allHashesToUpdate []struct{ guid, hash string }

	fmt.Println("Scanning for hash equivalent files...")
	totalSizeGroups := len(sizeGroups)
	processedSizeGroups := 0

	for size, filesInGroup := range sizeGroups {
		processedSizeGroups++
		if len(filesInGroup) < 2 {
			continue
		}

		if s.idx.config.Debug {
			fmt.Printf("- processing size group %d/%d (size: %s bytes, files: %d)\n",
				processedSizeGroups, totalSizeGroups, HumanizeBytes(size), len(filesInGroup))
		}

		// create list of files to create hash sums
		filesToHash := []*FileItem{}
		for _, file := range filesInGroup {
			if !file.Hash.Valid {
				filesToHash = append(filesToHash, file)
			} else {
				finalHashGroups[file.Hash.String] = append(finalHashGroups[file.Hash.String], file)
			}
		}

		// create hash sums
		if len(filesToHash) > 0 {
			type hashCalcResult struct {
				file    *FileItem
				hashStr string
				err     error
			}

			numJobs := len(filesToHash)
			jobsChan := make(chan *FileItem, numJobs)
			resultsChan := make(chan hashCalcResult, numJobs)
			var wg sync.WaitGroup

			numWorkers := calculateOptimalWorkers(numJobs)

			if s.idx.config.Debug {
				fmt.Printf("  Calculating %d hashes with %d workers...\n", numJobs, numWorkers)
			}

			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for jobFile := range jobsChan {
						var calculatedHash string
						var err error
						if jobFile.Hash.Valid && jobFile.Hash.String != "" {
							calculatedHash = jobFile.Hash.String
						} else {
							if s.idx.config.Debug {
								fmt.Printf("  Calculating hash for file %s...\n", jobFile.Path)
							}
							calculatedHash, err = CalculateFileHash(jobFile.Path, jobFile.Size)
						}
						resultsChan <- hashCalcResult{file: jobFile, hashStr: calculatedHash, err: err}
					}
				}()
			}

			for _, file := range filesToHash {
				jobsChan <- file
			}
			close(jobsChan)

			wg.Wait()
			close(resultsChan)

			var hashesToUpdateInDB []struct{ guid, hash string }
			for res := range resultsChan {
				if res.err != nil {
					fmt.Printf("  Warning: Failed to calculate hash for %s: %v\n", res.file.Path, res.err)
					continue
				}
				res.file.Hash = sql.NullString{String: res.hashStr, Valid: true}
				finalHashGroups[res.hashStr] = append(finalHashGroups[res.hashStr], res.file)
				hashesToUpdateInDB = append(hashesToUpdateInDB, struct{ guid, hash string }{res.file.Guid, res.hashStr})
			}

			allHashesToUpdate = append(allHashesToUpdate, hashesToUpdateInDB...)
		}
	}

	return finalHashGroups, allHashesToUpdate, nil
}

func calculateOptimalWorkers(numJobs int) int {
	numWorkers := runtime.NumCPU()
	if numWorkers > numJobs {
		numWorkers = numJobs
	}
	if numWorkers == 0 {
		numWorkers = 1
	}
	return numWorkers
}

// Updates hash values in the database
func (s *Scanner) updateHashesInIndex(hashesToUpdate []struct{ guid, hash string }) error {
	if len(hashesToUpdate) == 0 {
		return nil
	}

	tx, err := s.idx.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE files SET hash = ? WHERE guid = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	updatedCount := 0
	for _, h := range hashesToUpdate {
		_, err := stmt.Exec(h.hash, h.guid)
		if err != nil {
			fmt.Printf("  Warning: Failed to update hash for %s in DB: %v\n", h.guid, err)
		} else {
			updatedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if s.idx.config.Debug {
		fmt.Printf("  Updated %d hashes in DB.\n", updatedCount)
	}
	return nil
}

func (s *Scanner) findDuplicatesInHashGroup(hash string, filesInHashGroup []*FileItem) *ResultList {
	if len(filesInHashGroup) < 2 {
		return nil
	}

	var actualDuplicatesThisGroup []*FileItem
	actualDuplicatesThisGroup = append(actualDuplicatesThisGroup, filesInHashGroup[0])

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(chan struct {
		file      *FileItem
		identical bool
		err       error
	}, len(filesInHashGroup)-1)

	for i := 1; i < len(filesInHashGroup); i++ {
		wg.Add(1)
		go func(fileToCompare *FileItem) {
			defer wg.Done()
			identical, err := compareFilesBinarySampleSize(filesInHashGroup[0].Path, fileToCompare.Path, s.idx.config.BinaryCompareBytes)
			results <- struct {
				file      *FileItem
				identical bool
				err       error
			}{fileToCompare, identical, err}
		}(filesInHashGroup[i])
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			fmt.Printf("  Warning: Failed to compare %s and %s: %v\n", filesInHashGroup[0].Path, result.file.Path, result.err)
			continue
		}
		if result.identical {
			mu.Lock()
			actualDuplicatesThisGroup = append(actualDuplicatesThisGroup, result.file)
			mu.Unlock()
		}
	}

	if len(actualDuplicatesThisGroup) >= 2 {
		var duplicateGuids []string
		for _, f := range actualDuplicatesThisGroup {
			duplicateGuids = append(duplicateGuids, f.Guid)
		}
		return &ResultList{
			HashSum:   hash,
			FileGuids: duplicateGuids,
		}
	}

	return nil
}

func (s *Scanner) addDuplicatesToIndex(resultList *ResultList) error {
	if resultList == nil || len(resultList.FileGuids) < 2 {
		return nil
	}

	now := time.Now().Unix()
	tx, err := s.idx.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT INTO duplicates (guid, scanned) 
        VALUES (?, ?) 
        ON CONFLICT(guid) DO UPDATE SET scanned = ?
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, guid := range resultList.FileGuids {
		_, err := stmt.Exec(guid, now, now)
		if err != nil {
			return fmt.Errorf("failed to update duplicate for %s: %w", guid, err)
		}
	}

	return tx.Commit()
}
