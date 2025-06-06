package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFileName  = "FileIndex.db"
	minFileSize = int64(1024) // Set to 0 to include all files, or e.g. 1024 for files >= 1KB
)

type Index struct {
	db    *sql.DB
	files map[string]*FileItem // Map of Guid to FileItem
}

func NewIndex() (*Index, error) {
	// Check if database file exists
	_, err := os.Stat(dbFileName)
	dbExists := !os.IsNotExist(err)

	// Open database connection
	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create tables if they don't exist
	if !dbExists {
		_, err = db.Exec(`
			CREATE TABLE files (
				guid TEXT PRIMARY KEY,
				path TEXT NOT NULL,
				extension TEXT NOT NULL,
				size INTEGER NOT NULL,
				hash TEXT
			)
		`)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create tables: %v", err)
		}
	}

	index := &Index{
		db:    db,
		files: make(map[string]*FileItem),
	}

	// Load existing files from database
	if dbExists {
		err = index.loadFromDB()
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to load FileIndex from database: %v", err)
		}
	}

	return index, nil
}

func (idx *Index) GetIndexPath() string {
	absPath, _ := filepath.Abs(dbFileName)
	return absPath
}

func (idx *Index) GetAllFiles() []*FileItem {
	files := make([]*FileItem, 0, len(idx.files))
	for _, file := range idx.files {
		files = append(files, file)
	}
	return files
}

func (idx *Index) GetFileByGuid(guid string) *FileItem {
	return idx.files[guid]
}

func (idx *Index) loadFromDB() error {
	rows, err := idx.db.Query("SELECT guid, path, extension, size, hash FROM files")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var file FileItem
		var hash sql.NullString
		err := rows.Scan(&file.Guid, &file.Path, &file.Extension, &file.Size, &hash)
		if err != nil {
			return err
		}
		file.Hash = hash
		idx.files[file.Guid] = &file
	}

	return rows.Err()
}

func (idx *Index) Close() error {
	return idx.db.Close()
}

func (idx *Index) AddFile(path string) error {
	// Check if file exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Skip directories
	if fileInfo.IsDir() {
		return nil
	}

	// Skip if minimal file size to add is configured
	if minFileSize > 0 && fileInfo.Size() < minFileSize {
		fmt.Printf("Skipping %s (size: %d)\n", path, fileInfo.Size())
		return nil
	}

	// Generate a unique ID for the file
	guid := filepath.Clean(path) // Using the clean path as the GUID for simplicity

	// Get file extension
	extension := strings.TrimPrefix(filepath.Ext(path), ".")

	// Create file item
	file := &FileItem{
		Guid:      guid,
		Path:      path,
		Extension: extension,
		Size:      fileInfo.Size(),
		Hash:      sql.NullString{String: "", Valid: false},
	}

	// Add to in-memory FileIndex
	idx.files[guid] = file

	// Add to database
	_, err = idx.db.Exec(
		"INSERT OR REPLACE INTO files (guid, path, extension, size, hash) VALUES (?, ?, ?, ?, ?)",
		file.Guid, file.Path, file.Extension, file.Size, file.Hash,
	)
	if err != nil {
		return err
	}

	return nil
}

func (idx *Index) AddDirectory(dirPath string, recursive bool, filter string) error {
	// Check if directory exists
	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", dirPath)
	}

	// Walk through directory
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Warning: Error accessing %s: %v\n", path, err)
			return nil
		}

		// Skip directories
		if info.IsDir() {
			// If not recursive, skip subdirectories
			if !recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches filter
		if filter != "" {
			matched, err := filepath.Match(filter, filepath.Base(path))
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}

		// Add file to FileIndex
		err = idx.AddFile(path)
		if err != nil {
			fmt.Printf("Warning: Failed to add %s: %v\n", path, err)
		} else {
			fmt.Printf("Added/Updated: %s\n", path)
		}

		return nil
	}

	return filepath.Walk(dirPath, walkFunc)
}

func (idx *Index) Purge() (int, error) {
	count := 0
	for guid, file := range idx.files {
		_, err := os.Stat(file.Path)
		if os.IsNotExist(err) {
			delete(idx.files, guid)
			_, err = idx.db.Exec("DELETE FROM files WHERE guid = ?", guid)
			if err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}
