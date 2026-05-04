package domain

import "time"

// Server repräsentiert einen appleJuice Server.
type Server struct {
	ID        string
	Name      string
	Host      string
	Port      int
	LastSeen  time.Time
	Connected bool
}
