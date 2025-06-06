package main

import (
	"fmt"
	"os"
)

type App struct {
	index *Index
}

func NewApp() *App {
	idx, err := NewIndex()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating FileIndex: %v\n", err)
		os.Exit(1)
	}

	return &App{
		index: idx,
	}
}

func (a *App) Close() {
	if a.index != nil {
		a.index.Close()
	}
}

func (a *App) ShowConfig() {
	fmt.Printf("Index file: %s\n", a.index.GetIndexPath())
}

func (a *App) ShowFiles() {
	files := a.index.GetAllFiles()
	if len(files) == 0 {
		fmt.Println("No files in FileIndex")
		return
	}
	// show each file path
	for _, file := range files {
		fmt.Println(file.Path)
	}
	// show totals
	fmt.Printf("Files in FileIndex (%d total):\n", len(files))
}

func (a *App) Scan() {

	// No files in FileIndex skip
	files := a.index.GetAllFiles()
	if len(files) == 0 {
		fmt.Println("No files in FileIndex")
		return
	}

	results, err := a.index.ScanForDuplicates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	if len(results) == 0 {
		fmt.Println("No duplicate files found")
	} else {
		fmt.Printf("Found %d groups of duplicate files:\n", len(results))
		for i, result := range results {
			fmt.Printf("\nGroup %d (Hash: %s):\n", i+1, result.HashSum[:8]+"...")
			for _, guid := range result.FileGuids {
				file := a.index.GetFileByGuid(guid)
				if file != nil {
					fmt.Printf("  %s (%d bytes)\n", file.Path, file.Size)
				}
			}
		}
	}
}

func (a *App) Purge() {
	count, err := a.index.Purge()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Purged %d non-existent files from the FileIndex\n", count)
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
	fmt.Printf("Added/Updated %d files\n", newCount-currentCount)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
