package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type App struct {
	index  *Index
	config *Config
}

func NewApp() *App {
	config := NewConfig()
	idx, err := NewIndex(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating database: %v\n", err)
		os.Exit(1)
	}

	return &App{
		index:  idx,
		config: config,
	}
}

func (a *App) Close() {
	if a.index != nil {
		a.index.Close()
	}
}

func (a *App) ShowConfig() {
	fmt.Printf("*** Environment Configuration: ***\n")
	fmt.Printf("- Debug: %v\n", a.config.Debug)
	fmt.Printf("- DryRun: %v\n", a.config.DryRun)
	fmt.Printf("- Database file: %s\n", a.index.GetIndexPath())
	fmt.Printf("- Minimum file size: %d bytes\n", a.config.MinFileSize)
	fmt.Printf("- Sample size in bytes for binary comparism: %d bytes\n", a.config.SampleSizeBinaryCompare)
	fmt.Printf("- Database path: %s\n", a.config.DBFilename)
	fmt.Printf("- System trash directory: %s\n", GetTrashPath())
}

func (a *App) ShowFiles() {
	files := a.index.GetAllFiles()
	if len(files) == 0 {
		fmt.Println("No files in database")
		return
	}
	// show each file path
	for _, file := range files {
		fmt.Println(file.Path)
	}
	// show totals
	fmt.Printf("Files in database: %d total.\n", len(files))
}

func (a *App) ShowDupes() {
	files := a.index.GetAllDupes()
	if len(files) == 0 {
		fmt.Println("No duplicate files in database")
		return
	}
	// show each file path
	for _, file := range files {
		fmt.Println(file.Path)
	}
	// show totals
	fmt.Printf("Duplicate files in database: %d total.\n", len(files))
}

func (a *App) ShowHashes() {
	files := a.index.GetAllHashedFiles()
	if len(files) == 0 {
		fmt.Println("No hashed files in database")
		return
	}
	// show each file hash
	for _, file := range files {
		fmt.Println(file.Path)
	}
	// show totals
	fmt.Printf("Hashed files in database: %d total.\n", len(files))
}

func (a *App) StartScan() {

	// No files in FileIndex skip
	files := a.index.GetAllFiles()
	if len(files) == 0 {
		fmt.Println("No files in database. Nothing to scan.")
		return
	}

	// Start
	start := time.Now()
	scanner := NewScanner(a.index)              // Create scanner instance
	results, err := scanner.ScanForDuplicates() // Call method on scanner

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Stop and calculate duration
	duration := time.Since(start)
	if a.config.Debug {
		fmt.Printf("Scan finished in %v\n", duration)
	}

	// Print results
	if len(results) == 0 {
		fmt.Println("No duplicate files found!\n")
	} else {
		fmt.Printf("Found %d group(s) of duplicate files:\n", len(results))

		totalDuplicateSize := int64(0)
		totalDuplicateFiles := 0

		for i, result := range results {
			fmt.Printf("\nGroup %d (Hash: %s):\n", i+1, result.HashSum)
			groupSize := int64(0)
			var firstFile *FileItem

			for j, guid := range result.FileGuids {
				file := a.index.GetFileByGuid(guid)
				if file != nil {
					if firstFile == nil {
						firstFile = file
					}
					fmt.Printf("  %s (%s)\n", file.Path, file.HumanizedSize)
					if j > 0 { // Count all duplicates except the first (original)
						totalDuplicateFiles++
						groupSize += file.Size
					}
				}
			}
			totalDuplicateSize += groupSize
		}

		// Summary
		fmt.Printf("\nSummary: %d duplicate file(s) in %d group(s), %s used space\n",
			totalDuplicateFiles, len(results), HumanizeBytes(totalDuplicateSize))
	}
}

func (a *App) IndexPurge() {
	count, err := a.index.Purge()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Purged %d files from the database\n", count)
}

func (a *App) IndexUpdate() {
	count, err := a.index.Update()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Updated %d files in the database\n", count)
}

func (a *App) AddPathToIndex(path string, recursive bool, filter string) {
	if path == "" {
		fmt.Fprintf(os.Stderr, "Error: No path specified\n")
		os.Exit(1)
	}

	// remember  current amount of indexed files
	currentCount := len(a.index.GetAllFiles())

	// add directory or file
	fileItems, err := a.getFileInfos(path, recursive, filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	err = a.index.AddFileItems(fileItems)

	// remember new amount of indexed files
	newCount := len(a.index.GetAllFiles())
	// display changed files
	fmt.Printf("Updated %d files\n", newCount-currentCount)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func (a *App) getFileInfos(dirPath string, recursive bool, filter string) ([]*FileItem, error) {
	var fileItems []*FileItem
	minFileSize := a.index.config.MinFileSize

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			if !recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Filter check
		if filter != "" {
			if matched, _ := filepath.Match(filter, filepath.Base(path)); !matched {
				return nil
			}
		}

		// Size check
		if minFileSize > 0 && info.Size() < minFileSize {
			return nil
		}

		fileItems = append(fileItems, &FileItem{
			Guid:          filepath.Clean(path),
			Path:          path,
			Extension:     strings.TrimPrefix(filepath.Ext(path), "."),
			Size:          info.Size(),
			HumanizedSize: HumanizeBytes(info.Size()),
			ModTime:       info.ModTime().Unix(),
			Hash:          sql.NullString{String: "", Valid: false},
		})

		// fmt.Printf("  %s\n", path)

		return nil
	})

	return fileItems, err
}

