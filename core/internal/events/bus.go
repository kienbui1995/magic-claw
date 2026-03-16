package events

import (
	"sync"
	"time"
)

type Event struct {
	Type      string         `json:"type"`
	Source    string         `json:"source"`
	Payload   map[string]any `json:"payload,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Severity  string         `json:"severity"`
}

type Handler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

func (b *Bus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *Bus) Publish(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	if e.Severity == "" {
		e.Severity = "info"
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, h := range b.handlers[e.Type] {
		go h(e)
	}
	if e.Type != "*" {
		for _, h := range b.handlers["*"] {
			go h(e)
		}
	}
}
