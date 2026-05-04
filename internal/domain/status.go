package domain

// DownloadStatus repräsentiert den Status eines Downloads im Core.
type DownloadStatus int

const (
	DownloadSearching    DownloadStatus = 0
	DownloadDiskFull     DownloadStatus = 1
	DownloadFinishing    DownloadStatus = 12
	DownloadFinishError  DownloadStatus = 13
	DownloadFinished     DownloadStatus = 14
	DownloadCanceling    DownloadStatus = 15
	DownloadCreatingData DownloadStatus = 16
	DownloadCanceled     DownloadStatus = 17
	DownloadPaused       DownloadStatus = 18
)

// UserStatus repräsentiert den Status einer Quelle (User).
type UserStatus int

const (
	UserUnasked        UserStatus = 1
	UserConnecting     UserStatus = 2
	UserOldVersion     UserStatus = 3
	UserCannotOpen     UserStatus = 4
	UserInQueue        UserStatus = 5
	UserNoUsefulParts  UserStatus = 6
	UserTransferring   UserStatus = 7
	UserNoSpace        UserStatus = 8
	UserCompleted      UserStatus = 9
	UserUnreachable    UserStatus = 11
	UserTryingIndirect UserStatus = 12
	UserPaused         UserStatus = 13
	UserQueueFull      UserStatus = 14
	UserOwnLimit       UserStatus = 15
	UserServerRefused  UserStatus = 16
)

// OSType repräsentiert das Betriebssystem des Core-Hosts oder einer Quelle.
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
