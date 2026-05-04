package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/applejuicenetz/ajnx/internal/ui/view"
)

const (
	CurrentVersion = "v0.1.0"
	GitHubRepo     = "appleJuiceNetz/ajnx"
)

type AboutHandler struct {
	nickname string
}

func NewAboutHandler(nickname string) *AboutHandler {
	return &AboutHandler{nickname: nickname}
}

func (h *AboutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v := view.VersionInfo{
		Current: CurrentVersion,
		Latest:  "Lädt...", // Initialer Status
	}

	// In einer echten Implementierung würde man das cachen, 
	// hier holen wir es live für die Demo.
	latest, _ := fetchLatestVersion()
	if latest != "" {
		v.Latest = latest
		v.IsUpdate = isNewer(CurrentVersion, latest)
	} else {
		v.Latest = CurrentVersion // Fallback
	}

	view.AboutPage(v, h.nickname).Render(r.Context(), w)
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/appleJuiceNetz/ajnx/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func isNewer(current, latest string) bool {
	c := strings.TrimPrefix(current, "v")
	l := strings.TrimPrefix(latest, "v")
	return c < l // Sehr simpler Vergleich für diese Demo
}

type HelpHandler struct {
	nickname string
}

func NewHelpHandler(nickname string) *HelpHandler {
	return &HelpHandler{nickname: nickname}
}

func (h *HelpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	view.HelpPage(h.nickname).Render(r.Context(), w)
}
