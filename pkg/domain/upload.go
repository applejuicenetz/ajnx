package domain

import "time"

type UploadStatus int

const (
	UploadWaiting      UploadStatus = 2 // In Warteschlange
	UploadTransferring UploadStatus = 1 // Übertragung
	UploadConnecting   UploadStatus = 5 // Verbindungsaufbau
	UploadError        UploadStatus = 7 // Fehler (z.B. kann nicht verbinden)
)

type Upload struct {
	ID            string       `json:"id"`
	Nickname      string       `json:"nickname"`
	Filename      string       `json:"filename"`
	Status        UploadStatus `json:"status"`
	SpeedBps      int64        `json:"speed_bps"`
	UploadedBytes int64        `json:"uploaded_bytes"`
	QueuePosition int          `json:"queue_position"`
	IP            string       `json:"ip"`
	Port          int          `json:"port"`
	Version       string       `json:"version"`
	OS            int          `json:"os"`
	LastActive    time.Time    `json:"last_active"`
}
