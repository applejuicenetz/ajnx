package domain

// Download repräsentiert einen Download-Eintrag.
type Download struct {
	ID                  string
	ShareID             string
	Hash                string
	Size                int64
	Status              DownloadStatus
	Filename            string
	TargetDirectory     string
	PowerDownload       float64
	ReadyBytes          int64
	TemporaryFileNumber int
	Sources             []Source

	// Berechnete Felder
	ProgressPercent float64
	SpeedBps        int64
	EtaSeconds      int64
}

// Source repräsentiert eine Quelle für einen Download.
type Source struct {
	ID             string
	Nickname       string
	Status         UserStatus
	IP             string
	Port           int
	Version        string
	OperatingSystem OSType
	QueuePosition  int
	SpeedBps       int64
	DownloadedBytes int64
}
