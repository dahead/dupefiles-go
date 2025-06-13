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
	// Use s.idx to access the files from the Index struct
	for _, file := range s.idx.files {
		sizeGroups[file.Size] = append(sizeGroups[file.Size], file)
	}

	return sizeGroups, nil
}

// Calculates hashes for files in each size group
func (s *Scanner) ScanByHash(sizeGroups map[int64][]*FileItem) (map[string][]*FileItem, error) {
	finalHashGroups := make(map[string][]*FileItem)

	fmt.Println("Scanning for hash equivalent files...")
	totalSizeGroups := len(sizeGroups)
	processedSizeGroups := 0

	for _, filesInGroup := range sizeGroups {
		processedSizeGroups++
		if len(filesInGroup) < 2 {
			continue
		}

		fmt.Printf("Processing size group %d/%d\n",
			processedSizeGroups, totalSizeGroups)

		//fmt.Printf("Processing size group %d/%d (size: %s bytes, files: %d)\n",
		//	processedSizeGroups, totalSizeGroups, HumanizeBytes(size), len(filesInGroup))

		// fmt.Printf(".")

		filesToHash := []*FileItem{}
		for _, file := range filesInGroup {
			if !file.Hash.Valid {
				filesToHash = append(filesToHash, file)
			} else {
				finalHashGroups[file.Hash.String] = append(finalHashGroups[file.Hash.String], file)
			}
		}

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

			numWorkers := 8
			if numWorkers > numJobs {
				numWorkers = numJobs
			}
			// Ensure at least one worker
			if numWorkers == 0 {
				numWorkers = 1
			}

			// fmt.Printf("  Calculating %d hashes with %d workers...\n", numJobs, numWorkers)

			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for jobFile := range jobsChan {
						var calculatedHash string
						var err error
						// First try to lookup hash in index
						if jobFile.Hash.Valid && jobFile.Hash.String != "" {
							// Hash already exists in index, use it
							calculatedHash = jobFile.Hash.String
						} else {
							// Hash not found in index, calculate it
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

			hashesToUpdateInDB := []struct {
				guid string
				hash string
			}{}

			for res := range resultsChan {
				if res.err != nil {
					fmt.Printf("  Warning: Failed to calculate hash for %s: %v\n", res.file.Path, res.err)
					continue
				}
				res.file.Hash = sql.NullString{String: res.hashStr, Valid: true} // UpdateIndex in-memory FileItem
				finalHashGroups[res.hashStr] = append(finalHashGroups[res.hashStr], res.file)
				hashesToUpdateInDB = append(hashesToUpdateInDB, struct {
					guid string
					hash string
				}{res.file.Guid, res.hashStr})
			}

			// Batch update hashes in DB
			if len(hashesToUpdateInDB) > 0 {
				tx, err := s.idx.db.Begin()
				if err != nil {
					fmt.Printf("  Warning: Failed to begin transaction for hash updates: %v\n", err)
				} else {
					stmt, err := tx.Prepare("UPDATE files SET hash = ? WHERE guid = ?")
					if err != nil {
						fmt.Printf("  Warning: Failed to prepare statement for hash updates: %v\n", err)
						tx.Rollback()
					} else {
						updatedCount := 0
						for _, h := range hashesToUpdateInDB {
							_, err := stmt.Exec(h.hash, h.guid)
							if err != nil {
								fmt.Printf("  Warning: Failed to update hash for %s in DB: %v\n", h.guid, err)
							} else {
								updatedCount++
							}
						}
						stmt.Close()
						err = tx.Commit()
						if err != nil {
							fmt.Printf("  Warning: Failed to commit transaction for hash updates: %v\n", err)
						} else {
							// fmt.Printf("  Updated %d hashes in DB for current size group.\n", updatedCount)
						}
					}
				}
			}
		}
	}

	return finalHashGroups, nil
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
	var results []ResultList
	var resultsMu sync.Mutex

	fmt.Println("Verifying potential duplicates...")

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
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			result := s.processHashGroupParallel(h, files)
			if result != nil {
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

func (s *Scanner) processHashGroupParallel(hash string, filesInHashGroup []*FileItem) *ResultList {
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
			identical, err := CompareFilesBinaryRandom(filesInHashGroup[0].Path, fileToCompare.Path, s.idx.config.BinaryCompareBytes)
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
		// Record duplicates in the duplicates table
		now := time.Now().Unix()
		tx, err := s.idx.db.Begin()
		if err != nil {
			fmt.Printf("  Warning: Failed to begin transaction for duplicate updates: %v\n", err)
		} else {
			stmt, err := tx.Prepare(`
               INSERT INTO duplicates (guid, scanned) 
               VALUES (?, ?) 
               ON CONFLICT(guid) DO UPDATE SET scanned = ?
           `)
			if err != nil {
				fmt.Printf("  Warning: Failed to prepare statement for duplicate updates: %v\n", err)
				tx.Rollback()
			} else {
				for _, f := range actualDuplicatesThisGroup {
					_, err := stmt.Exec(f.Guid, now, now)
					if err != nil {
						fmt.Printf("  Warning: Failed to update duplicate for %s in DB: %v\n", f.Guid, err)
					}
				}
				stmt.Close()
				err = tx.Commit()
				if err != nil {
					fmt.Printf("  Warning: Failed to commit transaction for duplicate updates: %v\n", err)
				}
			}
		}

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
