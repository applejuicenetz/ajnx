package domain

import "time"

// Information enthält globale Status-Informationen des Cores.
type Information struct {
	Core    CoreInfo
	Network NetworkInfo
	Session SessionInfo
	Version VersionInfo
}

// CoreInfo enthält Basis-Infos über den laufenden Core.
type CoreInfo struct {
	Version string
	OS      OSType
}

type OSType int

const (
	OSUnknown   OSType = 0
	OSWindows   OSType = 1
	OSLinux     OSType = 2
	OSMacintosh OSType = 3
	OSSolaris   OSType = 4
	OSOS2       OSType = 5
	OSFreeBSD   OSType = 6
	OSNetWare   OSType = 7
)

func (o OSType) String() string {
	switch o {
	case OSWindows:
		return "Windows"
	case OSLinux:
		return "Linux"
	case OSMacintosh:
		return "macOS"
	case OSSolaris:
		return "Solaris"
	case OSOS2:
		return "OS/2"
	case OSFreeBSD:
		return "FreeBSD"
	case OSNetWare:
		return "NetWare"
	default:
		return "Unknown"
	}
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
	ShareFiles         int64
	ShareSize          int64
	ActiveDownloads    int
	ActiveUploads      int
}

