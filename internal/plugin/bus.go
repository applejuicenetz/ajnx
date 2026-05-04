package plugin

import (
	"sync"
)

// EventBus ist ein einfacher Broadcaster für Ereignisse.
type EventBus struct {
	mu          sync.RWMutex
	subscribers []chan Event
}

// NewBus erstellt einen neuen EventBus.
func NewBus() *EventBus {
	return &EventBus{
		subscribers: make([]chan Event, 0),
	}
}

// Subscribe fügt einen neuen Subscriber hinzu und gibt den Channel zurück.
func (b *EventBus) Subscribe() chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 100) // Puffer, um Blockaden zu vermeiden
	b.subscribers = append(b.subscribers, ch)
	return ch
}

// Publish sendet ein Ereignis an alle Subscriber.
func (b *EventBus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		// Nicht-blockierendes Senden, falls ein Plugin zu langsam ist
		select {
		case ch <- e:
		default:
			// Puffer voll – Ereignis für diesen Subscriber verworfen
		}
	}
}
