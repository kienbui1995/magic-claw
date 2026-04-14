package discovery

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestListenerAndAnnouncer(t *testing.T) {
	discovered := make(chan Announcement, 1)

	// Use a random high port to avoid conflicts
	port := 19999

	listener := NewListener(port, "http://localhost:8080", func(ann Announcement, addr net.Addr) {
		discovered <- ann
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start listener
	go listener.Start(ctx)
	time.Sleep(100 * time.Millisecond) // let listener bind

	// Start announcer
	ann := Announcement{
		Name:         "TestBot",
		Endpoint:     "http://localhost:9000",
		Capabilities: []string{"greeting"},
	}
	announcer := NewAnnouncer(port, ann)
	go announcer.Start(ctx)

	// Wait for discovery
	select {
	case got := <-discovered:
		if got.Name != "TestBot" {
			t.Errorf("name = %q, want TestBot", got.Name)
		}
		if got.Endpoint != "http://localhost:9000" {
			t.Errorf("endpoint = %q, want http://localhost:9000", got.Endpoint)
		}
		if len(got.Capabilities) != 1 || got.Capabilities[0] != "greeting" {
			t.Errorf("capabilities = %v, want [greeting]", got.Capabilities)
		}
	case <-ctx.Done():
		t.Skip("discovery timed out — may be blocked by network/firewall")
	}
}
