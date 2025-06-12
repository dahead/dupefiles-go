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
go build -o dupefiles
```

## Commands

### Scan for Duplicates
```bash
# Default behavior - scan for duplicates
./dupefiles

# Explicit scan command
./dupefiles --scan
```

### Add Files to Index

#### Add a single file
```bash
./dupefiles --add /path/to/file.txt
```

#### Add a directory (recursive by default)
```bash
./dupefiles --add /path/to/directory
```

#### Add directory with file filter (e.g., only MP4 files)
```bash
./dupefiles --add /path/to/videos *.mp4
```

### Index Management

#### Show configuration (index file location, etc.)
```bash
./dupefiles --showconfig
```

#### List all files in the index
```bash
./dupefiles --showfiles
```

#### Remove non-existent files from index
```bash
./dupefiles --purge
```