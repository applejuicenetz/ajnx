package core

import (
	"context"
	"fmt"

	"github.com/applejuicenetz/ajnx/internal/domain"
	"github.com/applejuicenetz/ajnx/internal/plugin"
)

// ErrForbidden wird zurückgegeben, wenn der Core eine Anfrage mit 403 ablehnt.
// Der Poller nutzt diesen Sentinel, um RefreshSession auszulösen.
var ErrForbidden = fmt.Errorf("core: forbidden (session expired)")

// CoreClient definiert die Schnittstelle zur Kommunikation mit einem appleJuice Core.
type CoreClient interface {
	// Abfragen
	Information(ctx context.Context) (*domain.Information, error)
	Downloads(ctx context.Context) ([]domain.Download, error)
	Uploads(ctx context.Context) ([]domain.Upload, error)
	Servers(ctx context.Context) ([]domain.Server, error)
	FullUpdate(ctx context.Context) (*domain.Information, []domain.Download, []domain.Upload, error)

	// Aktionen
	PauseDownload(ctx context.Context, ids ...string) error
	ResumeDownload(ctx context.Context, ids ...string) error
	CancelDownload(ctx context.Context, ids ...string) error
	SetPowerDownload(ctx context.Context, id string, powerDownload float64) error
	CleanDownloadList(ctx context.Context) error
	ProcessLink(ctx context.Context, link string) error
	ShutdownCore(ctx context.Context) error

	// Suche
	StartSearch(ctx context.Context, query string) error
	Searches(ctx context.Context) ([]domain.Search, error)
	CancelSearch(ctx context.Context, id string) error

	// Server
	ConnectServer(ctx context.Context, id string) error
	DisconnectServer(ctx context.Context) error
	RemoveServer(ctx context.Context, id string) error

	// Einstellungen
	GetSettings(ctx context.Context) (XMLSettings, error)
	UpdateSettings(ctx context.Context, s XMLSettings) error
	
	Shares(ctx context.Context) ([]domain.Share, error)
	GetShareDirs(ctx context.Context) ([]domain.ShareDir, error)
	SetShares(ctx context.Context, shares []domain.ShareDir) error

	// Authentifizierung
	CheckPassword(ctx context.Context, password string) (bool, error)
	RefreshSession(ctx context.Context) error

	// Monitoring
	SetEventBus(bus *plugin.EventBus)
}
