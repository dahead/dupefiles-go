package core

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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
