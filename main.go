package main

import (
	"flag"
	"fmt"
)

func main() {
	// show app information text in the CLI and copyright info
	fmt.Println("DupeFiles v0.1.1 - Copyright (c) 2025 dh")

	// Load configuration
	config := NewConfig()

	// Show configuration info if minimum file size is set
	if config.GetMinFileSize() > 0 {
		fmt.Printf("Using minimum file size threshold: %d bytes\n", config.GetMinFileSize())
	}
	if config.GetDBFilename() != "dupefiles.db" {
		fmt.Printf("Using database file: %s\n", config.GetDBFilename())
	}

	// flags
	var (
		addPath        = flag.String("add", "", "Add path to database (example: --add /home/user/documents)")
		showConfig     = flag.Bool("showconfig", false, "Show configuration")
		showFiles      = flag.Bool("showfiles", false, "Show all files in database")
		showDuplicates = flag.Bool("showduplicates", false, "Show all duplicate files in database")
		scan           = flag.Bool("scan", false, "Scan for duplicates")
		export         = flag.Bool("export", false, "Export duplicate files to STDOUT (example: --export > duplicates.txt)")
		purge          = flag.Bool("purge", false, "Remove non-existing files from database")
		update         = flag.Bool("update", false, "Updates file hashes in the database")
		quickScan      = flag.String("quickscan", "", "Add path to database and scan for duplicates (example: --quickscan /home/user/photos)")
	)
	flag.Parse()

	// start
	app := NewApp()

	switch {
	case *showConfig:
		app.ShowConfig()
	case *showFiles:
		app.ShowFiles()
	case *showDuplicates:
		app.ShowDuplicates()
	case *scan:
		app.Scan()
	case *export:
		app.Export()
	case *purge:
		app.Purge()
	case *update:
		app.Update()
	case *quickScan != "":
		filter := ""
		if flag.NArg() > 0 {
			filter = flag.Arg(0)
		}
		// First add the path to database
		app.AddPath(*quickScan, true, filter)
		// Then scan for duplicates
		app.Scan()
	case *addPath != "":
		filter := ""
		if flag.NArg() > 0 {
			filter = flag.Arg(0)
		}
		app.AddPath(*addPath, true, filter)
	default:
		// Default scan behavior
		app.Scan()
	}
}
