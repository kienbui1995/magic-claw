package registry

import (
	"time"

	"github.com/kienbui1995/magic/core/internal/events"
	"github.com/kienbui1995/magic/core/internal/protocol"
)

// HeartbeatTimeout is the duration after which a worker is marked offline.
const HeartbeatTimeout = 60 * time.Second

// StartHealthCheck runs a background goroutine that marks workers offline
// if no heartbeat is received. Returns a stop function to cancel it.
func (r *Registry) StartHealthCheck(interval time.Duration) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.checkHealth()
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}

func (r *Registry) checkHealth() {
	workers := r.store.ListWorkers()
	now := time.Now()
	for _, w := range workers {
		if w.Status == protocol.StatusActive && now.Sub(w.LastHeartbeat) > HeartbeatTimeout {
			w.Status = protocol.StatusOffline
			r.store.UpdateWorker(w) //nolint:errcheck
			r.bus.Publish(events.Event{
				Type:     "worker.offline",
				Source:   "registry",
				Severity: "warn",
				Payload:  map[string]any{"worker_id": w.ID, "worker_name": w.Name},
			})
		}
	}
}
