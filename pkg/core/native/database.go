package native

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS servers (
    id TEXT PRIMARY KEY,
    name TEXT,
    host TEXT,
    port INTEGER,
    lastseen INTEGER,
    connectiontry INTEGER,
    connected BOOLEAN
);

CREATE TABLE IF NOT EXISTS shares (
    filename TEXT PRIMARY KEY,
    size INTEGER,
    priority INTEGER,
    lastasked INTEGER,
    askcount INTEGER,
    searchcount INTEGER
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE IF NOT EXISTS shared_directories (
    path TEXT PRIMARY KEY,
    with_subfolders BOOLEAN
);
`

func (c *NativeCore) initDB() error {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/data/applejuice.db"
	}

	// Sicherstellen, dass das Verzeichnis existiert
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("konnte Datenbankverzeichnis nicht erstellen: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("konnte Datenbank nicht öffnen: %v", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return fmt.Errorf("konnte Schema nicht erstellen: %v", err)
	}

	c.db = db
	
	// Datei-Berechtigungen absichern (nur Besitzer darf lesen/schreiben)
	if err := os.Chmod(dbPath, 0600); err != nil {
		log.Printf("NativeCore: Warnung - konnte Berechtigungen für %s nicht anpassen: %v", dbPath, err)
	}

	log.Printf("NativeCore: Datenbank initialisiert und abgesichert unter %s", dbPath)
	return nil
}

func (c *NativeCore) closeDB() {
	if c.db != nil {
		c.db.Close()
	}
}

func (c *NativeCore) getSetting(key string) string {
	var val string
	err := c.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		return ""
	}
	return val
}

func (c *NativeCore) setSetting(key string, val interface{}) {
	_, _ = c.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, fmt.Sprintf("%v", val))
}
