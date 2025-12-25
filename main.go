package main

import (
	"df/core"
	"df/tui"
	"df/web"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var sDupeFilesInfo = "DupeFiles v0.2.0 - Copyright (c) 2025 dh"

func main() {
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
		tuiMode     = flag.Bool("tui", false, "Start TUI mode")
		webMode     = flag.Bool("webserver", false, "Start Webserver mode")
		webPort     = flag.Int("port", 8080, "Webserver port")
	)
	flag.Parse()

	// start
	app := core.NewApp()
	defer app.Close()

	// If no flags are provided, or -tui is set, start TUI
	if *tuiMode {
		m := tui.NewModel(app)
		p := tea.NewProgram(m, tea.WithAltScreen())
		app.SetProgram(p)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *webMode {
		ws := web.NewWebServer(app, *webPort)
		if err := ws.Start(); err != nil {
			fmt.Printf("Webserver error: %v", err)
			os.Exit(1)
		}
		return
	}

	if flag.NFlag() == 0 {
		m := tui.NewModel(app)
		p := tea.NewProgram(m, tea.WithAltScreen())
		app.SetProgram(p)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
		return
	}

	fmt.Println(sDupeFilesInfo)

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
		count, err := app.AddPathToIndex(*quickScan, true, filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Updated %d files\n", count)
		// Then scan for duplicates
		app.StartScan()
	case *addPath != "":
		//  todo: add parsing for recursive flag
		filter := ""
		// do we have a filter in the arguments?
		if flag.NArg() > 0 {
			filter = flag.Arg(0)
		}
		count, err := app.AddPathToIndex(*addPath, true, filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Updated %d files\n", count)
	case *removePath != "":
		count, err := app.RemovePathFromIndex(*removePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed %d files from database\n", count)
	case *export:
		app.Export()
	case *exportjson != "":
		app.ExportToJsonFile(*exportjson)
	case *exportcsv != "":
		app.ExportToCSVFile(*exportcsv)
	case *purgeIndex:
		app.IndexPurge()
	case *updateIndex:
		app.IndexUpdate()
	case *clearindex:
		app.IndexClear()
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
