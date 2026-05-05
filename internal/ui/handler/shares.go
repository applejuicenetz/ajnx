package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/applejuicenetz/ajnx/internal/domain"
	"github.com/applejuicenetz/ajnx/internal/ui/view"
)

type SharesHandler struct {
	gatewayURL   string
	gatewayToken string
	nickname     string
}

func NewSharesHandler(gatewayURL, gatewayToken, nickname string) *SharesHandler {
	return &SharesHandler{
		gatewayURL:   gatewayURL,
		gatewayToken: gatewayToken,
		nickname:     nickname,
	}
}

func (h *SharesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.URL.Path == "/shares/dirs/add" {
			h.handleAddDir(w, r)
			return
		}
		if r.URL.Path == "/shares/dirs/remove" {
			h.handleRemoveDir(w, r)
			return
		}
	}

	shares, err := h.fetchShares(r)
	if err != nil {
		// Bei Fehlern zeigen wir eine leere Liste oder Fehlermeldung
	}

	dirs, err := h.fetchShareDirs(r)
	if err != nil {
		// ...
	}

	view.SharesPage("shares", h.nickname, shares, dirs).Render(r.Context(), w)
}

func (h *SharesHandler) handleAddDir(w http.ResponseWriter, r *http.Request) {
	var newDir domain.ShareDir
	if err := json.NewDecoder(r.Body).Decode(&newDir); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// 1. Hole aktuelle Dirs
	dirs, err := h.fetchShareDirs(r)
	if err != nil {
		http.Error(w, "Fehler beim Abrufen der Verzeichnisse", http.StatusBadGateway)
		return
	}

	// 2. Prüfen ob bereits da
	for _, d := range dirs {
		if d.Path == newDir.Path {
			w.WriteHeader(http.StatusOK) // Bereits da, Erfolg vortäuschen
			return
		}
	}

	// 3. Hinzufügen
	dirs = append(dirs, newDir)

	// 4. Speichern
	if err := h.saveShareDirs(r, dirs); err != nil {
		http.Error(w, "Fehler beim Speichern", http.StatusBadGateway)
		return
	}

	w.Header().Set("HX-Trigger", "refreshShares")
	w.WriteHeader(http.StatusOK)
}

func (h *SharesHandler) handleRemoveDir(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Pfad fehlt", http.StatusBadRequest)
		return
	}

	// 1. Hole aktuelle Dirs
	dirs, err := h.fetchShareDirs(r)
	if err != nil {
		http.Error(w, "Fehler beim Abrufen der Verzeichnisse", http.StatusBadGateway)
		return
	}

	// 2. Filtern
	var newDirs []domain.ShareDir
	for _, d := range dirs {
		if d.Path != path {
			newDirs = append(newDirs, d)
		}
	}

	// 3. Speichern
	if err := h.saveShareDirs(r, newDirs); err != nil {
		http.Error(w, "Fehler beim Speichern", http.StatusBadGateway)
		return
	}

	w.Header().Set("HX-Trigger", "refreshShares")
	w.WriteHeader(http.StatusOK)
}

func (h *SharesHandler) saveShareDirs(r *http.Request, dirs []domain.ShareDir) error {
	url := h.gatewayURL + "/api/shares/dirs"
	body, _ := json.Marshal(dirs)
	req, err := http.NewRequestWithContext(r.Context(), "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := gatewayClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Gateway-Fehler: %d", resp.StatusCode)
	}
	return nil
}

func (h *SharesHandler) fetchShares(r *http.Request) ([]domain.Share, error) {
	url := h.gatewayURL + "/api/shares"
	req, err := http.NewRequestWithContext(r.Context(), "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := gatewayClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gateway-Fehler: %d", resp.StatusCode)
	}

	var shares []domain.Share
	if err := json.NewDecoder(resp.Body).Decode(&shares); err != nil {
		return nil, err
	}
	return shares, nil
}

func (h *SharesHandler) fetchShareDirs(r *http.Request) ([]domain.ShareDir, error) {
	url := h.gatewayURL + "/api/shares/dirs"
	req, err := http.NewRequestWithContext(r.Context(), "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := gatewayClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gateway-Fehler: %d", resp.StatusCode)
	}

	var dirs []domain.ShareDir
	if err := json.NewDecoder(resp.Body).Decode(&dirs); err != nil {
		return nil, err
	}
	return dirs, nil
}
