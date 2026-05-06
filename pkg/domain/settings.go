package domain

// Settings enthält die Konfiguration des appleJuice Cores.
type Settings struct {
	Nickname       string `json:"Nick"`
	Port           int    `json:"Port"`
	XMLPort        int    `json:"XMLPort"`
	MaxUpload      int64  `json:"MaxUpload"`
	MaxDownload    int64  `json:"MaxDownload"`
	SpeedPerSlot   int64  `json:"SpeedPerSlot"`
	MaxConnections int    `json:"MaxConnections"`
	AutoConnect    bool   `json:"AutoConnect"`
	MaxSources     int    `json:"MaxSourcesPerFile"`
	IncomingDir    string `json:"IncomingDir"`
	TempDir        string `json:"TempDir"`
	PathMappings   map[string]string `json:"PathMappings"`
}
