package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const DefaultIndexFilename = "dupefiles.db"

// Config holds application configuration
type Config struct {
	Debug                   bool   // Show debug information
	DryRun                  bool   // Relevant for moving, trashing files. Set to true, only a simulation will follow. No files will get touched.
	MinFileSize             int64  // Minimum file size in bytes
	DBFilename              string // Database filename
	SampleSizeBinaryCompare int    // Sample size for binary comparison. If 0 always the whole file gets compared. If > 0 only this amount of bytes get compared. The bytes are picked randomly across the whole file.
}

// NewConfig creates a new configuration with default values and environment variable overrides
func NewConfig() *Config {
	config := &Config{
		Debug:                   false,
		DryRun:                  false,
		MinFileSize:             1024,                      // default minimum file size
		DBFilename:              GetDefaultIndexFilename(), // default database filename
		SampleSizeBinaryCompare: 0,
	}

	// Read Debug
	if os.Getenv("DF_DEBUG") == "true" {
		config.Debug = true
	}

	// Read DryRun
	if os.Getenv("DF_DRYRUN") == "true" {
		config.DryRun = true
	}

	// Read minimum file size from environment variable
	if envMinSize := os.Getenv("DF_MINSIZE"); envMinSize != "" {
		if parsed, err := strconv.ParseInt(envMinSize, 10, 64); err == nil {
			config.MinFileSize = parsed
		}
	}

	// Read database filename from environment variable
	if envDBFile := os.Getenv("DF_DBFILE"); envDBFile != "" {
		config.DBFilename = envDBFile
	}

	// Read SampleSizeBinaryCompare
	if envBCS := os.Getenv("DF_BINARY_COMPARE_SIZE"); envBCS != "" {
		if parsed, err := strconv.Atoi(envBCS); err == nil {
			config.SampleSizeBinaryCompare = parsed
		}
	}

	if config.Debug {
		fmt.Println("Configuration loaded from environment variables. Debug is on.")
	}

	return config
}

func GetDefaultIndexFilename() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir can't be determined
		return DefaultIndexFilename
	}

	configDir := filepath.Join(homeDir, ".config", "dupefiles")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		// Fallback to current directory if config dir can't be created
		return DefaultIndexFilename
	}

	return filepath.Join(configDir, DefaultIndexFilename)
}
