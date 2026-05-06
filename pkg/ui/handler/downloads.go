package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/applejuicenetz/ajnx/pkg/domain"
	"github.com/applejuicenetz/ajnx/pkg/ui/view"
)

var downloadsGatewayClient = &http.Client{Timeout: 5 * time.Second}

type DownloadsHandler struct {
	gatewayURL   string
	gatewayToken string
	nickname     string
}

func NewDownloadsHandler(gatewayURL, gatewayToken, nickname string) *DownloadsHandler {
	return &DownloadsHandler{gatewayURL: gatewayURL, gatewayToken: gatewayToken, nickname: nickname}
}

func (h *DownloadsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/downloads" && r.Method == http.MethodGet:
		h.serveFullPage(w, r)
	case path == "/partials/downloads-list" && r.Method == http.MethodGet:
		h.serveTableRows(w, r)
	case path == "/partials/downloads-cards" && r.Method == http.MethodGet:
		h.serveCards(w, r)
	case strings.HasPrefix(path, "/downloads/") && r.Method == http.MethodPost:
		if path == "/downloads/clean" {
			h.handleClean(w, r)
			return
		}
		h.serveAction(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *DownloadsHandler) serveFullPage(w http.ResponseWriter, r *http.Request) {
	downloads, err := h.fetchDownloads(r)
	if err != nil {
		http.Error(w, "Downloads nicht verfügbar", http.StatusServiceUnavailable)
		return
	}
	view.DownloadsPage(downloads, h.nickname).Render(r.Context(), w)
}

func (h *DownloadsHandler) serveTableRows(w http.ResponseWriter, r *http.Request) {
	downloads, err := h.fetchDownloads(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	view.DownloadRows(downloads).Render(r.Context(), w)
}

func (h *DownloadsHandler) serveCards(w http.ResponseWriter, r *http.Request) {
	downloads, err := h.fetchDownloads(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	view.DownloadCards(downloads).Render(r.Context(), w)
}

// serveAction leitet Pause/Resume/Cancel-Anfragen an das Gateway weiter.
// Erwartet Pfad: /downloads/{id}/{action}
func (h *DownloadsHandler) serveAction(w http.ResponseWriter, r *http.Request) {
	// /downloads/{id}/{action} → ["", "downloads", "{id}", "{action}"]
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		http.NotFound(w, r)
		return
	}
	id := parts[2]
	action := parts[3]

	var endpoint string
	switch action {
	case "pause":
		endpoint = fmt.Sprintf("%s/api/downloads/%s/pause", h.gatewayURL, id)
	case "resume":
		endpoint = fmt.Sprintf("%s/api/downloads/%s/resume", h.gatewayURL, id)
	case "cancel":
		endpoint = fmt.Sprintf("%s/api/downloads/%s/cancel", h.gatewayURL, id)
	case "pdl-inc", "pdl-dec":
		h.handlePDLAdjustment(w, r, id, action == "pdl-inc")
		return
	default:
		http.NotFound(w, r)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, nil)
	if err != nil {
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := downloadsGatewayClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		http.Error(w, "Aktion fehlgeschlagen", http.StatusBadGateway)
		return
	}
	resp.Body.Close()

	// Wenn es eine HTMX-Anfrage ist, kein Full-Reload, sondern Trigger für Auto-Refresh senden
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", "refreshDownloads")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Fallback für non-HTMX
	http.Redirect(w, r, "/downloads", http.StatusSeeOther)
}

func (h *DownloadsHandler) handlePDLAdjustment(w http.ResponseWriter, r *http.Request, id string, increment bool) {
	downloads, err := h.fetchDownloads(r)
	if err != nil {
		http.Error(w, "Fehler beim Abrufen der Downloads", http.StatusInternalServerError)
		return
	}

	var currentPDL float64
	found := false
	for _, d := range downloads {
		if d.ID == id {
			currentPDL = d.PowerDownload
			found = true
			break
		}
	}

	if !found {
		http.NotFound(w, r)
		return
	}

	newPDL := currentPDL
	if increment {
		newPDL += 1.0
	} else {
		newPDL -= 1.0
	}

	if newPDL < 0 { // appleJuice Powerdownload geht bis -10 Core-Wert, was 0.0 in UI ist
		newPDL = 0
	}

	// Sende an Gateway
	endpoint := fmt.Sprintf("%s/api/downloads/%s/pdl", h.gatewayURL, id)
	payload := fmt.Sprintf(`{"value": %.1f}`, newPDL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, strings.NewReader(payload))
	if err != nil {
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := downloadsGatewayClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		http.Error(w, "PDL-Anpassung fehlgeschlagen", http.StatusBadGateway)
		return
	}
	resp.Body.Close()

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", "refreshDownloads")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/downloads", http.StatusSeeOther)
}

func (h *DownloadsHandler) handleClean(w http.ResponseWriter, r *http.Request) {
	endpoint := fmt.Sprintf("%s/api/downloads/clean", h.gatewayURL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, nil)
	if err != nil {
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := downloadsGatewayClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		http.Error(w, "Cleanup fehlgeschlagen", http.StatusBadGateway)
		return
	}
	resp.Body.Close()

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", "refreshDownloads")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/downloads", http.StatusSeeOther)
}

func (h *DownloadsHandler) fetchDownloads(r *http.Request) ([]domain.Download, error) {
	url := fmt.Sprintf("%s/api/downloads", h.gatewayURL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := downloadsGatewayClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned %d", resp.StatusCode)
	}

	var downloads []domain.Download
	if err := json.NewDecoder(resp.Body).Decode(&downloads); err != nil {
		return nil, err
	}
	return downloads, nil
}
