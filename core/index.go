package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Index struct {
	db     *sql.DB
	files  map[string]*FileItem // Map of Guid to FileItem
	config *Config
}

func NewIndex(config *Config) (*Index, error) {
	dbFileName := config.DBFilename

	_, err := os.Stat(dbFileName)
	dbExists := !os.IsNotExist(err)

	db, err := sql.Open("sqlite3", dbFileName+"?_journal_mode=WAL&_busy_timeout=5000") // Added WAL and busy_timeout
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if !dbExists {

		// create files table
		_, err = db.Exec(`
			CREATE TABLE files (
				guid TEXT PRIMARY KEY,
				path TEXT NOT NULL,
				extension TEXT NOT NULL,
				size INTEGER NOT NULL,
				mod_time INTEGER NOT NULL, -- Added
				hash TEXT,
				humanized_size TEXT
			)
		`)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create tables: %v", err)
		}

		// Create duplicates table
		_, err = db.Exec(`
			CREATE TABLE duplicates (
				guid TEXT PRIMARY KEY,
				scanned INTEGER NOT NULL,
				FOREIGN KEY (guid) REFERENCES files(guid)
			)
		`)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create duplicates table: %v", err)
		}

		_, err = db.Exec(`CREATE INDEX idx_files_path ON files (path)`) // Index path for faster lookups if needed
		if err != nil {
			// Log error but don't fail creation
			fmt.Fprintf(os.Stderr, "Warning: failed to create path index: %v\n", err)
		}
		_, err = db.Exec(`CREATE INDEX idx_files_size ON files (size)`) // Index size for faster grouping
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create size index: %v\n", err)
		}
		_, err = db.Exec(`CREATE INDEX idx_files_hash ON files (hash)`) // Index hash for faster grouping
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create hash index: %v\n", err)
		}
	}

	index := &Index{
		db:     db,
		files:  make(map[string]*FileItem),
		config: config,
	}

	if dbExists {
		err = index.loadFilesFromDB()
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to load database: %v", err)
		}
	}

	return index, nil
}

func (idx *Index) GetIndexPath() string {
	absPath, _ := filepath.Abs(idx.config.DBFilename)
	return absPath
}

func (idx *Index) GetAllFiles() []*FileItem {
	files := make([]*FileItem, 0, len(idx.files))
	for _, file := range idx.files {
		files = append(files, file)
	}
	return files
}

func (idx *Index) GetAllDupes() []*FileItem {
	query := `
		SELECT f.guid, f.path, f.extension, f.size, f.mod_time, f.hash, f.humanized_size 
		FROM files f
		INNER JOIN duplicates d ON f.guid = d.guid
		ORDER BY f.size DESC, f.hash
	`

	rows, err := idx.db.Query(query)
	if err != nil {
		fmt.Printf("Warning: Failed to query duplicate files: %v\n", err)
		return []*FileItem{}
	}
	defer rows.Close()

	var duplicateFiles []*FileItem
	for rows.Next() {
		var file FileItem
		var hash sql.NullString
		err := rows.Scan(&file.Guid, &file.Path, &file.Extension, &file.Size, &file.ModTime, &hash, &file.HumanizedSize)
		if err != nil {
			fmt.Printf("Warning: Failed to scan duplicate file row: %v\n", err)
			continue
		}
		file.Hash = hash
		duplicateFiles = append(duplicateFiles, &file)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Warning: Error iterating duplicate files: %v\n", err)
	}

	return duplicateFiles
}

func (idx *Index) GetRestOfDuplicates() []*FileItem {
	// get all duplicates except the first one of each size+hash group
	query := `
		SELECT f.guid, f.path, f.extension, f.size, f.mod_time, f.hash, f.humanized_size 
		FROM files f
		INNER JOIN duplicates d ON f.guid = d.guid
		WHERE f.guid NOT IN (
			SELECT MIN(f2.guid)
			FROM files f2
			INNER JOIN duplicates d2 ON f2.guid = d2.guid
			GROUP BY f2.size, f2.hash
		)
		ORDER BY f.size DESC, f.hash
	`

	rows, err := idx.db.Query(query)
	if err != nil {
		fmt.Printf("Warning: Failed to query duplicate files: %v\n", err)
		return []*FileItem{}
	}
	defer rows.Close()

	var duplicateFiles []*FileItem
	for rows.Next() {
		var file FileItem
		var hash sql.NullString
		err := rows.Scan(&file.Guid, &file.Path, &file.Extension, &file.Size, &file.ModTime, &hash, &file.HumanizedSize)
		if err != nil {
			fmt.Printf("Warning: Failed to scan duplicate file row: %v\n", err)
			continue
		}
		file.Hash = hash
		duplicateFiles = append(duplicateFiles, &file)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Warning: Error iterating duplicate files: %v\n", err)
	}

	return duplicateFiles
}

