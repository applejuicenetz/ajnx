package setup

import "sync"

// WizardState hält den Fortschritt während des Setups im Arbeitsspeicher.
type WizardState struct {
	Language    string
	CoreHost    string
	CorePort    string
	CoreVersion string
	CorePassword string
	CurrentStep int
}

// SessionStore verwaltet den WizardState für die aktuelle Sitzung.
type SessionStore struct {
	mu    sync.RWMutex
	state *WizardState
}

// NewSessionStore erstellt einen neuen SessionStore mit sinnvollen Standardwerten.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		state: &WizardState{
			Language:    "de",
			CorePort:    "9851",
			CurrentStep: 1,
		},
	}
}

// GetState gibt eine Kopie des aktuellen Zustands zurück (kein Pointer – Data-Race-sicher).
func (s *SessionStore) GetState() WizardState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.state
}

// UpdateState aktualisiert den Zustand.
func (s *SessionStore) UpdateState(fn func(*WizardState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.state)
}

// Clear löscht den Zustand (nach Abschluss).
func (s *SessionStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = &WizardState{Language: "de", CorePort: "9851", CurrentStep: 1}
}
