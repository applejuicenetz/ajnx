package gateway

import (
	"sync"
	"github.com/applejuicenetz/ajnx/pkg/domain"
)

// State hält den aktuellen Zustand des Cores im Arbeitsspeicher des Gateways.
type State struct {
	mu          sync.RWMutex
	Information *domain.Information
	Downloads   []domain.Download
	Uploads     []domain.Upload
	Servers     []domain.Server
	LastError   string
}

// NewState erstellt einen neuen leeren State.
func NewState() *State {
	return &State{
		Downloads: []domain.Download{},
		Uploads:   []domain.Upload{},
		Servers:   []domain.Server{},
	}
}

// UpdateInformation aktualisiert die globalen Infos.
func (s *State) UpdateInformation(info *domain.Information) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Information = info
}

// GetInformation gibt eine Kopie der Infos zurück.
func (s *State) GetInformation() *domain.Information {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Information
}

// UpdateDownloads aktualisiert die Download-Liste.
func (s *State) UpdateDownloads(downloads []domain.Download) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Downloads = downloads
}

// GetDownloads gibt die Download-Liste zurück.
func (s *State) GetDownloads() []domain.Download {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Downloads
}

// UpdateServers aktualisiert die Server-Liste.
func (s *State) UpdateServers(servers []domain.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Servers = servers
}

// GetServers gibt die Server-Liste zurück.
func (s *State) GetServers() []domain.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Servers
}

// UpdateUploads aktualisiert die Upload-Liste.
func (s *State) UpdateUploads(uploads []domain.Upload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Uploads = uploads
}

// GetUploads gibt die Upload-Liste zurück.
func (s *State) GetUploads() []domain.Upload {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Uploads
}

// UpdateError setzt den letzten Polling-Fehler.
func (s *State) UpdateError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.LastError = ""
	} else {
		s.LastError = err.Error()
	}
}

// GetLastError gibt den letzten Fehler als String zurück.
func (s *State) GetLastError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastError
}
