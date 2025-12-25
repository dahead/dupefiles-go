package core

import (
	"crypto/rand"
	"database/sql"
	"fmt"
)

type FileItem struct {
	Guid          string
	Path          string
	Extension     string
	Size          int64
	ModTime       int64 // Added: Unix timestamp of modification
	Hash          sql.NullString
	HumanizedSize string // Added: Human-readable size string
}

type DuplicateGroup struct {
	GroupID   int      // Eindeutige ID der Gruppe
	Hash      string   // Hash-Wert der Datei
	Size      int64    // Größe der Datei in Byte
	HumanSize string   // Größe der Datei in menschenlesbarer Form (z.B. "10 MB")
	FileCount int      // Anzahl der Dateien in der Gruppe
	Files     []string // Liste der Dateipfade in der Gruppe
}

func generateGUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback or panic, for simplicity, let's panic if random generation fails
		panic(fmt.Sprintf("failed to generate GUID: %v", err))
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
