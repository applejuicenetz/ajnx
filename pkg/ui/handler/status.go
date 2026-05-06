package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/applejuicenetz/ajnx/pkg/domain"
	"github.com/applejuicenetz/ajnx/pkg/ui/view"
)

var gatewayClient = &http.Client{Timeout: 5 * time.Second}

type StatusHandler struct {
	gatewayURL   string
	gatewayToken string
	nickname     string
}

func NewStatusHandler(gatewayURL, gatewayToken, nickname string) *StatusHandler {
	return &StatusHandler{gatewayURL: gatewayURL, gatewayToken: gatewayToken, nickname: nickname}
}

// fetchStatus holt Statusdaten vom Gateway mit dem konfigurierten Service-Token.
func (h *StatusHandler) fetchStatus(r *http.Request) (*domain.Information, int, string) {
	url := fmt.Sprintf("%s/api/status", h.gatewayURL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := gatewayClient.Do(req)
	if err != nil {
		return nil, http.StatusServiceUnavailable, "Gateway nicht erreichbar"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fehlermeldung vom Gateway lesen
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		msg := string(body[:n])
		if msg == "" {
			msg = fmt.Sprintf("Gateway returned %d", resp.StatusCode)
		}
		return nil, resp.StatusCode, msg
	}

	var info domain.Information
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}
	return &info, http.StatusOK, ""
}

func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	info, _, msg := h.fetchStatus(r)
	if msg != "" {
		view.StatusError(msg, h.nickname).Render(r.Context(), w)
		return
	}
	view.StatusPage(info, h.nickname).Render(r.Context(), w)
}

func (h *StatusHandler) ServeStatusStats(w http.ResponseWriter, r *http.Request) {
	info, _, _ := h.fetchStatus(r)
	// Wir geben auch bei Fehlern (info == nil) das Template zurück.
	// Das Template muss damit umgehen können (Null-Checks).
	view.StatusStats(info).Render(r.Context(), w)
}

func (h *StatusHandler) ServeShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := fmt.Sprintf("%s/api/core/shutdown", h.gatewayURL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := gatewayClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		http.Error(w, "Core konnte nicht beendet werden", http.StatusBadGateway)
		return
	}
	resp.Body.Close()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Core wird beendet..."))
}