// Get all files that have hash values
func (idx *Index) GetAllHashedFiles() []*FileItem {
	query := `
		SELECT f.guid, f.path, f.extension, f.size, f.mod_time, f.hash, f.humanized_size 
		FROM files f
		WHERE f.hash IS NOT NULL
		ORDER BY f.size DESC, f.hash
	`

	rows, err := idx.db.Query(query)
	if err != nil {
		fmt.Printf("Warning: Failed to query hashed files: %v\n", err)
		return []*FileItem{}
	}
	defer rows.Close()

	var resultFiles []*FileItem
	for rows.Next() {
		var file FileItem
		var hash sql.NullString
		err := rows.Scan(&file.Guid, &file.Path, &file.Extension, &file.Size, &file.ModTime, &hash, &file.HumanizedSize)
		if err != nil {
			fmt.Printf("Warning: Failed to scan file row: %v\n", err)
			continue
		}
		file.Hash = hash
		resultFiles = append(resultFiles, &file)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Warning: Error iterating files: %v\n", err)
	}

	return resultFiles
}

func (idx *Index) GetFileByGuid(guid string) *FileItem {
	return idx.files[guid]
}

func (idx *Index) loadFilesFromDB() error {
	rows, err := idx.db.Query("SELECT guid, path, extension, size, mod_time, hash, humanized_size FROM files")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var file FileItem
		var hash sql.NullString
		err := rows.Scan(&file.Guid, &file.Path, &file.Extension, &file.Size, &file.ModTime, &hash, &file.HumanizedSize)
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

func (idx *Index) Clear() error {
	_, err := idx.db.Exec("DELETE FROM files")
	if err != nil {
		return err
	}
	_, err = idx.db.Exec("DELETE FROM duplicates")
	if err != nil {
		return err
	}
	idx.files = make(map[string]*FileItem)
	return nil
}

func (idx *Index) AddFileItems(fileItems []*FileItem) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO files (guid, path, extension, size, mod_time, hash, humanized_size) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, file := range fileItems {
		idx.files[file.Guid] = file
		stmt.Exec(file.Guid, file.Path, file.Extension, file.Size, file.ModTime, file.Hash, file.HumanizedSize)
		if idx.config.Debug {
			fmt.Printf("Debug: Adding %s to index\n", file.Guid)
		}
	}

	return tx.Commit()
}

func (idx *Index) RemoveFile(guid string) error {
	delete(idx.files, guid)
	_, err := idx.db.Exec("DELETE FROM files WHERE guid = ?", guid)
	return err
}

func (idx *Index) Purge() (int, error) {
	count := 0
	guidsToDelete := []string{}

	// Create a copy of guids to iterate over safely while potentially deleting from map
	// though we collect them first anyway.
	for guid, file := range idx.files {
		_, err := os.Stat(file.Path)
		if os.IsNotExist(err) {
			guidsToDelete = append(guidsToDelete, guid)
		}
	}

	if len(guidsToDelete) == 0 {
		return 0, nil
	}

	// Begin transaction
	tx, err := idx.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction for purge: %v", err)
	}
	defer tx.Rollback()

	// Prepare statements
	stmtFiles, err := tx.Prepare("DELETE FROM files WHERE guid = ?")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare delete statement for files: %v", err)
	}
	defer stmtFiles.Close()

	stmtDupes, err := tx.Prepare("DELETE FROM duplicates WHERE guid = ?")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare delete statement for duplicates: %v", err)
	}
	defer stmtDupes.Close()

	for _, guid := range guidsToDelete {
		// 1. Remove from database files table
		_, errExec := stmtFiles.Exec(guid)
		if errExec != nil {
			fmt.Fprintf(os.Stderr, "Error deleting file %s from files table: %v\n", guid, errExec)
			continue
		}

		// 2. Remove from database duplicates table
		_, errExec = stmtDupes.Exec(guid)
		if errExec != nil {
			fmt.Fprintf(os.Stderr, "Error deleting file %s from duplicates table: %v\n", guid, errExec)
			// Continue even if this fails
		}

		// 3. Remove from in-memory index
		delete(idx.files, guid)
		count++
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("failed to commit transaction for purge: %v", err)
	}

	return count, nil
}

