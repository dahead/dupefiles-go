package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

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
