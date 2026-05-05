package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/applejuicenetz/ajnx/internal/domain"
	"github.com/applejuicenetz/ajnx/internal/ui/view"
)

var searchGatewayClient = &http.Client{Timeout: 15 * time.Second}

type SearchHandler struct {
	gatewayURL   string
	gatewayToken string
	nickname     string
}

func NewSearchHandler(gatewayURL, gatewayToken, nickname string) *SearchHandler {
	return &SearchHandler{
		gatewayURL:   gatewayURL,
		gatewayToken: gatewayToken,
		nickname:     nickname,
	}
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if strings.HasSuffix(r.URL.Path, "/cancel") {
			h.HandleCancel(w, r)
			return
		}
		h.HandleStart(w, r)
		return
	}

	searches, err := h.fetchSearches(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Page-Parameter sammeln
	pages := make(map[string]int)
	for k, v := range r.URL.Query() {
		if strings.HasPrefix(k, "page_") && len(v) > 0 {
			var p int
			fmt.Sscanf(v[0], "%d", &p)
			if p > 0 {
				pages[k[5:]] = p
			}
		}
	}

	view.SearchPage("search", h.nickname, searches, pages).Render(r.Context(), w)
}

func (h *SearchHandler) ServeResults(w http.ResponseWriter, r *http.Request) {
	searches, err := h.fetchSearches(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Page-Parameter sammeln
	pages := make(map[string]int)
	for k, v := range r.URL.Query() {
		if strings.HasPrefix(k, "page_") && len(v) > 0 {
			var p int
			fmt.Sscanf(v[0], "%d", &p)
			if p > 0 {
				pages[k[5:]] = p
			}
		}
	}

	view.SearchResults(searches, pages).Render(r.Context(), w)
}

func (h *SearchHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	if query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	payload := map[string]string{"query": query}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(r.Context(), "POST", h.gatewayURL+"/api/search", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := searchGatewayClient.Do(req)
	if err != nil {
		http.Error(w, "Gateway nicht erreichbar", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Gateway Fehler: %s", string(body)), resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("<div class='p-4 bg-primary/10 text-primary rounded-xl mb-4 animate-pulse'>Suche wurde gestartet, Ergebnisse werden geladen...</div>"))
}

func (h *SearchHandler) HandleCancel(w http.ResponseWriter, r *http.Request) {
	// ID aus dem Pfad extrahieren (/search/{id}/cancel)
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := parts[2]

	req, _ := http.NewRequestWithContext(r.Context(), "POST", h.gatewayURL+"/api/search/"+id+"/cancel", nil)
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := searchGatewayClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Fehler beim Abbrechen der Suche", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
}

func (h *SearchHandler) fetchSearches(r *http.Request) ([]domain.Search, error) {
	req, _ := http.NewRequestWithContext(r.Context(), "GET", h.gatewayURL+"/api/search", nil)
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := searchGatewayClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}

	var searches []domain.Search
	if err := json.NewDecoder(resp.Body).Decode(&searches); err != nil {
		return nil, err
	}
	return searches, nil
}
