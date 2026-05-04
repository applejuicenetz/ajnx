package core

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/applejuicenetz/ajnx/internal/domain"
	"github.com/applejuicenetz/ajnx/internal/plugin"
)


// XMLCoreClient ist die Implementierung des CoreClient-Interfaces für die XML-Schnittstelle.
type XMLCoreClient struct {
	mu          sync.RWMutex
	baseURL     string
	password    string // MD5 Hash des Passworts
	sessionID   string
	httpClient  *http.Client
	eventBus    *plugin.EventBus
	staticInfo  *XMLGeneralInformation // einmalig beim Login gecacht
}

// NewXMLCoreClient erstellt einen neuen XML-basierten Client.
func NewXMLCoreClient(baseURL, password string) *XMLCoreClient {
	return &XMLCoreClient{
		baseURL:  baseURL,
		password: md5hex(password),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// NewXMLCoreClientPrehashed erstellt einen Client mit einem bereits gehashten Passwort.
func NewXMLCoreClientPrehashed(baseURL, passwordHash string) *XMLCoreClient {
	return &XMLCoreClient{
		baseURL:  baseURL,
		password: strings.ToLower(passwordHash),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// UpdateConfig aktualisiert die Verbindungsparameter zur Laufzeit.
func (c *XMLCoreClient) UpdateConfig(baseURL, passwordHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = baseURL
	c.password = strings.ToLower(passwordHash)
	c.sessionID = "" // Session zurücksetzen
	c.staticInfo = nil
}

// md5hex berechnet den MD5-Hash eines Strings.
func md5hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// Information fragt Live-Stats vom Core ab.
func (c *XMLCoreClient) Information(ctx context.Context) (*domain.Information, error) {
	resp, err := c.doRequest(ctx, "/xml/modified.xml?timestamp=0")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mod XMLModified
	if err := DecodeXML(resp.Body, &mod); err != nil {
		return nil, err
	}

	info := &domain.Information{}
	c.mu.RLock()
	if c.staticInfo != nil {
		info.Core = domain.CoreInfo{
			Version: c.staticInfo.General.Version,
			System:  parseOSType(c.staticInfo.General.System),
		}
	}
	c.mu.RUnlock()

	if mod.NetworkInfo != nil {
		serverDisplay := mod.NetworkInfo.ConnectedWithServer
		for _, s := range mod.Servers {
			if s.ID == mod.NetworkInfo.ConnectedWithServer && s.Name != "" {
				serverDisplay = s.Name
				break
			}
		}

		info.Network = domain.NetworkInfo{
			Users:             mod.NetworkInfo.Users,
			Files:             mod.NetworkInfo.Files,
			Firewalled:        mod.NetworkInfo.Firewalled == "true",
			IP:                mod.NetworkInfo.IP,
			ConnectedWithServer: serverDisplay,
			ConnectedSince:    time.Unix(mod.NetworkInfo.ConnectedSince/1000, 0),
		}
		if mod.NetworkInfo.Filesize != "" {
			var fs float64
			fmt.Sscanf(mod.NetworkInfo.Filesize, "%f", &fs)
			info.Network.FilesizeMb = int64(fs)
		}
	}

	if mod.Information != nil {
		info.Session = domain.SessionInfo{
			UploadBytes:        mod.Information.SessionUpload,
			DownloadBytes:      mod.Information.SessionDownload,
			Credits:            mod.Information.Credits,
			UploadSpeedBps:     mod.Information.UploadSpeed,
			DownloadSpeedBps:   mod.Information.DownloadSpeed,
			OpenConnections:    mod.Information.OpenConnections,
			MaxUploadPositions: mod.Information.MaxUploadPositions,
		}
	}

	return info, nil
}

// Downloads fragt die Liste der aktuellen Downloads ab.
func (c *XMLCoreClient) Downloads(ctx context.Context) ([]domain.Download, error) {
	resp, err := c.doRequest(ctx, "/xml/modified.xml?timestamp=0")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mod XMLModified
	if err := DecodeXML(resp.Body, &mod); err != nil {
		return nil, err
	}

	sourcesByDownload := make(map[string][]domain.Source)
	for _, s := range mod.Users {
		src := domain.Source{
			ID:              s.ID,
			Nickname:        s.Nickname,
			Status:          domain.UserStatus(s.Status),
			IP:              s.IP,
			Port:            s.Port,
			Version:         s.Version,
			OperatingSystem: domain.OSType(s.OperatingSystem),
			QueuePosition:   s.QueuePosition,
			SpeedBps:        s.Speed,
			DownloadedBytes: s.Downloaded,
		}
		sourcesByDownload[s.DownloadID] = append(sourcesByDownload[s.DownloadID], src)
	}

	downloads := make([]domain.Download, len(mod.Downloads))
	for i, d := range mod.Downloads {
		downloads[i] = domain.Download{
			ID:                  d.ID,
			Hash:                d.Hash,
			Size:                d.Size,
			Status:              domain.DownloadStatus(d.Status),
			Filename:            d.Filename,
			TargetDirectory:     d.TargetDirectory,
			PowerDownload:       float64(d.PowerDownload+10) / 10,
			ReadyBytes:          d.Ready,
			TemporaryFileNumber: d.TemporaryFileNumber,
			Sources:             sourcesByDownload[d.ID],
		}

		if d.Size > 0 {
			downloads[i].ProgressPercent = (float64(d.Ready) / float64(d.Size)) * 100
		}

		var totalSpeed int64
		for _, s := range downloads[i].Sources {
			totalSpeed += s.SpeedBps
		}
		downloads[i].SpeedBps = totalSpeed

		if downloads[i].SpeedBps > 0 && d.Size > d.Ready {
			downloads[i].EtaSeconds = (d.Size - d.Ready) / downloads[i].SpeedBps
		}
	}

	return downloads, nil
}

// Servers fragt die Serverliste ab.
func (c *XMLCoreClient) Servers(ctx context.Context) ([]domain.Server, error) {
	resp, err := c.doRequest(ctx, "/xml/modified.xml?timestamp=0")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mod XMLModified
	if err := DecodeXML(resp.Body, &mod); err != nil {
		return nil, err
	}

	servers := make([]domain.Server, len(mod.Servers))
	for i, s := range mod.Servers {
		servers[i] = domain.Server{
			ID:        s.ID,
			Name:      s.Name,
			Host:      s.Host,
			Port:      s.Port,
			LastSeen:  time.Unix(s.LastSeen/1000, 0),
			Connected: s.Connected == 1,
		}
	}

	return servers, nil
}

// Uploads fragt die aktuelle Upload-Liste ab.
func (c *XMLCoreClient) Uploads(ctx context.Context) ([]domain.Upload, error) {
	resp, err := c.doRequest(ctx, "/xml/modified.xml?timestamp=0")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mod XMLModified
	if err := DecodeXML(resp.Body, &mod); err != nil {
		return nil, err
	}

	var uploads []domain.Upload
	for _, u := range mod.Users {
		// In appleJuice XML-API sind User mit DownloadID == "0" Uploads.
		// User mit DownloadID != "0" sind Quellen für unsere eigenen Downloads.
		if u.DownloadID != "0" {
			continue
		}

		status := domain.UploadWaiting
		// Status-Mapping laut appleJuice Spezifikation:
		// 1 = Übertrage (Active)
		// 2 = Warteschlange
		// 5, 6 = Verbindungsversuch
		// 7 = Fehler
		switch u.Status {
		case 1:
			status = domain.UploadTransferring
		case 5, 6:
			status = domain.UploadConnecting
		case 7:
			status = domain.UploadError
		default:
			status = domain.UploadWaiting
		}

		uploads = append(uploads, domain.Upload{
			ID:            u.ID,
			Nickname:      u.Nickname,
			Filename:      u.Filename,
			Status:        status,
			SpeedBps:      u.Speed,
			UploadedBytes: u.Downloaded, // Bei Uploads ist 'Downloaded' das, was wir hochgeladen haben
			QueuePosition: u.QueuePosition,
			IP:            u.IP,
			Port:          u.Port,
			Version:       u.Version,
			OS:            u.OperatingSystem,
			LastActive:    time.Now(), // Core liefert keinen LastActive Timestamp pro User in modified.xml
		})
	}

	return uploads, nil
}

func (c *XMLCoreClient) PauseDownload(ctx context.Context, ids ...string) error {
	params := buildMultiIDParam("Id", ids)
	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/pausedownload?%s", params))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) ResumeDownload(ctx context.Context, ids ...string) error {
	params := buildMultiIDParam("id", ids)
	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/resumedownload?%s", params))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) CancelDownload(ctx context.Context, ids ...string) error {
	params := buildMultiIDParam("id", ids)
	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/canceldownload?%s", params))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) CleanDownloadList(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "/function/cleandownloadlist")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) ConnectServer(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/setserver?id=%s", id))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) DisconnectServer(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "/function/unloadserver")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) ShutdownCore(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "/function/exitcore")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) RemoveServer(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/removeserver?id=%s", id))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) ProcessLink(ctx context.Context, link string) error {
	if unescaped, err := url.QueryUnescape(link); err == nil {
		link = unescaped
	}
	encodedLink := url.QueryEscape(link)
	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/processlink?link=%s&subdir=", encodedLink))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) StartSearch(ctx context.Context, query string) error {
	encodedQuery := strings.ReplaceAll(url.QueryEscape(query), "+", "%20")
	urlStr := fmt.Sprintf("/function/search?search=%s", encodedQuery)
	
	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := c.doRequest(ctx, urlStr)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("suche konnte nicht gestartet werden: %w", lastErr)
}

