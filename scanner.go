package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

const sizeThreshold = 2 * 1024 * 1024 * 1024 // 2GB

type ResultList struct {
	HashSum   string
	FileGuids []string
}

func (idx *Index) CalculateHash(file *FileItem) (string, error) {
	// Open file
	f, err := os.Open(file.Path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Choose hash algorithm based on file size
	var h hash.Hash
	if file.Size < sizeThreshold {
		h = md5.New()
	} else {
		h = sha256.New()
	}

	// Calculate hash
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}

	// Return hash as hex string
	hashStr := hex.EncodeToString(h.Sum(nil))

	// Update file hash in memory
	file.Hash = sql.NullString{String: hashStr, Valid: true}

	// Update file hash in database
	_, err = idx.db.Exec("UPDATE files SET hash = ? WHERE guid = ?", hashStr, file.Guid)
	if err != nil {
		return "", err
	}

	return hashStr, nil
}

func CompareFilesBinary(path1, path2 string) (bool, error) {
	// Open first file
	f1, err := os.Open(path1)
	if err != nil {
		return false, err
	}
	defer f1.Close()

	// Open second file
	f2, err := os.Open(path2)
	if err != nil {
		return false, err
	}
	defer f2.Close()

	// Compare files in chunks
	const chunkSize = 64 * 1024 // 64KB
	buf1 := make([]byte, chunkSize)
	buf2 := make([]byte, chunkSize)

	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)

		// Check for read errors
		if err1 != nil && err1 != io.EOF {
			return false, err1
		}
		if err2 != nil && err2 != io.EOF {
			return false, err2
		}

		// Check if read sizes differ
		if n1 != n2 {
			return false, nil
		}

		// Check if both files are at EOF
		if err1 == io.EOF && err2 == io.EOF {
			return true, nil
		}

		// Compare chunks
		if !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}
	}
}

func (idx *Index) ScanForDuplicates() ([]ResultList, error) {

	// Group files by size
	sizeGroups := make(map[int64][]*FileItem)
	for _, file := range idx.files {
		sizeGroups[file.Size] = append(sizeGroups[file.Size], file)
	}

	// Process each size group
	var results []ResultList
	hashGroups := make(map[string][]*FileItem)

	fmt.Println("Scanning for duplicates...")
	totalGroups := len(sizeGroups)
	processedGroups := 0

	for size, files := range sizeGroups {
		// Skip groups with only one file
		if len(files) < 2 {
			processedGroups++
			continue
		}

		fmt.Printf("Processing size group %d/%d (size: %d bytes, files: %d)\n",
			processedGroups+1, totalGroups, size, len(files))

		// Calculate hashes for files in this size group
		for _, file := range files {
			if !file.Hash.Valid {
				hash, err := idx.CalculateHash(file)
				if err != nil {
					fmt.Printf("Warning: Failed to calculate hash for %s: %v\n", file.Path, err)
					continue
				}
				file.Hash.String = hash
				file.Hash.Valid = true
			}
			hashGroups[file.Hash.String] = append(hashGroups[file.Hash.String], file)
		}

		processedGroups++
	}

	// Process hash groups
	fmt.Println("Verifying potential duplicates...")
	totalHashGroups := len(hashGroups)
	processedHashGroups := 0

	for hash, files := range hashGroups {
		// Skip groups with only one file
		if len(files) < 2 {
			processedHashGroups++
			continue
		}

		fmt.Printf("Processing hash group %d/%d (hash: %s, files: %d)\n",
			processedHashGroups+1, totalHashGroups, hash[:8]+"...", len(files))

		// Compare files byte by byte
		duplicateGuids := []string{}

		// Compare each file with every other file
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				file1 := files[i]
				file2 := files[j]

				// Skip if already confirmed as duplicates
				if contains(duplicateGuids, file1.Guid) && contains(duplicateGuids, file2.Guid) {
					continue
				}

				identical, err := CompareFilesBinary(file1.Path, file2.Path)
				if err != nil {
					fmt.Printf("Warning: Failed to compare %s and %s: %v\n", file1.Path, file2.Path, err)
					continue
				}

				if identical {
					// Add to duplicates if not already added
					if !contains(duplicateGuids, file1.Guid) {
						duplicateGuids = append(duplicateGuids, file1.Guid)
					}
					if !contains(duplicateGuids, file2.Guid) {
						duplicateGuids = append(duplicateGuids, file2.Guid)
					}
				}
			}
		}

		// If duplicates found, add to results
		if len(duplicateGuids) >= 2 {
			results = append(results, ResultList{
				HashSum:   hash,
				FileGuids: duplicateGuids,
			})
		}

		processedHashGroups++
	}

	return results, nil
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
