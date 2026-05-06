package native

import (
	"embed"
	"fmt"
	"log"

	"github.com/applejuicenetz/ajnx/pkg/core"
	"github.com/applejuicenetz/ajnx/pkg/domain"
)

//go:embed fixtures/*.xml
var fixtureFS embed.FS

// Lokale Strukturen für das Fixture-Format (weicht von der Live-API ab!)
type fixtureServerList struct {
	Servers []fixtureServer `xml:"server"`
}
type fixtureServer struct {
	Name          string `xml:"name,attr"`
	Host          string `xml:"host,attr"`
	Port          int    `xml:"port,attr"`
	LastSeen      int64  `xml:"lastseen,attr"`
	ConnectionTry int    `xml:"connectiontry,attr"`
}

type fixtureShareList struct {
	Files []fixtureFile `xml:"file"`
}
type fixtureFile struct {
	Name        string `xml:"name,attr"`
	Priority    int    `xml:"priority,attr"`
	LastAsked   int64  `xml:"lastasked,attr"`
	AskCount    int    `xml:"askcount,attr"`
	SearchCount int    `xml:"searchcount,attr"`
}

// LoadFromFixtures lädt den initialen Status aus den eingebetteten XML-Dateien.
func (c *NativeCore) LoadFromFixtures() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Prüfen, ob bereits Daten in der DB sind (Seeding nur beim ersten Mal)
	var count int
	if err := c.db.QueryRow("SELECT COUNT(*) FROM servers").Scan(&count); err == nil && count > 0 {
		log.Println("NativeCore: Datenbank bereits initialisiert, überspringe Fixture-Import.")
		return c.loadFromDB()
	}

	log.Println("NativeCore: Starte Daten-Import aus eingebetteten Fixtures (Initial Seeding)...")

	// 1. Settings laden
	if f, err := fixtureFS.Open("fixtures/settings.xml"); err != nil {
		log.Printf("NativeCore: Warnung - konnte settings.xml nicht öffnen: %v", err)
	} else {
		defer f.Close()
		var s core.XMLSettings
		if err := core.DecodeXML(f, &s); err != nil {
			log.Printf("NativeCore: Fehler beim Parsen von settings.xml: %v", err)
		} else {
			// In DB speichern (Key-Value)
			settingsMap := map[string]interface{}{
				"nick":               s.Nick,
				"port":               s.Port,
				"xmlport":            s.XMLPort,
				"maxupload":          s.MaxUpload,
				"maxdownload":        s.MaxDownload,
				"speedperslot":       s.SpeedPerSlot,
				"maxconnections":     s.MaxConnections,
				"autoconnect":        s.AutoConnect,
				"maxsourcesperfile":  s.MaxSourcesPerFile,
				"incomingdirectory":  s.IncomingDir,
				"temporarydirectory": s.TempDir,
			}
			for k, v := range settingsMap {
				_, _ = c.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", k, fmt.Sprintf("%v", v))
			}
			log.Println("NativeCore: Settings aus Fixtures geladen und in DB gespeichert.")
		}
	}

	// 2. Server laden (Format: <servers><server ... />)
	if f, err := fixtureFS.Open("fixtures/server.xml"); err != nil {
		log.Printf("NativeCore: Fehler - konnte server.xml nicht öffnen: %v", err)
	} else {
		defer f.Close()
		var list fixtureServerList
		if err := core.DecodeXML(f, &list); err != nil {
			log.Printf("NativeCore: Fehler beim Parsen von server.xml: %v", err)
		} else {
			for i, s := range list.Servers {
				c.servers = append(c.servers, domain.Server{
					ID:        fmt.Sprintf("srv-%d", i),
					Name:      s.Name,
					Host:      s.Host,
					Port:      s.Port,
					Connected: s.ConnectionTry > 0,
				})
				// In DB speichern
				_, _ = c.db.Exec("INSERT INTO servers (id, name, host, port, lastseen, connectiontry, connected) VALUES (?, ?, ?, ?, ?, ?, ?)",
					fmt.Sprintf("srv-%d", i), s.Name, s.Host, s.Port, s.LastSeen, s.ConnectionTry, s.ConnectionTry > 0)

				if s.ConnectionTry > 0 {
					c.info.Network.ConnectedWithServer = s.Name
				}
			}
			log.Printf("NativeCore: %d Server aus Fixtures geladen und in DB gespeichert.", len(list.Servers))
		}
	}

	// 3. Shares laden (Format: <database><file ... />)
	if f, err := fixtureFS.Open("fixtures/shareinfo.xml"); err != nil {
		log.Printf("NativeCore: Warnung - konnte shareinfo.xml nicht öffnen: %v", err)
	} else {
		defer f.Close()
		var list fixtureShareList
		if err := core.DecodeXML(f, &list); err != nil {
			log.Printf("NativeCore: Fehler beim Parsen von shareinfo.xml: %v", err)
		} else {
			for _, s := range list.Files {
				c.shares = append(c.shares, domain.Share{
					ID:       s.Name, // Wir nutzen den Pfad als ID
					Filename: s.Name,
					Size:     0,
					Priority: s.Priority,
				})
				// In DB speichern
				_, _ = c.db.Exec("INSERT INTO shares (filename, size, priority, lastasked, askcount, searchcount) VALUES (?, ?, ?, ?, ?, ?)",
					s.Name, 0, s.Priority, s.LastAsked, s.AskCount, s.SearchCount)
			}
			log.Printf("NativeCore: %d Dateien im Share geladen und in DB gespeichert.", len(list.Files))
		}
	}

	return nil
}

func (c *NativeCore) loadFromDB() error {
	// Settings laden
	rowsSettings, err := c.db.Query("SELECT key, value FROM settings")
	if err == nil {
		defer rowsSettings.Close()
		for rowsSettings.Next() {
			var key, value string
			if err := rowsSettings.Scan(&key, &value); err == nil {
				log.Printf("NativeCore: Setting geladen - %s: %s", key, value)
				// Hier könnten wir c.info oder andere Felder aktualisieren
			}
		}
	}

	// Server laden
	rows, err := c.db.Query("SELECT id, name, host, port, connected FROM servers")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s domain.Server
			if err := rows.Scan(&s.ID, &s.Name, &s.Host, &s.Port, &s.Connected); err == nil {
				c.servers = append(c.servers, s)
				if s.Connected {
					c.info.Network.ConnectedWithServer = s.Name
				}
			}
		}
	}

	// Shares laden
	rowsS, err := c.db.Query("SELECT filename, size, priority FROM shares")
	if err == nil {
		defer rowsS.Close()
		for rowsS.Next() {
			var s domain.Share
			if err := rowsS.Scan(&s.Filename, &s.Size, &s.Priority); err == nil {
				s.ID = s.Filename
				c.shares = append(c.shares, s)
			}
		}
	}

	log.Printf("NativeCore: %d Server und %d Shares aus Datenbank geladen.", len(c.servers), len(c.shares))
	return nil
}