func (c *XMLCoreClient) Searches(ctx context.Context) ([]domain.Search, error) {
	resp, err := c.doRequest(ctx, "/xml/modified.xml?filter=search&timestamp=0")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mod XMLModified
	if err := DecodeXML(resp.Body, &mod); err != nil {
		return nil, err
	}

	entriesBySearch := make(map[string][]domain.SearchResult)
	for _, e := range mod.SearchEntries {
		names := make([]domain.SearchResultName, len(e.Filenames))
		for k, f := range e.Filenames {
			names[k] = domain.SearchResultName{
				Name:  f.Name,
				Count: f.Count,
			}
		}
		entriesBySearch[e.SearchID] = append(entriesBySearch[e.SearchID], domain.SearchResult{
			ID:       e.ID,
			SearchID: e.SearchID,
			Hash:     e.Hash,
			Size:     e.Size,
			Names:    names,
		})
	}

	searches := make([]domain.Search, len(mod.Searches))
	for i, s := range mod.Searches {
		searches[i] = domain.Search{
			ID:          s.ID,
			Query:       s.SearchText,
			FoundFiles:  s.FoundFiles,
			SumSearches: s.SumSearches,
			Running:     s.Running == "true",
			Results:     entriesBySearch[s.ID],
		}
	}

	return searches, nil
}

