package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/applejuicenetz/ajnx/internal/domain"
	"github.com/applejuicenetz/ajnx/internal/ui/view"
)

var serversGatewayClient = &http.Client{Timeout: 5 * time.Second}

type ServersHandler struct {
	gatewayURL   string
	gatewayToken string
	nickname     string
}

func NewServersHandler(gatewayURL, gatewayToken, nickname string) *ServersHandler {
	return &ServersHandler{gatewayURL: gatewayURL, gatewayToken: gatewayToken, nickname: nickname}
}

func (h *ServersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/server" && r.Method == http.MethodGet:
		h.serveFullPage(w, r)
	case path == "/partials/servers-cards" && r.Method == http.MethodGet:
		h.serveCards(w, r)
	case strings.HasPrefix(path, "/servers/") && r.Method == http.MethodPost:
		if path == "/servers/disconnect" {
			h.handleDisconnect(w, r)
			return
		}
		h.serveAction(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *ServersHandler) serveFullPage(w http.ResponseWriter, r *http.Request) {
	servers, err := h.fetchServers(r)
	if err != nil {
		http.Error(w, "Serverliste nicht verfügbar", http.StatusServiceUnavailable)
		return
	}
	view.ServersPage(servers, h.nickname).Render(r.Context(), w)
}

func (h *ServersHandler) serveCards(w http.ResponseWriter, r *http.Request) {
	servers, err := h.fetchServers(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	view.ServerCards(servers).Render(r.Context(), w)
}

func (h *ServersHandler) serveAction(w http.ResponseWriter, r *http.Request) {
	// /servers/{id}/{action}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		http.NotFound(w, r)
		return
	}
	id := parts[2]
	action := parts[3]

	var endpoint string
	switch action {
	case "connect":
		endpoint = fmt.Sprintf("%s/api/servers/%s/connect", h.gatewayURL, id)
	case "remove":
		endpoint = fmt.Sprintf("%s/api/servers/%s/remove", h.gatewayURL, id)
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

	resp, err := serversGatewayClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		http.Error(w, "Aktion fehlgeschlagen", http.StatusBadGateway)
		return
	}
	resp.Body.Close()

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", "refreshServers")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Redirect(w, r, "/server", http.StatusSeeOther)
}

func (h *ServersHandler) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	endpoint := fmt.Sprintf("%s/api/servers/disconnect", h.gatewayURL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, nil)
	if err != nil {
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := serversGatewayClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		http.Error(w, "Trennen fehlgeschlagen", http.StatusBadGateway)
		return
	}
	resp.Body.Close()

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", "refreshServers")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/server", http.StatusSeeOther)
}

func (h *ServersHandler) fetchServers(r *http.Request) ([]domain.Server, error) {
	url := fmt.Sprintf("%s/api/servers", h.gatewayURL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := serversGatewayClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned %d", resp.StatusCode)
	}

	var servers []domain.Server
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, err
	}
	return servers, nil
}
