package plugin

import (
	"fmt"
	"log"
)

// Plugin definiert die Schnittstelle für AJNX-Erweiterungen.
type Plugin interface {
	ID() string
	Start(bus *EventBus) error
	Stop() error
}

// Manager verwaltet den Lebenszyklus aller Plugins.
type Manager struct {
	bus     *EventBus
	plugins []Plugin
}

func NewManager(bus *EventBus) *Manager {
	return &Manager{
		bus:     bus,
		plugins: make([]Plugin, 0),
	}
}

// Register fügt ein Plugin hinzu.
func (m *Manager) Register(p Plugin) {
	m.plugins = append(m.plugins, p)
}

// StartAll startet alle registrierten Plugins.
func (m *Manager) StartAll() error {
	for _, p := range m.plugins {
		log.Printf("Plugin [%s] wird gestartet...", p.ID())
		if err := p.Start(m.bus); err != nil {
			return fmt.Errorf("plugin %s konnte nicht gestartet werden: %v", p.ID(), err)
		}
	}
	return nil
}

// StopAll stoppt alle registrierten Plugins.
func (m *Manager) StopAll() {
	for _, p := range m.plugins {
		log.Printf("Plugin [%s] wird gestoppt...", p.ID())
		p.Stop()
	}
}

// Bus gibt den EventBus zurück.
func (m *Manager) Bus() *EventBus {
	return m.bus
}
