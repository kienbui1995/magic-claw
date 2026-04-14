// Package discovery provides UDP broadcast-based worker auto-discovery for local development.
//
// Workers broadcast their presence on a UDP port. The MagiC server listens
// and auto-registers discovered workers. This is for local dev only —
// production should use explicit registration via the REST API.
//
// Protocol: Workers send JSON {"name":"Bot","endpoint":"http://host:port","capabilities":["cap1"]}
// Server responds with {"ack":true,"server":"http://host:port"} on discovery.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

const (
	DefaultPort    = 9999
	BroadcastAddr  = "255.255.255.255"
	MaxPacketSize  = 4096
	AnnounceInterval = 5 * time.Second
)

// Announcement is what workers broadcast.
type Announcement struct {
	Name         string   `json:"name"`
	Endpoint     string   `json:"endpoint"`
	Capabilities []string `json:"capabilities"`
}

// Ack is what the server responds with.
type Ack struct {
	Ack    bool   `json:"ack"`
	Server string `json:"server"`
}

// OnDiscover is called when a new worker is discovered.
type OnDiscover func(ann Announcement, addr net.Addr)

// Listener listens for worker broadcast announcements.
type Listener struct {
	port       int
	serverURL  string
	onDiscover OnDiscover
}

// NewListener creates a discovery listener.
func NewListener(port int, serverURL string, onDiscover OnDiscover) *Listener {
	return &Listener{port: port, serverURL: serverURL, onDiscover: onDiscover}
}

// Start begins listening for worker announcements. Blocks until ctx is cancelled.
func (l *Listener) Start(ctx context.Context) error {
	addr := &net.UDPAddr{Port: l.port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("discovery listen: %w", err)
	}
	defer conn.Close()

	log.Printf("[discovery] listening on UDP :%d", l.port)

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	buf := make([]byte, MaxPacketSize)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil // context cancelled
			}
			continue
		}

		var ann Announcement
		if err := json.Unmarshal(buf[:n], &ann); err != nil {
			continue
		}
		if ann.Name == "" || ann.Endpoint == "" {
			continue
		}

		// Send ack
		ack, _ := json.Marshal(Ack{Ack: true, Server: l.serverURL})
		conn.WriteToUDP(ack, remoteAddr) //nolint:errcheck

		if l.onDiscover != nil {
			l.onDiscover(ann, remoteAddr)
		}
	}
}

// Announcer broadcasts worker presence for auto-discovery.
type Announcer struct {
	port int
	ann  Announcement
}

// NewAnnouncer creates a worker announcer.
func NewAnnouncer(port int, ann Announcement) *Announcer {
	return &Announcer{port: port, ann: ann}
}

// Start begins broadcasting. Blocks until ctx is cancelled.
func (a *Announcer) Start(ctx context.Context) error {
	addr := &net.UDPAddr{IP: net.ParseIP(BroadcastAddr), Port: a.port}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("discovery announce: %w", err)
	}
	defer conn.Close()

	data, _ := json.Marshal(a.ann)
	ticker := time.NewTicker(AnnounceInterval)
	defer ticker.Stop()

	// Send immediately
	conn.Write(data) //nolint:errcheck

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			conn.Write(data) //nolint:errcheck
		}
	}
}