func (c *XMLCoreClient) CancelSearch(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/cancelsearch?id=%s", id))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) CheckPassword(ctx context.Context, password string) (bool, error) {
	c.mu.RLock()
	baseURL := c.baseURL
	c.mu.RUnlock()

	urlStr := fmt.Sprintf("%s/xml/settings.xml?password=%s", baseURL, password)
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}

	return resp.StatusCode == http.StatusOK, nil
}

func (c *XMLCoreClient) RefreshSession(ctx context.Context) error {
	c.mu.RLock()
	password := c.password
	c.mu.RUnlock()

	ok, err := c.CheckPassword(ctx, password)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}

	c.mu.Lock()
	c.sessionID = password
	c.mu.Unlock()

	resp, err := c.doRequest(ctx, "/xml/information.xml")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var genInfo XMLGeneralInformation
	if err := DecodeXML(resp.Body, &genInfo); err != nil {
		return err
	}
	
	c.mu.Lock()
	c.staticInfo = &genInfo
	c.mu.Unlock()

	return nil
}

func (c *XMLCoreClient) GetSettings(ctx context.Context) (XMLSettings, error) {
	resp, err := c.doRequest(ctx, "/xml/settings.xml")
	if err != nil {
		return XMLSettings{}, err
	}
	defer resp.Body.Close()

	var settings XMLSettings
	if err := DecodeXML(resp.Body, &settings); err != nil {
		return XMLSettings{}, err
	}
	return settings, nil
}

