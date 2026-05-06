package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/applejuicenetz/ajnx/pkg/core"
	"github.com/applejuicenetz/ajnx/pkg/setup"
	"github.com/applejuicenetz/ajnx/pkg/ui/view"
)

type SettingsHandler struct {
	setupMgr   *setup.Manager
	gatewayURL string
	token      string
	nickname   string
}

func NewSettingsHandler(setupMgr *setup.Manager, gatewayURL, token, nickname string) *SettingsHandler {
	return &SettingsHandler{
		setupMgr:   setupMgr,
		gatewayURL: gatewayURL,
		token:      token,
		nickname:   nickname,
	}
}

func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lock := h.setupMgr.Snapshot()
	
	// Sicherstellen, dass die Map initialisiert ist
	if lock.Plugins == nil {
		lock.Plugins = make(map[string]bool)
	}

	// Core-Einstellungen laden
	coreSettings, err := h.fetchCoreSettings(r)
	if err != nil {
		// Wenn der Core nicht erreichbar ist, zeigen wir eine Warnung
	}

	view.SettingsPage("settings", h.nickname, lock, coreSettings).Render(r.Context(), w)
}

func (h *SettingsHandler) fetchCoreSettings(r *http.Request) (core.XMLSettings, error) {
	url := h.gatewayURL + "/api/settings"
	req, err := http.NewRequestWithContext(r.Context(), "GET", url, nil)
	if err != nil {
		return core.XMLSettings{}, err
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	addNativeHeader(r, req)

	resp, err := gatewayClient.Do(req)
	if err != nil {
		return core.XMLSettings{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return core.XMLSettings{}, fmt.Errorf("Gateway-Fehler: %d", resp.StatusCode)
	}

	var settings core.XMLSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return core.XMLSettings{}, err
	}
	return settings, nil
}

func (h *SettingsHandler) HandleCoreSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Form-Daten in XMLSettings konvertieren
	r.ParseForm()
	
	maxUp, _ := strconv.ParseInt(r.FormValue("max_upload"), 10, 64)
	maxDown, _ := strconv.ParseInt(r.FormValue("max_download"), 10, 64)
	speedSlot, _ := strconv.ParseInt(r.FormValue("speed_per_slot"), 10, 64)
	maxConn, _ := strconv.Atoi(r.FormValue("max_connections"))
	maxSources, _ := strconv.Atoi(r.FormValue("max_sources_per_file"))
	port, _ := strconv.Atoi(r.FormValue("port"))
	xmlPort, _ := strconv.Atoi(r.FormValue("xml_port"))
	
	settings := core.XMLSettings{
		Nick:              r.FormValue("nickname"),
		MaxUpload:         maxUp * 1024, // KB zu Bytes
		MaxDownload:       maxDown * 1024,
		SpeedPerSlot:      speedSlot * 1024,
		MaxConnections:    maxConn,
		MaxSourcesPerFile: maxSources,
		AutoConnect:       r.FormValue("auto_connect"),
		IncomingDir:       r.FormValue("incoming_dir"),
		TempDir:           r.FormValue("temp_dir"),
		Port:              port,
		XMLPort:           xmlPort,
	}

	jsonBody, _ := json.Marshal(settings)
	url := h.gatewayURL + "/api/settings"
	req, err := http.NewRequestWithContext(r.Context(), "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	addNativeHeader(r, req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := gatewayClient.Do(req)
	if err != nil {
		http.Error(w, "Gateway nicht erreichbar", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Fehler beim Speichern: %s", string(body)), resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Einstellungen gespeichert"))
}

func (h *SettingsHandler) HandleTogglePlugin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	enabled := r.FormValue("enabled") == "on"

	lock := h.setupMgr.Snapshot()
	if lock.Plugins == nil {
		lock.Plugins = make(map[string]bool)
	}
	
	lock.Plugins[id] = enabled
	
	if err := h.setupMgr.UpdatePlugins(lock.Plugins); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SettingsHandler) HandleUpdatePathMappings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var mappings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&mappings); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.setupMgr.UpdatePathMappings(mappings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
