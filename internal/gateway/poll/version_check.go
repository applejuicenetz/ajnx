package poll

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/applejuicenetz/ajnx/internal/domain"
)

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/applejuicenetz/ajnx/releases/latest")
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

	// "v1.0.0" -> "1.0.0"
	return strings.TrimPrefix(release.TagName, "v"), nil
}

func checkUpdate(current string) domain.VersionInfo {
	info := domain.VersionInfo{
		Current: current,
		Latest:  current,
	}

	latest, err := fetchLatestVersion()
	if err == nil && latest != "" {
		info.Latest = latest
		if latest != current {
			info.Available = true
		}
	}

	return info
}
