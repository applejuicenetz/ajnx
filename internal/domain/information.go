package domain

import "time"

// Information enthält globale Status-Informationen des Cores.
type Information struct {
	Core    CoreInfo
	Network NetworkInfo
	Session SessionInfo
}

// CoreInfo enthält Basis-Infos über den laufenden Core.
type CoreInfo struct {
	Version string
	System  OSType
}

// NetworkInfo enthält Statistiken über das appleJuice Netzwerk.
type NetworkInfo struct {
	Users             int64
	Files             int64
	FilesizeMb        int64
	Firewalled        bool
	IP                string
	ConnectedWithServer string
	ConnectedSince    time.Time
}

// SessionInfo enthält Statistiken der aktuellen Laufzeit.
type SessionInfo struct {
	UploadBytes        int64
	DownloadBytes      int64
	Credits            int64
	UploadSpeedBps     int64
	DownloadSpeedBps   int64
	OpenConnections    int
	MaxUploadPositions int
}

// ShareDir repräsentiert ein freigegebenes Verzeichnis im Core.
type ShareDir struct {
	Path           string
	WithSubfolders bool
}
