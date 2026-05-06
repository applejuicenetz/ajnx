package setup

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Lock repräsentiert den Inhalt der setup.lock Datei.
type Lock struct {
	CompletedAt  time.Time `json:"completed_at"`
	Version      string    `json:"version"`
	CoreHost     string    `json:"core_host"`
	CorePort     int       `json:"core_port"`
	CorePassword string    `json:"core_password"`
	Nickname     string    `json:"nickname"`
	SessionSecret string `json:"session_secret"`
	// Plugins enthält die Aktivierungs-Status der einzelnen Plugins (ID -> Enabled)
	Plugins map[string]bool `json:"plugins"`
	// PathMappings enthält Pfad-Übersetzungen (Docker-Pfad -> Host-Pfad)
	PathMappings map[string]string `json:"path_mappings"`
}

// Manager verwaltet den Status des Erst-Setups.
type Manager struct {
	mu       sync.RWMutex
	path     string
	complete bool
	data     *Lock
}

// NewManager erstellt einen neuen Setup-Manager.
func NewManager(path string) (*Manager, error) {
	m := &Manager{path: path}
	if err := m.Load(); err != nil {
		// Datei nicht vorhanden oder keine Leseberechtigung → Setup steht noch aus.
		// Complete() wird beim Schreibversuch einen klaren Fehler liefern falls
		// auch keine Schreibrechte vorhanden sind.
		if os.IsNotExist(err) || os.IsPermission(err) {
			return m, nil
		}
		return nil, err
	}
	return m, nil
}

// Load lädt den Lock-Status von der Festplatte.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		m.complete = false
		return err
	}

	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return err
	}

	m.data = &lock
	m.complete = true
	return nil
}

// IsComplete gibt zurück, ob das Setup abgeschlossen ist.
func (m *Manager) IsComplete() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.complete
}

// Complete schließt das Setup ab und schreibt die Lock-Datei.
// SessionSecret wird automatisch generiert – kein manuelles Setzen nötig.
func (m *Manager) Complete(data Lock) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data.CompletedAt = time.Now()
	data.Version = "1.0"

	// SessionSecret erzeugen, falls noch nicht vorhanden
	if data.SessionSecret == "" {
		secret, err := generateSecret()
		if err != nil {
			return fmt.Errorf("session secret konnte nicht generiert werden: %w", err)
		}
		data.SessionSecret = secret
	}

	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(m.path, jsonData, 0600); err != nil {
		return err
	}

	m.data = &data
	m.complete = true
	return nil
}

// MD5Hex gibt den MD5-Hash von s als Hex-String zurück.
func MD5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// generateSecret erzeugt einen kryptographisch sicheren 32-Byte-Zufallsstring (64 Hex-Zeichen).
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Snapshot gibt eine Kopie der Lock-Daten zurück.
func (m *Manager) Snapshot() Lock {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.data == nil {
		return Lock{}
	}
	return *m.data
}

// IsPluginEnabled gibt zurück, ob ein bestimmtes Plugin aktiviert ist.
func (m *Manager) IsPluginEnabled(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.data == nil || m.data.Plugins == nil {
		return false
	}
	return m.data.Plugins[id]
}

// UpdatePlugins aktualisiert die Plugin-Einstellungen und speichert sie.
func (m *Manager) UpdatePlugins(plugins map[string]bool) error {
	m.mu.Lock()
	if m.data == nil {
		m.mu.Unlock()
		return fmt.Errorf("setup noch nicht abgeschlossen")
	}
	m.data.Plugins = plugins
	data := *m.data
	m.mu.Unlock()

	return m.Complete(data)
}

// UpdatePathMappings aktualisiert die Pfad-Übersetzungen und speichert sie.
func (m *Manager) UpdatePathMappings(mappings map[string]string) error {
	m.mu.Lock()
	if m.data == nil {
		m.mu.Unlock()
		return fmt.Errorf("setup noch nicht abgeschlossen")
	}
	m.data.PathMappings = mappings
	data := *m.data
	m.mu.Unlock()

	return m.Complete(data)
}