func (c *XMLCoreClient) UpdateSettings(ctx context.Context, s XMLSettings) error {
	v := url.Values{}
	v.Set("Nickname", s.Nick)
	v.Set("MaxUpload", fmt.Sprintf("%d", s.MaxUpload))
	v.Set("MaxDownload", fmt.Sprintf("%d", s.MaxDownload))
	v.Set("Speedperslot", fmt.Sprintf("%d", s.SpeedPerSlot/1024))
	v.Set("MaxConnections", fmt.Sprintf("%d", s.MaxConnections))
	v.Set("MaxSourcesPerFile", fmt.Sprintf("%d", s.MaxSourcesPerFile))
	v.Set("AutoConnect", s.AutoConnect)
	v.Set("Incomingdirectory", s.IncomingDir)
	v.Set("Temporarydirectory", s.TempDir)
	v.Set("Port", fmt.Sprintf("%d", s.Port))
	v.Set("XMLPort", fmt.Sprintf("%d", s.XMLPort))

	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/setsettings?%s", v.Encode()))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) SetShares(ctx context.Context, shares []domain.ShareDir) error {
	v := url.Values{}
	v.Set("countshares", fmt.Sprintf("%d", len(shares)))
	for i, s := range shares {
		idx := i + 1
		v.Set(fmt.Sprintf("sharedirectory%d", idx), s.Path)
		sub := "False"
		if s.WithSubfolders {
			sub = "True"
		}
		v.Set(fmt.Sprintf("sharesub%d", idx), sub)
	}

	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/setsettings?%s", v.Encode()))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *XMLCoreClient) doRequest(ctx context.Context, path string) (*http.Response, error) {
	c.mu.RLock()
	baseURL := c.baseURL
	password := c.password
	c.mu.RUnlock()

	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	urlStr := fmt.Sprintf("%s%s%spassword=%s", baseURL, path, sep, password)
	log.Printf("[CoreClient] %s (Password: %s...)", path, password[:4])

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core nicht erreichbar (%s): %w", baseURL, err)
	}

	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, ErrForbidden
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("core fehler %d: %s", resp.StatusCode, string(body))
	}

	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(path, "/function/") {
		if strings.HasPrefix(trimmed, "<html") || strings.HasPrefix(trimmed, "<!") {
			return nil, fmt.Errorf("core: HTML statt XML erhalten (Passwort falsch?)")
		}
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

func (c *XMLCoreClient) SetEventBus(bus *plugin.EventBus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventBus = bus
}

func buildMultiIDParam(firstKey string, ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, id := range ids {
		if i == 0 {
			sb.WriteString(fmt.Sprintf("%s=%s", firstKey, id))
		} else {
			sb.WriteString(fmt.Sprintf("&id%d=%s", i, id))
		}
	}
	return sb.String()
}

func parseOSType(s string) domain.OSType {
	switch strings.ToLower(s) {
	case "windows": return domain.OSWindows
	case "linux": return domain.OSLinux
	case "mac", "macintosh", "macos": return domain.OSMacintosh
	case "solaris": return domain.OSSolaris
	case "os2", "os/2": return domain.OSOS2
	case "freebsd": return domain.OSFreeBSD
	case "netware": return domain.OSNetWare
	default: return domain.OSUnknown
	}
}
