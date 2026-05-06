package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session repräsentiert eine aktive Benutzer-Sitzung im Gateway.
type Session struct {
	Token     string
	ExpiresAt time.Time
	CoreHost  string
	Password  string // MD5 des Core-Passworts
}

// SessionManager verwaltet alle aktiven Sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionManager erstellt einen neuen SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession erstellt eine neue Session und gibt den Token zurück.
func (m *SessionManager) CreateSession(host, password string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[token] = &Session{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CoreHost:  host,
		Password:  password,
	}

	return token, nil
}

// ValidateToken prüft, ob ein Token gültig ist und verlängert die Session (Sliding Window).
func (m *SessionManager) ValidateToken(token string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[token]
	if !ok {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(m.sessions, token)
		return nil, false
	}

	// Sliding Window: Ablaufzeit verlängern
	session.ExpiresAt = time.Now().Add(24 * time.Hour)
	return session, true
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
