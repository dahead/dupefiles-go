package core

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	MinFileSize        int64  // Minimum file size in bytes
	DBFilename         string // Database filename
	BinaryCompareBytes int
}

// NewConfig creates a new configuration with default values and environment variable overrides
func NewConfig() *Config {
	config := &Config{
		MinFileSize: 1024 * 1000,    // default minimum file size
		DBFilename:  "dupefiles.db", // default database filename
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
