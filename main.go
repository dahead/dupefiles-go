package main

import (
	"df/core"
	"flag"
	"fmt"
)

func main() {
	// show app information text in the CLI and copyright info
	fmt.Println("DupeFiles v0.1.3 - Copyright (c) 2025 dh")

	// flags
	var (
		debug      = flag.Bool("debug", false, "Enable debug mode")
		addPath    = flag.String("add", "path", "Add path to database (example: --add /home/user/documents)")
		showConfig = flag.Bool("config", false, "Show configuration")
		showFiles  = flag.Bool("files", false, "Show all files in database")
		showDupes  = flag.Bool("dupes", false, "Show all duplicate files in database")
		showHashes = flag.Bool("hashes", false, "Show file hashes in the database")
		scan       = flag.Bool("scan", false, "Scan for duplicates")
		export     = flag.Bool("export", false, "Export duplicate files to STDOUT (example: --export > duplicates.txt)")
		purge      = flag.Bool("purge", false, "Remove non-existing files from database")
		update     = flag.Bool("update", false, "Updates file hashes in the database")
		quickScan  = flag.String("qs", "path", "Add path to database and scan for duplicates (example: --quickscan /home/user/photos)")
		move       = flag.String("move", "path", "Move duplicate files to a new directory")
		trash      = flag.Bool("trash", false, "Move duplicate files to trash")
		forget     = flag.Bool("forget", false, "Remove duplicate files from database")
		headshot   = flag.Bool("headshot", false, "Remove hashes from database")
	)
	flag.Parse()

	// start
	app := core.NewApp()

	switch {
	case *debug:
		app.Debug = true
	case *showConfig:
		app.ShowConfig()
	case *showFiles:
		app.ShowFiles()
	case *showDupes:
		app.ShowDupes()
	case *showHashes:
		app.ShowHashes()
	case *scan:
		app.Scan()
	case *export:
		app.Export()
	case *purge:
		app.PurgeIndex()
	case *update:
		app.UpdateIndex()
	case *trash:
		app.MoveDuplicatesToTrash()
	case *forget:
		app.DatabaseForgetDuplicates()
	case *headshot:
		app.DatabaseForgetDuplicates()
		app.DatabaseForgetHashes()
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
	case *move != "":
		app.MoveDuplicates(*move)
	default:
		// Default scan behavior
		app.Scan()
	}
}
