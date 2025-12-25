# DupeFiles

A command-line application for finding duplicate files on your system using an indexed approach for efficient scanning.

## Overview

DupeFiles (short: `df`) is a CLI tool that helps you identify duplicate files by maintaining an indexed database of scanned files. The application uses a multi-step verification process to ensure accurate duplicate detection:

1. **Size comparison** - Files with different sizes cannot be duplicates
2. **Hash calculation** - MD5 for files < 2GB, SHA-256 for larger files
3. **Binary comparison** - Byte-by-byte verification for final confirmation

## Features

- **Indexed scanning** - Maintains a local SQLite database (`index.db`) for fast subsequent scans
- **Smart hashing** - Uses appropriate hash algorithms based on file size
- **Binary verification** - Ensures 100% accuracy with byte-by-byte comparison
- **Flexible file addition** - Add individual files or entire directories
- **File filtering** - Support for file extension filters
- **Index maintenance** - Remove non-existent files from the index

## Installation

Ensure you have Go 1.24+ installed, then build the application:

``` bash
go build -o df
```

## Commands

### Interactive Modes

#### TUI Mode (Terminal User Interface)
Starts an interactive terminal interface to manage files and duplicates. This is the default mode if no flags are provided.
```bash
./df --tui
# or simply
./df
```

#### Webserver Mode
Starts a web interface on the specified port (default 8080).
```bash
./df --webserver
./df --webserver --port 9000
```

### Quick Start
Add a directory and immediately scan it:
```bash
./df --qs /path/to/directory
```

### Scan for Duplicates
```bash
# Start scanning files already in the database
./df --scan
```

### Index Management

#### Add Files to Index
```bash
# Add a single file
./df --add /path/to/file.txt

# Add a directory (recursive)
./df --add /path/to/directory

# Add directory with file filter (e.g., only MP4 files)
./df --add /path/to/videos *.mp4
```

#### Remove Files from Index
```bash
./df --remove /path/to/directory
```

#### Maintenance
```bash
# Show configuration (index file location, etc.)
./df --config

# List all files in the index
./df --files

# List all duplicate files in the index
./df --dupes

# Show file hashes in the database
./df --hashes

# Update file hashes in the index
./df --updateIndex

# Remove non-existent files from index
./df --purgeIndex

# Clear all files from the database
./df --clear
```

### Exporting Results

#### Export to STDOUT
```bash
./df --export > duplicates.txt
```

#### Export to File (JSON or CSV)
```bash
./df --export-json results.json
./df --export-csv results.csv
```

### Duplicate File Management

#### Move duplicate files to a new directory
```bash
./df --move /path/to/destination
```

#### Move duplicate files to trash
```bash
./df --trash
```

Note: on removable media like USB or SSD drives this currently does not work, because the app is trying to move the external files to the local trash. Instead you have to use `--move /external/trash.directory/`.

#### Remove duplicate files from database
```bash
./df --forget
```

#### Remove hashes from database
```bash
./df --headshot
```

#### Enable debug mode
```bash
./df --debug
```

## License

Copyright (c) 2025 dh
