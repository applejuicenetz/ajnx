package core

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/applejuicenetz/ajnx/internal/domain"
	"github.com/applejuicenetz/ajnx/internal/plugin"
)


// XMLCoreClient implementiert die Schnittstelle zum klassischen appleJuice Core.
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
	mod, err := c.fetchModified(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	static := c.staticInfo
	c.mu.RUnlock()

	return mapInformation(mod, static), nil
}

// FullUpdate holt alle wichtigen Daten in einem Request.
func (c *XMLCoreClient) FullUpdate(ctx context.Context) (*domain.Information, []domain.Download, []domain.Upload, error) {
	mod, err := c.fetchModified(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	c.mu.RLock()
	static := c.staticInfo
	c.mu.RUnlock()

	return mapInformation(mod, static), mapDownloads(mod), mapUploads(mod), nil
}

func (c *XMLCoreClient) fetchModified(ctx context.Context) (*XMLModified, error) {
	resp, err := c.doRequest(ctx, "/xml/modified.xml?timestamp=0")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mod XMLModified
	if err := DecodeXML(resp.Body, &mod); err != nil {
		return nil, err
	}
	return &mod, nil
}

func mapInformation(mod *XMLModified, staticInfo *XMLGeneralInformation) *domain.Information {
	info := &domain.Information{}
	if staticInfo != nil {
		info.Core = domain.CoreInfo{
			Version: staticInfo.General.Version,
			OS:      parseOSType(staticInfo.General.System),
		}
	}

	if mod.NetworkInfo != nil {
		serverDisplay := mod.NetworkInfo.ConnectedWithServer
		for _, s := range mod.Servers {
			if s.ID == mod.NetworkInfo.ConnectedWithServer && s.Name != "" {
				serverDisplay = s.Name
				break
			}
		}

		info.Network = domain.NetworkInfo{
			Users:               mod.NetworkInfo.Users,
			Files:               mod.NetworkInfo.Files,
			Firewalled:          mod.NetworkInfo.Firewalled == "true",
			IP:                  mod.NetworkInfo.IP,
			ConnectedWithServer: serverDisplay,
			ConnectedSince:      time.Unix(mod.NetworkInfo.ConnectedSince/1000, 0),
		}
		if mod.NetworkInfo.Filesize != "" {
			if fs, err := strconv.ParseFloat(mod.NetworkInfo.Filesize, 64); err == nil {
				info.Network.FilesizeMb = int64(fs)
			}
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
			ShareFiles:         mod.Information.ShareFiles,
			ShareSize:          mod.Information.ShareSize,
		}
	}

	return info
}

// Downloads fragt die Liste der aktuellen Downloads ab.
func (c *XMLCoreClient) Downloads(ctx context.Context) ([]domain.Download, error) {
	mod, err := c.fetchModified(ctx)
	if err != nil {
		return nil, err
	}
	return mapDownloads(mod), nil
}

func mapDownloads(mod *XMLModified) []domain.Download {
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

	return downloads
}

// Servers fragt die Serverliste ab.
func (c *XMLCoreClient) Servers(ctx context.Context) ([]domain.Server, error) {
	mod, err := c.fetchModified(ctx)
	if err != nil {
		return nil, err
	}
	return mapServers(mod), nil
}

func mapServers(mod *XMLModified) []domain.Server {
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
	return servers
}

// Uploads fragt die aktuelle Upload-Liste ab.
func (c *XMLCoreClient) Uploads(ctx context.Context) ([]domain.Upload, error) {
	mod, err := c.fetchModified(ctx)
	if err != nil {
		return nil, err
	}
	return mapUploads(mod), nil
}

func mapUploads(mod *XMLModified) []domain.Upload {
	var uploads []domain.Upload
	for _, u := range mod.Users {
		// In appleJuice XML-API sind User mit DownloadID == "0" Uploads.
		// User mit DownloadID != "0" sind Quellen für unsere eigenen Downloads.
		if u.DownloadID != "0" {
			continue
		}

		status := domain.UploadWaiting
		// Mapping laut appleJuice Spezifikation: 1=Active, 2=Queue, 5/6=Connecting, 7=Error
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

	return uploads
}

func (c *XMLCoreClient) PauseDownload(ctx context.Context, ids ...string) error {
	params := buildMultiIDParam("id", ids)
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

func (c *XMLCoreClient) SetPowerDownload(ctx context.Context, id string, powerDownload float64) error {
	// CoreValue = (UIValue * 10) - 10
	coreValue := int((powerDownload * 10) - 10)
	if coreValue < -10 {
		coreValue = -10
	}
	if coreValue > 1000 { // appleJuice Limit ist meist 100, aber wir erlauben etwas mehr
		coreValue = 1000
	}

	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/setpowerdownload?id=%s&powerdownload=%d", id, coreValue))
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

	// Slots leeren falls wir weniger haben als vorher (bis zu 20 AJ-Standard)
	for i := len(shares); i < 20; i++ {
		idx := i + 1
		v.Set(fmt.Sprintf("sharedirectory%d", idx), "")
		v.Set(fmt.Sprintf("sharesub%d", idx), "False")
	}

	resp, err := c.doRequest(ctx, fmt.Sprintf("/function/setsettings?%s", v.Encode()))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Shares fragt die Liste der freigegebenen Dateien ab.
func (c *XMLCoreClient) Shares(ctx context.Context) ([]domain.Share, error) {
	resp, err := c.doRequest(ctx, "/xml/share.xml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list XMLShareList
	if err := DecodeXML(resp.Body, &list); err != nil {
		return nil, err
	}

	// Beide Listen kombinieren (direkt und geschachtelt)
	rawShares := list.Shares
	if len(list.Nested.Shares) > 0 {
		rawShares = append(rawShares, list.Nested.Shares...)
	}

	shares := make([]domain.Share, len(rawShares))
	for i, s := range rawShares {
		hash := s.Hash
		if hash == "" {
			hash = s.AltHash
		}

		shares[i] = domain.Share{
			ID:            s.ID,
			Hash:          hash,
			Size:          s.Size,
			Filename:      s.Filename,
			ShortFilename: s.ShortFilename,
			Priority:      s.Priority,
		}
	}

	return shares, nil
}

func (c *XMLCoreClient) GetShareDirs(ctx context.Context) ([]domain.ShareDir, error) {
	resp, err := c.doRequest(ctx, "/xml/settings.xml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		XMLName xml.Name `xml:"applejuice"`
		Share   struct {
			Directories []struct {
				Path string `xml:"name"`
				Mode string `xml:"sharemode"`
			} `xml:"directory"`
		} `xml:"share"`
	}

	if err := DecodeXML(resp.Body, &res); err != nil {
		return nil, err
	}

	var shares []domain.ShareDir
	for _, d := range res.Share.Directories {
		shares = append(shares, domain.ShareDir{
			Path:           d.Path,
			WithSubfolders: strings.ToLower(d.Mode) == "subdirectory",
		})
	}

	return shares, nil
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
			sb.WriteString(fmt.Sprintf("&%s%d=%s", firstKey, i, id))
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
