package core

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Debug              bool   // Show debug information. Todo!
	DryRun             bool   // Relevant for moving, trashing files. Set to true, only a simulation will follow. No files will get touched.
	MinFileSize        int64  // Minimum file size in bytes
	DBFilename         string // Database filename
	BinaryCompareBytes int64  // Sample size for binary comparison. If 0 always the whole file gets compared. If > 0 only this amount of bytes get compared. The bytes are picked randomly across the whole file.
}

// NewConfig creates a new configuration with default values and environment variable overrides
func NewConfig() *Config {
	config := &Config{
		Debug:              false,
		DryRun:             false,
		MinFileSize:        1024 * 1000,    // default minimum file size
		DBFilename:         "dupefiles.db", // default database filename
		BinaryCompareBytes: 0,
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

	// Read BinaryCompareBytes
	if envBCS := os.Getenv("DF_BINARY_COMPARE_SIZE"); envBCS != "" {
		if parsed, err := strconv.ParseInt(envBCS, 10, 64); err == nil {
			config.BinaryCompareBytes = parsed
			fmt.Print("Environment variable read.")
		}
	}

	//if config.Debug {
	//	fmt.Println("Configuration loaded from environment variables. Debug is on.")
	//}

	// fmt.Println("Configuration loaded from environment variables.")
	// fmt.Printf("ENV: ", os.Getenv("DF_DEBUG"))

	return config
}

// GetMinFileSize returns the minimum file size threshold
func (c *Config) GetMinFileSize() int64 {
	return c.MinFileSize
}

// GetDBFilename returns the database filename
func (c *Config) GetDBFilename() string {
	return c.DBFilename
}
