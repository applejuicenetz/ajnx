package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var linksGatewayClient = &http.Client{Timeout: 5 * time.Second}

type LinksHandler struct {
	gatewayURL   string
	gatewayToken string
}

func NewLinksHandler(gatewayURL, gatewayToken string) *LinksHandler {
	return &LinksHandler{
		gatewayURL:   gatewayURL,
		gatewayToken: gatewayToken,
	}
}

func (h *LinksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	link := ""
	if r.Method == http.MethodPost {
		r.ParseForm()
		link = strings.TrimSpace(r.FormValue("link"))
	} else if r.Method == http.MethodGet {
		link = strings.TrimSpace(r.URL.Query().Get("link"))
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if link == "" {
		if r.Method == http.MethodGet {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, "Link ist erforderlich", http.StatusBadRequest)
		return
	}

	// Browser sendet bei web+ajfsp oft den ganzen String inkl. web+ajfsp:
	link = strings.TrimPrefix(link, "web+ajfsp:")
	link = strings.TrimPrefix(link, "web+ajlink:")

	if !strings.HasPrefix(link, "ajfsp://") {
		http.Error(w, "Ungültiger appleJuice Link", http.StatusBadRequest)
		return
	}

	payload := map[string]string{"link": link}
	jsonBody, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		fmt.Sprintf("%s/api/links", h.gatewayURL),
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.gatewayToken)
	addNativeHeader(r, req)

	resp, err := linksGatewayClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Gateway nicht erreichbar: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		http.Error(w, fmt.Sprintf("Gateway-Fehler: HTTP %d", resp.StatusCode), resp.StatusCode)
		return
	}

	if r.Method == http.MethodGet {
		// Protokoll-Handler Redirect
		http.Redirect(w, r, "/?added=true", http.StatusSeeOther)
		return
	}

	w.Header().Set("HX-Trigger", "refreshDownloads")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Download hinzugefügt")
}