func (a *App) RemovePathFromIndex(path string) {
	if path == "" {
		fmt.Fprintf(os.Stderr, "Error: No path specified\n")
		os.Exit(1)
	}

	// Check if path exists
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Normalize path for comparison
	normalizedPath := filepath.Clean(path)

	// Begin transaction
	tx, err := a.index.db.Begin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback() // Rollback if not committed

	// Prepare statement for deleting files
	stmt, err := tx.Prepare("DELETE FROM files WHERE path = ? OR path LIKE ?")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to prepare statement: %v\n", err)
		os.Exit(1)
	}
	defer stmt.Close()

	// Execute statement
	result, err := stmt.Exec(normalizedPath, normalizedPath+string(filepath.Separator)+"%")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to execute statement: %v\n", err)
		os.Exit(1)
	}

	// Get number of affected rows
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get rows affected: %v\n", err)
		os.Exit(1)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to commit transaction: %v\n", err)
		os.Exit(1)
	}

	// Remove from in-memory index
	removedFromMemory := 0
	for guid, file := range a.index.files {
		if file.Path == normalizedPath || strings.HasPrefix(file.Path, normalizedPath+string(filepath.Separator)) {
			delete(a.index.files, guid)
			removedFromMemory++
		}
	}

	fmt.Printf("Removed %d files from database\n", rowsAffected)
}

func (a *App) MoveDuplicateFilesToDirectory(path string) {
	if path == "" {
		fmt.Fprintf(os.Stderr, "Error: No path specified\n")
		os.Exit(1)
	}

	dirInfo, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !dirInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", path)
		os.Exit(1)
	}

	// move files to directory - only duplicates, keeping the first file of each size+hash group
	files := a.index.GetRestOfDuplicates()
	movedCount := 0

	for _, file := range files {
		// Get the base filename from the original path
		baseFileName := filepath.Base(file.Path)
		// Create the destination path by joining the target directory with the filename
		destPath := filepath.Join(path, baseFileName)

		// Check if destination file already exists
		if _, err := os.Stat(destPath); err == nil {
			// File exists, create a unique name
			ext := filepath.Ext(baseFileName)
			name := baseFileName[:len(baseFileName)-len(ext)]
			destPath = filepath.Join(path, fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext))
		}

		if a.config.DryRun {
			fmt.Printf("Would move %s to %s\n", file.Path, destPath)
		}

		if !a.config.DryRun {

			// Todo: invalid cross-device link

			err = os.Rename(file.Path, destPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error moving %s: %v\n", file.Path, err)
				continue
			}

			// Update the file path in the database
			oldGuid := file.Guid
			file.Path = destPath
			file.Guid = filepath.Clean(destPath)

			// Update the database
			_, err = a.index.db.Exec(
				"UPDATE files SET path = ?, guid = ? WHERE guid = ?",
				file.Path, file.Guid, oldGuid,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error updating database for %s: %v\n", file.Path, err)
			}

			// Update the in-memory index
			delete(a.index.files, oldGuid)
			a.index.files[file.Guid] = file

			movedCount++

		}

	}

	fmt.Printf("Moved %d duplicate files to %s\n", movedCount, path)
}

func (a *App) MoveDuplicateFilesToTrash() {
	// Get OS specific path of trash directory
	trashpath := GetTrashPath()
	// Move duplicate files
	a.MoveDuplicateFilesToDirectory(trashpath)
}

// Delete from duplicate table
func (a *App) IndexForgetDuplicateFiles() {
	a.index.ForgetDuplicates()
}

// Null all hashes in the database file table
func (a *App) IndexForgetHashes() {
	a.index.ForgetHashes()
}

func (a *App) IndexClear() {
	// Todo: delete all from every table

	// Begin transaction
	tx, err := a.index.db.Begin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback() // Rollback if not committed

	// Prepare statement for deleting files
	// Todo: also delete duplicates table?
	stmt, err := tx.Prepare("DELETE FROM files")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to prepare statement: %v\n", err)
		os.Exit(1)
	}
	defer stmt.Close()

	// Execute statement
	result, err := stmt.Exec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to execute statement: %v\n", err)
		os.Exit(1)
	}

	// Get number of affected rows
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get rows affected: %v\n", err)
		os.Exit(1)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to commit transaction: %v\n", err)
		os.Exit(1)
	}

	// Remove from in-memory index
	removedFromMemory := 0
	for guid := range a.index.files {
		delete(a.index.files, guid)
		removedFromMemory++
	}

	fmt.Printf("Removed %d files from database\n", rowsAffected)
}
