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

### Add directory and immediately scan it
```bash
./df --qs /path/to/directory
```

### Scan for Duplicates
```bash
# Default behavior - scan for duplicates
./df

# Explicit scan command
./df --scan
```

### Add Files to Index

#### Add a single file
```bash
./df --add /path/to/file.txt
```

#### Add a directory (recursive by default)
```bash
./df --add /path/to/directory
```

#### Add directory with file filter (e.g., only MP4 files)
```bash
./df --add /path/to/videos *.mp4
```

### Index Management

#### Show configuration (index file location, etc.)
```bash
./df --config
```

#### List all files in the index
```bash
./df --files
```

#### List all duplicate files in the index
```bash
./df --dupes
```

#### Show file hashes in the database
```bash
./df --hashes
```

#### Update files in the index
```bash
./df --update
```

#### Remove non-existent files from index
```bash
./df --purge
```

#### Export duplicate files to a text file
```bash
./df --export > duplicates.txt
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

Note: on removable media like USB or SSD drives this currently does not work, because the app is trying to move the external files to the local trash. Instead you have to use --move /external/trash.directory/

#### Remove duplicate files from database
```bash
./df --forget
```

#### Remove hashes from database
```bash
./df --headshot
```

### Debug Mode

#### Enable debug mode
```bash
./df --debug
```
