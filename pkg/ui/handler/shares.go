package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/applejuicenetz/ajnx/pkg/domain"
	"github.com/applejuicenetz/ajnx/pkg/ui/view"
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
	path := r.URL.Path

	if r.Method == http.MethodPost {
		switch path {
		case "/shares/add":
			h.handleAdd(w, r)
		case "/shares/remove":
			h.handleRemove(w, r)
		case "/shares/toggle":
			h.handleToggle(w, r)
		case "/shares/check":
			h.handleCheck(w, r)
		default:
			http.NotFound(w, r)
		}
		return
	}

	if path == "/shares/browse" {
		dir := r.URL.Query().Get("path")
		if dir == "" {
			dir = "/"
		}
		dirs, _ := h.fetchDirectories(r, dir)
		view.DirectoryBrowser(dir, dirs).Render(r.Context(), w)
		return
	}

	if path == "/partials/share-dirs" {
		dirs, _ := h.fetchShareDirs(r)
		settings, _ := h.fetchSettings(r)
		tempDir := ""
		incomingDir := ""
		var pathMappings map[string]string
		if settings != nil {
			tempDir = settings.TempDir
			incomingDir = settings.IncomingDir
			pathMappings = settings.PathMappings
		}
		view.ShareDirsTable(dirs, tempDir, incomingDir, pathMappings).Render(r.Context(), w)
		return
	}

	shares, _ := h.fetchShares(r)
	dirs, _ := h.fetchShareDirs(r)
	settings, _ := h.fetchSettings(r)
	tempDir := ""
	incomingDir := ""
	var pathMappings map[string]string
	if settings != nil {
		tempDir = settings.TempDir
		incomingDir = settings.IncomingDir
		pathMappings = settings.PathMappings
	}

	view.SharesPage(shares, dirs, tempDir, incomingDir, pathMappings, h.nickname).Render(r.Context(), w)
}

func (h *SharesHandler) handleAdd(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	withSubs := r.FormValue("with_subs") == "1"

	if path == "" {
		http.Error(w, "Pfad erforderlich", http.StatusBadRequest)
		return
	}

	payload := struct {
		Path     string `json:"path"`
		WithSubs bool   `json:"with_subs"`
	}{Path: path, WithSubs: withSubs}

	if err := h.doPost(r, "/api/shares/dirs/add", payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshShares")
	w.WriteHeader(http.StatusOK)
}

func (h *SharesHandler) handleRemove(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Pfad erforderlich", http.StatusBadRequest)
		return
	}

	payload := struct {
		Path string `json:"path"`
	}{Path: path}

	if err := h.doPost(r, "/api/shares/dirs/remove", payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshShares")
	w.WriteHeader(http.StatusOK)
}

func (h *SharesHandler) handleToggle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	newMode := r.URL.Query().Get("subs") == "1"

	if path == "" {
		http.Error(w, "Pfad erforderlich", http.StatusBadRequest)
		return
	}

	payload := struct {
		Path     string `json:"path"`
		WithSubs bool   `json:"with_subs"`
	}{Path: path, WithSubs: newMode}

	if err := h.doPost(r, "/api/shares/dirs/toggle", payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshShares")
	w.WriteHeader(http.StatusOK)
}

func (h *SharesHandler) handleCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.doPost(r, "/api/shares/check", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SharesHandler) doPost(r *http.Request, apiPath string, payload interface{}) error {
	url := h.gatewayURL + apiPath
	var bodyReader *strings.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = strings.NewReader(string(b))
	} else {
		bodyReader = strings.NewReader("")
	}

	req, err := http.NewRequestWithContext(r.Context(), "POST", url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	req.Header.Set("Content-Type", "application/json")
	addNativeHeader(r, req)

	resp, err := gatewayClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Gateway-Fehler: %d", resp.StatusCode)
	}
	return nil
}

func (h *SharesHandler) fetchSettings(r *http.Request) (*domain.Settings, error) {
	url := h.gatewayURL + "/api/settings"
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

	var s domain.Settings
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
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

func (h *SharesHandler) fetchDirectories(r *http.Request, path string) ([]string, error) {
	url := fmt.Sprintf("%s/api/browse?path=%s", h.gatewayURL, strings.ReplaceAll(path, "+", "%20"))
	req, _ := http.NewRequestWithContext(r.Context(), "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := gatewayClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dirs []string
	json.NewDecoder(resp.Body).Decode(&dirs)
	return dirs, nil
}
