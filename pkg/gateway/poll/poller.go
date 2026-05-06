package poll

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/applejuicenetz/ajnx/pkg/core"
	"github.com/applejuicenetz/ajnx/pkg/domain"
	"github.com/applejuicenetz/ajnx/pkg/gateway"
	"github.com/applejuicenetz/ajnx/pkg/setup"
)

// Poller verwaltet die Hintergrund-Abfragen des Cores.
type Poller struct {
	client     core.CoreClient
	state      *gateway.State
	setupMgr   *setup.Manager
	lastConfig string             // "host:port:password" zur Erkennung von Änderungen
	verInfo    domain.VersionInfo // Cache für die Versions-Info
}

// NewPoller erstellt einen neuen Poller.
func NewPoller(client core.CoreClient, state *gateway.State, setupMgr *setup.Manager) *Poller {
	return &Poller{
		client:   client,
		state:    state,
		setupMgr: setupMgr,
	}
}

// Run initialisiert die Core-Session und startet die Polling-Loops.
func (p *Poller) Run(ctx context.Context) {
	// Initialer Sync der Config falls Manager vorhanden
	p.checkConfigReload()

	if err := p.client.RefreshSession(ctx); err != nil {
		log.Printf("poller: initiale Session konnte nicht geholt werden (noch kein Setup?): %v", err)
	}

	// Initialer Check für Updates
	p.updateVersion(ctx)

	go p.pollLoop(ctx, 5*time.Second, p.updateAll)
	go p.pollLoop(ctx, 30*time.Second, p.updateServers)
	go p.pollLoop(ctx, 1*time.Hour, p.updateVersion)
}

func (p *Poller) checkConfigReload() {
	if p.setupMgr == nil || !p.setupMgr.IsComplete() {
		return
	}

	snap := p.setupMgr.Snapshot()
	current := fmt.Sprintf("%s:%d:%s", snap.CoreHost, snap.CorePort, snap.CorePassword)

	if current != p.lastConfig {
		log.Printf("poller: Neue Konfiguration erkannt (Core: %s:%d). Aktualisiere Client...", snap.CoreHost, snap.CorePort)
		if xmlClient, ok := p.client.(*core.XMLCoreClient); ok {
			baseURL := fmt.Sprintf("%s:%d", snap.CoreHost, snap.CorePort)
			xmlClient.UpdateConfig(baseURL, snap.CorePassword)
			p.lastConfig = current
			// Wir setzen LastError zurück, falls wir nun wieder verbunden sind
			p.state.UpdateError(nil)
		}
	}
}

func (p *Poller) updateAll(ctx context.Context) error {
	p.checkConfigReload()
	
	info, downloads, uploads, err := p.client.FullUpdate(ctx)
	if err != nil {
		p.state.UpdateError(err)
		return err
	}

	info.Version = p.verInfo

	// Aktive Transfers zählen
	activeDL := 0
	for _, d := range downloads {
		if d.SpeedBps > 0 {
			activeDL++
		}
	}
	activeUL := 0
	for _, u := range uploads {
		if u.Status == domain.UploadTransferring {
			activeUL++
		}
	}
	info.Session.ActiveDownloads = activeDL
	info.Session.ActiveUploads = activeUL

	p.state.UpdateInformation(info)
	p.state.UpdateDownloads(downloads)
	p.state.UpdateUploads(uploads)
	p.state.UpdateError(nil)
	return nil
}

func (p *Poller) updateVersion(ctx context.Context) error {
	p.verInfo = checkUpdate(domain.Version)
	return nil
}

func (p *Poller) pollLoop(ctx context.Context, interval time.Duration, task func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := task(ctx); err != nil {
				if errors.Is(err, core.ErrForbidden) {
					log.Println("poller: Session abgelaufen – erneuere Session")
					if refreshErr := p.client.RefreshSession(ctx); refreshErr != nil {
						log.Printf("poller: Session-Erneuerung fehlgeschlagen: %v", refreshErr)
						p.state.UpdateError(refreshErr)
					}
				} else {
					log.Printf("poller: Fehler beim Polling: %v", err)
				}
			}
		}
	}
}

func (p *Poller) updateServers(ctx context.Context) error {
	servers, err := p.client.Servers(ctx)
	if err != nil {
		return err
	}
	p.state.UpdateServers(servers)
	return nil
}
