package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/applejuicenetz/ajnx/internal/domain"
	"github.com/applejuicenetz/ajnx/internal/ui/view"
)

var uploadsGatewayClient = &http.Client{Timeout: 5 * time.Second}

type UploadsHandler struct {
	gatewayURL   string
	gatewayToken string
	nickname     string
}

func NewUploadsHandler(gatewayURL, gatewayToken, nickname string) *UploadsHandler {
	return &UploadsHandler{gatewayURL: gatewayURL, gatewayToken: gatewayToken, nickname: nickname}
}

func (h *UploadsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/uploads" && r.Method == http.MethodGet:
		h.serveFullPage(w, r)
	case path == "/partials/uploads-list" && r.Method == http.MethodGet:
		h.serveTableRows(w, r)
	case path == "/partials/uploads-cards" && r.Method == http.MethodGet:
		h.serveCards(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *UploadsHandler) serveFullPage(w http.ResponseWriter, r *http.Request) {
	uploads, err := h.fetchUploads(r)
	if err != nil {
		http.Error(w, "Uploads nicht verfügbar", http.StatusServiceUnavailable)
		return
	}
	view.UploadsPage(uploads, h.nickname).Render(r.Context(), w)
}

func (h *UploadsHandler) serveTableRows(w http.ResponseWriter, r *http.Request) {
	uploads, err := h.fetchUploads(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	view.UploadRows(uploads).Render(r.Context(), w)
}

func (h *UploadsHandler) serveCards(w http.ResponseWriter, r *http.Request) {
	uploads, err := h.fetchUploads(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	view.UploadCards(uploads).Render(r.Context(), w)
}

func (h *UploadsHandler) fetchUploads(r *http.Request) ([]domain.Upload, error) {
	url := fmt.Sprintf("%s/api/uploads", h.gatewayURL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := uploadsGatewayClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned %d", resp.StatusCode)
	}

	var uploads []domain.Upload
	if err := json.NewDecoder(resp.Body).Decode(&uploads); err != nil {
		return nil, err
	}
	return uploads, nil
}
