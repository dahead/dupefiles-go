package main

import (
	"df/core"
	"flag"
	"fmt"
)

func main() {
	fmt.Println("DupeFiles v0.1.4 - Copyright (c) 2025 dh")

	// flags
	var (
		addPath     = flag.String("add", "", "Add path to database")
		removePath  = flag.String("remove", "", "Remove path from database")
		showConfig  = flag.Bool("config", false, "Show configuration")
		showFiles   = flag.Bool("files", false, "Show all files in database")
		showDupes   = flag.Bool("dupes", false, "Show all duplicate files in database")
		showHashes  = flag.Bool("hashes", false, "Show file hashes in the database")
		scan        = flag.Bool("scan", false, "StartScan for duplicates")
		export      = flag.Bool("export", false, "Export duplicate files to STDOUT")
		exportjson  = flag.String("export-json", "", "Export duplicate files to a filename")
		exportcsv   = flag.String("export-csv", "", "Export duplicate files to a filename")
		clearindex  = flag.Bool("clear", false, "Clear all files in database")
		purgeIndex  = flag.Bool("purgeIndex", false, "Remove non-existing files from database")
		updateIndex = flag.Bool("updateIndex", false, "Updates file hashes in the database")
		quickScan   = flag.String("qs", "", "Add path to database and scan for duplicates (example: ./df --qs /home/user/photos)")
		move        = flag.String("move", "", "Move duplicate files to a new directory")
		trash       = flag.Bool("trash", false, "Move duplicate files to trash")
		forget      = flag.Bool("forget", false, "Remove duplicate files from database")
		headshot    = flag.Bool("headshot", false, "Remove hashes from database")
	)
	flag.Parse()

	// start
	app := core.NewApp()

	switch {
	case *showConfig:
		app.ShowConfig()
	case *showFiles:
		app.ShowFiles()
	case *showDupes:
		app.ShowDupes()
	case *showHashes:
		app.ShowHashes()
	case *scan:
		app.StartScan()
	case *quickScan != "":
		filter := ""
		// do we have a filter in the arguments?
		if flag.NArg() > 0 {
			filter = flag.Arg(0)
		}
		// First add the path to database
		app.AddPathToIndex(*quickScan, true, filter)
		// Then scan for duplicates
		app.StartScan()
	case *addPath != "":
		//  todo: add parsing for recursive flag
		filter := ""
		// do we have a filter in the arguments?
		if flag.NArg() > 0 {
			filter = flag.Arg(0)
		}
		app.AddPathToIndex(*addPath, true, filter)
	case *removePath != "":
		app.RemovePathFromIndex(*removePath)
	case *export:
		app.Export()
	case *exportjson != "":
		app.ExportToJsonFile(*exportjson)
	case *exportcsv != "":
		app.ExportToCSVFile(*exportcsv)
	case *purgeIndex:
		app.PurgeIndex()
	case *updateIndex:
		app.UpdateIndex()
	case *clearindex:
		app.ClearIndex()
	case *forget:
		app.IndexForgetDuplicateFiles()
	case *headshot:
		app.IndexForgetHashes()
	case *move != "":
		app.MoveDuplicateFilesToDirectory(*move)
	case *trash:
		app.MoveDuplicateFilesToTrash()
	default:
		// Default scan behavior
		app.StartScan()
	}
}