func (idx *Index) Update() (int, error) {
	count := 0
	filesToUpdateInDB := []*FileItem{}
	guidsToDelete := []string{}

	for _, file := range idx.files {
		fileInfo, err := os.Stat(file.Path)
		if os.IsNotExist(err) {
			guidsToDelete = append(guidsToDelete, file.Guid)
			continue
		}
		if err != nil {
			fmt.Printf("Warning: Error accessing %s during update: %v\n", file.Path, err)
			continue
		}

		newModTime := fileInfo.ModTime().Unix()
		if fileInfo.Size() != file.Size || newModTime != file.ModTime {
			file.Size = fileInfo.Size()
			file.ModTime = newModTime

			// Invalidate old hash and recalculate
			// The CalculateHash method now only returns hash string and error
			newHashString, errHash := CalculateFileHash(file.Path, file.Size)

			if errHash != nil {
				fmt.Printf("Warning: Failed to calculate hash for updated file %s: %v\n", file.Path, errHash)
				file.Hash = sql.NullString{String: "", Valid: false}
			} else {
				file.Hash = sql.NullString{String: newHashString, Valid: true}
			}
			filesToUpdateInDB = append(filesToUpdateInDB, file)
			count++
			fmt.Printf("Marked for update: %s (new size: %d, new mod_time: %d)\n", file.Path, file.Size, file.ModTime)
		}
	}

	// Perform deletions
	if len(guidsToDelete) > 0 {
		txDel, err := idx.db.Begin()
		if err != nil {
			return count, fmt.Errorf("update: failed to begin delete transaction: %v", err)
		}
		stmtDel, err := txDel.Prepare("DELETE FROM files WHERE guid = ?")
		if err != nil {
			txDel.Rollback()
			return count, fmt.Errorf("update: failed to prepare delete statement: %v", err)
		}
		for _, guid := range guidsToDelete {
			delete(idx.files, guid)
			if _, errExec := stmtDel.Exec(guid); errExec != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete %s during update: %v\n", guid, errExec)
			}
		}
		stmtDel.Close()
		if errCommit := txDel.Commit(); errCommit != nil {
			return count, fmt.Errorf("update: failed to commit delete transaction: %v", errCommit)
		}
		count += len(guidsToDelete) // Count towards modified items
	}

	// Perform updates
	if len(filesToUpdateInDB) > 0 {
		txUpd, err := idx.db.Begin()
		if err != nil {
			return count, fmt.Errorf("update: failed to begin update transaction: %v", err)
		}
		stmtUpd, err := txUpd.Prepare("UPDATE files SET size = ?, hash = ?, mod_time = ? WHERE guid = ?")
		if err != nil {
			txUpd.Rollback()
			return count, fmt.Errorf("update: failed to prepare update statement: %v", err)
		}
		for _, fileToUpdate := range filesToUpdateInDB {
			if _, errExec := stmtUpd.Exec(fileToUpdate.Size, fileToUpdate.Hash, fileToUpdate.ModTime, fileToUpdate.Guid); errExec != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to update %s during update: %v\n", fileToUpdate.Guid, errExec)
			}
		}
		stmtUpd.Close()
		if errCommit := txUpd.Commit(); errCommit != nil {
			return count, fmt.Errorf("update: failed to commit update transaction: %v", errCommit)
		}
	}
	return count, nil
}

// Delete all known duplicates
func (idx *Index) ForgetDuplicates() error {
	result, err := idx.db.Exec(
		// 			stmt, err := tx.Prepare(`
		//               INSERT INTO duplicates (guid, scanned)
		//               VALUES (?, ?)
		//               ON CONFLICT(guid) DO UPDATE SET scanned = ?
		//           `)
		"DELETE from duplicates",
	)
	if err != nil {
		return fmt.Errorf("failed to forget duplicates: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	fmt.Printf("Removed %d duplicate files from database\n", rowsAffected)
	return nil
}

// Delete all calculated hash values
func (idx *Index) ForgetHashes() error {
	result, err := idx.db.Exec(
		"UPDATE files SET hash = NULL WHERE hash IS NOT NULL",
	)
	if err != nil {
		return fmt.Errorf("failed to forget hashes: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	fmt.Printf("Cleared hashes for %d files in database\n", rowsAffected)
	return nil
}
