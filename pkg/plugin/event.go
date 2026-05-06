package plugin

import "time"

// EventType definiert den Typ eines Ereignisses.
type EventType string

const (
	EventCoreRequest  EventType = "core.request"
	EventCoreResponse EventType = "core.response"
	EventCoreLog      EventType = "core.log"
)

// Event stellt ein einzelnes Ereignis im System dar.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// CoreRequestData enthält Details zu einer API-Anfrage an den Core.
type CoreRequestData struct {
	Path   string
	Method string
}

// CoreResponseData enthält Details zu einer API-Antwort vom Core.
type CoreResponseData struct {
	Path string
	Body []byte
}

// CoreLogData enthält eine Log-Zeile vom Core.
type CoreLogData struct {
	Line string
}
