package main

import (
	"flag"
	"fmt"
)

func main() {
	// show app information text in the CLI and copyright info
	fmt.Println("DupeFiles v0.1.0 - Copyright (c) 2025 dh")

	// flags
	var (
		addPath    = flag.String("add", "", "Add path to FileIndex")
		showConfig = flag.Bool("showconfig", false, "Show configuration")
		showFiles  = flag.Bool("showfiles", false, "Show all files in FileIndex")
		scan       = flag.Bool("scan", false, "Scan for duplicates")
		purge      = flag.Bool("purge", false, "Remove non-existing files from FileIndex")
	)
	flag.Parse()

	// start
	app := NewApp()

	switch {
	case *showConfig:
		app.ShowConfig()
	case *showFiles:
		app.ShowFiles()
	case *scan:
		app.Scan()
	case *purge:
		app.Purge()
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
