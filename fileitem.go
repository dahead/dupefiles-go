package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
)

type FileItem struct {
	Guid      string
	Path      string
	Extension string
	Size      int64
	Hash      sql.NullString
}

func generateGUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func NewFileItem(path, extension string, size int64) FileItem {
	return FileItem{
		Guid:      generateGUID(),
		Path:      path,
		Extension: extension,
		Size:      size,
		Hash:      sql.NullString{String: "", Valid: false},
	}
}
