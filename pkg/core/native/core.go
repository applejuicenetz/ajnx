package native

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"github.com/applejuicenetz/ajnx/pkg/core"
	"github.com/applejuicenetz/ajnx/pkg/domain"
	"github.com/applejuicenetz/ajnx/pkg/plugin"
	"net"
)

type NativeCore struct {
	bus       *plugin.EventBus
	mu        sync.RWMutex
	db        *sql.DB
	info      domain.Information
	downloads []domain.Download
	uploads   []domain.Upload
	servers   []domain.Server
	shares    []domain.Share
	currentConn net.Conn
}

func NewNativeCore() *NativeCore {
	c := &NativeCore{
		downloads: []domain.Download{},
		uploads:   []domain.Upload{},
		servers:   []domain.Server{},
		shares:    []domain.Share{},
		info: domain.Information{
			Core: domain.CoreInfo{
				Version: "aJnX:Core 0.40.0-beta",
				OS:      domain.OSLinux,
			},
		
		},
	}

	// Datenbank initialisieren
	if err := c.initDB(); err != nil {
		log.Fatalf("NativeCore: Kritischer Fehler bei DB-Initialisierung: %v", err)
	}

	// Initialer Daten-Import aus den eingebetteten XML-Dateien (Seeding)
	_ = c.LoadFromFixtures()

	return c
}

// Sicherstellen, dass NativeCore das CoreClient Interface erfüllt.
var _ core.CoreClient = (*NativeCore)(nil)

func (c *NativeCore) Information(ctx context.Context) (*domain.Information, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &c.info, nil
}

func (c *NativeCore) Downloads(ctx context.Context) ([]domain.Download, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.downloads, nil
}

func (c *NativeCore) Uploads(ctx context.Context) ([]domain.Upload, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.uploads, nil
}

func (c *NativeCore) Servers(ctx context.Context) ([]domain.Server, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.servers, nil
}

func (c *NativeCore) FullUpdate(ctx context.Context) (*domain.Information, []domain.Download, []domain.Upload, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &c.info, c.downloads, c.uploads, nil
}

func (c *NativeCore) PauseDownload(ctx context.Context, ids ...string) error  { return nil }
func (c *NativeCore) ResumeDownload(ctx context.Context, ids ...string) error { return nil }
func (c *NativeCore) CancelDownload(ctx context.Context, ids ...string) error { return nil }
func (c *NativeCore) SetPowerDownload(ctx context.Context, id string, powerDownload float64) error {
	return nil
}
func (c *NativeCore) CleanDownloadList(ctx context.Context) error { return nil }
func (c *NativeCore) ProcessLink(ctx context.Context, link string) error  { return nil }
func (c *NativeCore) ShutdownCore(ctx context.Context) error             { return nil }

func (c *NativeCore) StartSearch(ctx context.Context, query string) error { return nil }
func (c *NativeCore) Searches(ctx context.Context) ([]domain.Search, error) {
	return []domain.Search{}, nil
}
func (c *NativeCore) CancelSearch(ctx context.Context, id string) error { return nil }

func (c *NativeCore) ConnectServer(ctx context.Context, id string) error {
	c.mu.RLock()
	var target *domain.Server
	for _, s := range c.servers {
		if s.ID == id {
			target = &s
			break
		}
	}
	c.mu.RUnlock()

	if target == nil {
		return fmt.Errorf("server nicht gefunden")
	}

	return c.ConnectToServer(ctx, target.Host, target.Port)
}
func (c *NativeCore) DisconnectServer(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.currentConn != nil {
		c.currentConn.Close()
		c.currentConn = nil
	}
	c.info.Network.ConnectedWithServer = ""
	
	// Alle Server als nicht verbunden markieren
	for i := range c.servers {
		c.servers[i].Connected = false
	}

	return nil
}

func (c *NativeCore) RemoveServer(ctx context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Aus DB löschen
	_, _ = c.db.Exec("DELETE FROM servers WHERE id = ?", id)

	// Aus Speicher löschen
	for i, s := range c.servers {
		if s.ID == id {
			c.servers = append(c.servers[:i], c.servers[i+1:]...)
			break
		}
	}
	return nil
}

func (c *NativeCore) GetSettings(ctx context.Context) (core.XMLSettings, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	s := core.XMLSettings{
		Nick:              c.getSetting("nick"),
		IncomingDir:       c.getSetting("incomingdirectory"),
		TempDir:           c.getSetting("temporarydirectory"),
		AutoConnect:       c.getSetting("autoconnect"),
	}

	fmt.Sscanf(c.getSetting("port"), "%d", &s.Port)
	fmt.Sscanf(c.getSetting("xmlport"), "%d", &s.XMLPort)
	fmt.Sscanf(c.getSetting("maxupload"), "%d", &s.MaxUpload)
	fmt.Sscanf(c.getSetting("maxdownload"), "%d", &s.MaxDownload)
	fmt.Sscanf(c.getSetting("speedperslot"), "%d", &s.SpeedPerSlot)
	fmt.Sscanf(c.getSetting("maxconnections"), "%d", &s.MaxConnections)
	fmt.Sscanf(c.getSetting("maxsourcesperfile"), "%d", &s.MaxSourcesPerFile)

	return s, nil
}

func (c *NativeCore) UpdateSettings(ctx context.Context, s core.XMLSettings) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setSetting("nick", s.Nick)
	c.setSetting("port", s.Port)
	c.setSetting("xmlport", s.XMLPort)
	c.setSetting("maxupload", s.MaxUpload)
	c.setSetting("maxdownload", s.MaxDownload)
	c.setSetting("speedperslot", s.SpeedPerSlot)
	c.setSetting("maxconnections", s.MaxConnections)
	c.setSetting("autoconnect", s.AutoConnect)
	c.setSetting("maxsourcesperfile", s.MaxSourcesPerFile)
	c.setSetting("incomingdirectory", s.IncomingDir)
	c.setSetting("temporarydirectory", s.TempDir)

	return nil
}

func (c *NativeCore) Shares(ctx context.Context) ([]domain.Share, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.shares, nil
}
func (c *NativeCore) GetShareDirs(ctx context.Context) ([]domain.ShareDir, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows, err := c.db.Query("SELECT path, with_subfolders FROM shared_directories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dirs []domain.ShareDir
	for rows.Next() {
		var d domain.ShareDir
		if err := rows.Scan(&d.Path, &d.WithSubfolders); err == nil {
			dirs = append(dirs, d)
		}
	}
	return dirs, nil
}

func (c *NativeCore) SetShares(ctx context.Context, shares []domain.ShareDir) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, _ = tx.Exec("DELETE FROM shared_directories")
	for _, d := range shares {
		_, err := tx.Exec("INSERT INTO shared_directories (path, with_subfolders) VALUES (?, ?)", d.Path, d.WithSubfolders)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (c *NativeCore) CheckPassword(ctx context.Context, password string) (bool, error) {
	return true, nil
}
func (c *NativeCore) RefreshSession(ctx context.Context) error { return nil }

func (c *NativeCore) SetEventBus(bus *plugin.EventBus) {
	c.bus = bus
}
