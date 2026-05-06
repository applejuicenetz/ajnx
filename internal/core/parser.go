package core

import (
	"encoding/xml"
	"io"
)

// DecodeXML ist ein Hilfs-Wrapper um XML-Daten in eine Struktur zu dekodieren.
func DecodeXML(r io.Reader, v interface{}) error {
	decoder := xml.NewDecoder(r)
	return decoder.Decode(v)
}

// XML-Strukturen für die Core-Antworten (Mapping auf Domain-Models folgt)

// XMLGeneralInformation für information.xml
type XMLGeneralInformation struct {
	XMLName xml.Name `xml:"applejuice"`
	Version string   `xml:"generalinformation>version"`
	System  string   `xml:"generalinformation>system"`
}

// XMLModified für modified.xml
type XMLModified struct {
	XMLName     xml.Name        `xml:"applejuice"`
	Information *XMLInformation `xml:"information"`
	NetworkInfo *XMLNetworkInfo `xml:"networkinfo"`
	Downloads   []XMLDownload   `xml:"download"`
	Servers     []XMLServer     `xml:"server"`
	Users       []XMLSource     `xml:"user"`
	Searches    []XMLSearch     `xml:"search"`
	SearchEntries []XMLSearchEntry `xml:"searchentry"`
}

type XMLSearch struct {
	ID          string           `xml:"id,attr"`
	SearchText  string           `xml:"searchtext,attr"`
	FoundFiles  int              `xml:"foundfiles,attr"`
	SumSearches int              `xml:"sumsearches,attr"`
	Running     string           `xml:"running,attr"` // "true" / "false"
}

type XMLSearchEntry struct {
	ID       string              `xml:"id,attr"`
	SearchID string              `xml:"searchid,attr"`
	Hash     string              `xml:"checksum,attr"`
	Size     int64               `xml:"size,attr"`
	Filenames []XMLSearchFilename `xml:"filename"`
}

type XMLSearchFilename struct {
	Name  string `xml:"name,attr"`
	Count int    `xml:"user,attr"`
}



type XMLInformation struct {
	Credits            int64 `xml:"credits,attr"`
	SessionUpload      int64 `xml:"sessionupload,attr"`
	SessionDownload    int64 `xml:"sessiondownload,attr"`
	UploadSpeed        int64 `xml:"uploadspeed,attr"`
	DownloadSpeed      int64 `xml:"downloadspeed,attr"`
	OpenConnections    int   `xml:"openconnections,attr"`
	MaxUploadPositions int   `xml:"maxuploadpositions,attr"`
	ShareFiles         int64 `xml:"sharefiles,attr"`
	ShareSize          int64 `xml:"sharesize,attr"`
}

type XMLNetworkInfo struct {
	Users               int64  `xml:"users,attr"`
	Files               int64  `xml:"files,attr"`
	Filesize            string `xml:"filesize,attr"`
	Firewalled          string `xml:"firewalled,attr"`
	IP                  string `xml:"ip,attr"`
	ConnectedWithServer string `xml:"connectedwithserverid,attr"`
	ConnectedSince      int64  `xml:"connectedsince,attr"`
}

type XMLDownload struct {
	ID                  string      `xml:"id,attr"`
	Hash                string      `xml:"hash,attr"`
	Size                int64       `xml:"size,attr"`
	Status              int         `xml:"status,attr"`
	Filename            string      `xml:"filename,attr"`
	TargetDirectory     string      `xml:"targetdirectory,attr"`
	PowerDownload       int         `xml:"powerdownload,attr"`
	Ready               int64       `xml:"ready,attr"`
	TemporaryFileNumber int         `xml:"temporaryfilenumber,attr"`
	Sources             []XMLSource `xml:"-"`
}

type XMLSource struct {
	ID              string `xml:"id,attr"`
	DownloadID      string `xml:"downloadid,attr"`
	Nickname        string `xml:"nickname,attr"`
	Status          int    `xml:"status,attr"`
	IP              string `xml:"ip,attr"`
	Port            int    `xml:"port,attr"`
	Version         string `xml:"version,attr"`
	OperatingSystem int    `xml:"operatingsystem,attr"`
	QueuePosition   int    `xml:"queueposition,attr"`
	Speed           int64  `xml:"speed,attr"`
	Downloaded      int64  `xml:"downloaded,attr"`
	Filename        string `xml:"filename,attr"`
	Source          int    `xml:"source,attr"`
}

type XMLServer struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	Host      string `xml:"host,attr"`
	Port      int    `xml:"port,attr"`
	LastSeen  int64  `xml:"lastseen,attr"`
	Connected int    `xml:"connected,attr"`
}

type XMLShareList struct {
	XMLName xml.Name   `xml:"shares"`
	Shares  []XMLShare `xml:"share"`
	// Manche Cores schachteln die Liste in ein <sharelist> Tag
	Nested struct {
		Shares []XMLShare `xml:"share"`
	} `xml:"sharelist"`
}

type XMLShare struct {
	ID            string `xml:"id,attr"`
	Hash          string `xml:"checksum,attr"`
	AltHash       string `xml:"hash,attr"` // Alternativ-Tag für manche Core-Versionen
	Size          int64  `xml:"size,attr"`
	Filename      string `xml:"filename,attr"`
	ShortFilename string `xml:"shortfilename,attr"`
	Priority      int    `xml:"priority,attr"`
	LastAsked     int64  `xml:"lastasked,attr"`
	AskCount      int64  `xml:"askcount,attr"`
	SearchCount   int64  `xml:"searchcount,attr"`
}

type XMLDownloadList struct {
	XMLName   xml.Name      `xml:"applejuice"`
	Downloads []XMLDownload `xml:"download"`
}

type XMLServerList struct {
	XMLName xml.Name    `xml:"applejuice"`
	Servers []XMLServer `xml:"server"`
}

type XMLSettings struct {
	XMLName           xml.Name `xml:"settings"`
	Nick              string   `xml:"nick"`
	Port              int      `xml:"port"`
	XMLPort           int      `xml:"xmlport"`
	MaxUpload         int64    `xml:"maxupload"`
	MaxDownload       int64    `xml:"maxdownload"`
	SpeedPerSlot      int64    `xml:"speedperslot"`
	MaxConnections    int      `xml:"maxconnections"`
	AutoConnect       string   `xml:"autoconnect"` // "True"/"False"
	MaxSourcesPerFile int      `xml:"maxsourcesperfile"`
	IncomingDir       string   `xml:"incomingdirectory"`
	TempDir           string   `xml:"temporarydirectory"`
}

type XMLSession struct {
	XMLName xml.Name `xml:"applejuice"`
	Session struct {
		ID string `xml:"id,attr"`
	} `xml:"session"`
}
