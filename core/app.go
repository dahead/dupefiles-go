package core

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

func HumanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func GetTrashPath() string {
	var path string

	// Determine OS-specific trash directory
	switch runtime.GOOS {
	case "darwin":
		// macOS
		path = os.Getenv("HOME") + "/.Trash"
	case "linux":
		// Linux (follows FreeDesktop.org trash specification)
		// First try XDG_DATA_HOME
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			path = filepath.Join(xdgDataHome, "Trash")
		} else {
			// Default to ~/.local/share/Trash
			path = filepath.Join(os.Getenv("HOME"), ".local/share/Trash")
		}
	case "windows":
		// Windows
		path = filepath.Join(os.Getenv("USERPROFILE"), "RecycleBin")
	default:
		// Default fallback
		path = filepath.Join(os.Getenv("HOME"), ".Trash")
	}

	// Check if the directory exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: Trash directory %s does not exist\n", path)
		// Try to create the directory
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to create trash directory: %v\n", err)
		} else {
			fmt.Printf("Created trash directory: %s\n", path)
		}
	}

	return path
}

func (a *App) ShowConfig() {
	fmt.Printf("Config:\n")
	fmt.Printf("Debug: %v\n", a.config.Debug)
	fmt.Printf("DryRun: %v\n", a.config.DryRun)
	fmt.Printf("Database file: %s\n", a.index.GetIndexPath())
	fmt.Printf("Minimum file size: %d bytes\n", a.config.GetMinFileSize())
	fmt.Printf("Binary compare byte size: %d bytes\n", a.config.BinaryCompareBytes)
	fmt.Printf("Database filename: %s\n", a.config.GetDBFilename())
	fmt.Printf("System trash directory: %s\n", GetTrashPath())
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
	// show each file path
	for _, file := range files {
		fmt.Println(file.Path)
	}
	// show totals
	fmt.Printf("Hashed files in database: %d total.\n", len(files))
}

func (a *App) Scan() {
	// No files in FileIndex skip
	files := a.index.GetAllFiles()
	if len(files) == 0 {
		fmt.Println("No files in database. Nothing to scan.")
		return
	}

	// Todo: Start a  system timer and measure scan duration

	scanner := NewScanner(a.index)              // Create scanner instance
	results, err := scanner.ScanForDuplicates() // Call method on scanner

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	if len(results) == 0 {
		fmt.Println("No duplicate files found!\n")
	} else {
		fmt.Print()
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
		fmt.Printf("\nSummary: %d duplicate files in %d groups, %s used space\n",
			totalDuplicateFiles, len(results), HumanizeBytes(totalDuplicateSize))
	}
}

// Export the duplicates in a report
func (a *App) Export() {
	files := a.index.GetAllDupes()

	// No files in FileIndex skip
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No duplicate files in database\n")
		os.Exit(1)
	}

	fmt.Printf("# DupeFiles Export - Found %d groups of possible duplicate files\n", len(files))
	fmt.Printf("# Format: [Group Number] [Hash] [File Count] [Total Size]\n")
	fmt.Println("#")

	totalDuplicateSize := int64(0)
	totalFiles := 0

	for i, file := range files {
		totalFiles++
		groupSize := file.Size // Größe des duplizierten Files
		totalDuplicateSize += groupSize

		// Todo: Check printf parameters here. Why "1"?
		fmt.Printf("[Group %d] %v %d %s\n", i+1, file.Hash, 1, HumanizeBytes(groupSize))
		fmt.Printf("- %s (%s)\n", file.Path, file.HumanizedSize)
		fmt.Println() // Empty line between groups
	}

	fmt.Printf("# Summary: %d possible duplicate files in %d groups, %s total used space\n",
		totalFiles, len(files), HumanizeBytes(totalDuplicateSize))
}

func (a *App) ExportToJsonFile(filename string) error {
	files := a.index.GetAllDupes()

	// No files in FileIndex skip
	if len(files) == 0 {
		return fmt.Errorf("no duplicate files in database")
	}

	// Create output filename if not provided
	if filename == "" {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("dupefiles_export_%s.json", timestamp)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	// Group files by hash
	groups := make(map[string]*DuplicateGroup)
	for _, file := range files {
		hash := file.Hash.String
		if _, exists := groups[hash]; !exists {
			groups[hash] = &DuplicateGroup{
				GroupID:   len(groups) + 1,
				Hash:      hash,
				Size:      file.Size,
				HumanSize: HumanizeBytes(file.Size),
				FileCount: 0,
				Files:     []string{},
			}
		}
		groups[hash].FileCount++
		groups[hash].Files = append(groups[hash].Files, file.Path)
	}

	// Convert map to slice for JSON output
	var duplicateGroups []*DuplicateGroup
	for _, group := range groups {
		duplicateGroups = append(duplicateGroups, group)
	}

	// Create JSON output
	jsonData, err := json.MarshalIndent(duplicateGroups, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Write to file
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	fmt.Printf("Exported %d duplicate groups to %s\n", len(duplicateGroups), filename)
	return nil
}

func (a *App) ExportToCSVFile(filename string) error {
	files := a.index.GetAllDupes()

	// No files in FileIndex skip
	if len(files) == 0 {
		return fmt.Errorf("no duplicate files in database")
	}

	// Create output filename if not provided
	if filename == "" {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("dupefiles_export_%s.csv", timestamp)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	// Group files by hash
	groups := make(map[string]*DuplicateGroup)
	for _, file := range files {
		hash := file.Hash.String
		if _, exists := groups[hash]; !exists {
			groups[hash] = &DuplicateGroup{
				GroupID:   len(groups) + 1,
				Hash:      hash,
				Size:      file.Size,
				HumanSize: HumanizeBytes(file.Size),
				FileCount: 0,
				Files:     []string{},
			}
		}
		groups[hash].FileCount++
		groups[hash].Files = append(groups[hash].Files, file.Path)
	}

	// Create CSV file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Group ID", "Hash", "Size (bytes)", "Human Size", "File Count", "File Path"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %v", err)
	}

	// Write data
	for _, group := range groups {
		for _, filePath := range group.Files {
			record := []string{
				fmt.Sprintf("%d", group.GroupID),
				group.Hash,
				fmt.Sprintf("%d", group.Size),
				group.HumanSize,
				fmt.Sprintf("%d", group.FileCount),
				filePath,
			}
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("failed to write CSV record: %v", err)
			}
		}
	}

	fmt.Printf("Exported %d duplicate groups to %s\n", len(groups), filename)
	return nil
}

func (a *App) PurgeIndex() {
	count, err := a.index.Purge()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Purged %d files from the database\n", count)
}

func (a *App) UpdateIndex() {
	count, err := a.index.Update()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Updated %d files in the database\n", count)
}

func (a *App) AddPath(path string, recursive bool, filter string) {
	if path == "" {
		fmt.Fprintf(os.Stderr, "Error: No path specified\n")
		os.Exit(1)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// remember  current amount of indexed files
	currentCount := len(a.index.GetAllFiles())

	// add directory or file
	if fileInfo.IsDir() {
		err = a.index.AddDirectory(path, recursive, filter)
	} else {
		err = a.index.AddFile(path)
		if err == nil {
			fmt.Printf("Added: %s\n", path)
		}
	}

	// remember new amount of indexed files
	newCount := len(a.index.GetAllFiles())
	// display changed files
	fmt.Printf("Updated %d files\n", newCount-currentCount)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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

func (a *App) ClearIndex() {
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
