package benchmarks

import (
	"fmt"
	"io"
	"log"
	"sync"
	"testing"

	"github.com/kienbui1995/magic/core/internal/events"
	"github.com/kienbui1995/magic/core/internal/protocol"
	"github.com/kienbui1995/magic/core/internal/registry"
	"github.com/kienbui1995/magic/core/internal/store"
)

// suppressRegistryLogs silences log output for registry benchmarks.
func suppressRegistryLogs(b *testing.B) {
	b.Helper()
	orig := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(orig) })
}

// newRegistryStack creates a fresh bus + store + registry.
// Uses a large event buffer to prevent buffer saturation during high-throughput benchmarks.
func newRegistryStack() (*registry.Registry, func()) {
	bus := events.NewBusWithConfig(64, 1<<20)
	s := store.NewMemoryStore()
	reg := registry.New(s, bus)
	cleanup := func() { bus.Stop() }
	return reg, cleanup
}

// registerPayload returns a minimal RegisterPayload for worker i.
func registerPayload(i int) protocol.RegisterPayload {
	return protocol.RegisterPayload{
		Name: fmt.Sprintf("worker-%d", i),
		Capabilities: []protocol.Capability{
			{Name: "text-gen", EstCostPerCall: 0.001},
		},
		Endpoint: protocol.Endpoint{Type: "http", URL: fmt.Sprintf("http://worker-%d:8080", i)},
		Limits:   protocol.WorkerLimits{MaxConcurrentTasks: 10},
	}
}

// BenchmarkWorkerRegistration measures sequential worker registration throughput.
func BenchmarkWorkerRegistration(b *testing.B) {
	suppressRegistryLogs(b)
	reg, cleanup := newRegistryStack()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := reg.Register(registerPayload(i)); err != nil {
			b.Fatalf("Register failed: %v", err)
		}
	}
}

// BenchmarkWorkerRegistration_Parallel measures concurrent worker registration throughput.
func BenchmarkWorkerRegistration_Parallel(b *testing.B) {
	suppressRegistryLogs(b)
	reg, cleanup := newRegistryStack()
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if _, err := reg.Register(registerPayload(i)); err != nil {
				b.Errorf("Register failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkHeartbeat_100Workers simulates 100 concurrent heartbeats each iteration.
func BenchmarkHeartbeat_100Workers(b *testing.B) {
	suppressRegistryLogs(b)
	reg, cleanup := newRegistryStack()
	defer cleanup()

	const n = 100
	workerIDs := make([]string, n)
	for i := 0; i < n; i++ {
		w, err := reg.Register(registerPayload(i))
		if err != nil {
			b.Fatalf("Register failed: %v", err)
		}
		workerIDs[i] = w.ID
	}

	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		var wg sync.WaitGroup
		wg.Add(n)
		for _, id := range workerIDs {
			id := id
			go func() {
				defer wg.Done()
				reg.Heartbeat(protocol.HeartbeatPayload{ //nolint:errcheck
					WorkerID:    id,
					CurrentLoad: 1,
					Status:      protocol.StatusActive,
				})
			}()
		}
		wg.Wait()
	}
}

// BenchmarkHeartbeat_1000Workers simulates 1000 concurrent heartbeats each iteration.
func BenchmarkHeartbeat_1000Workers(b *testing.B) {
	suppressRegistryLogs(b)
	reg, cleanup := newRegistryStack()
	defer cleanup()

	const n = 1000
	workerIDs := make([]string, n)
	for i := 0; i < n; i++ {
		w, err := reg.Register(registerPayload(i))
		if err != nil {
			b.Fatalf("Register failed: %v", err)
		}
		workerIDs[i] = w.ID
	}

	b.ResetTimer()
	for iter := 0; iter < b.N; iter++ {
		var wg sync.WaitGroup
		wg.Add(n)
		for _, id := range workerIDs {
			id := id
			go func() {
				defer wg.Done()
				reg.Heartbeat(protocol.HeartbeatPayload{ //nolint:errcheck
					WorkerID:    id,
					CurrentLoad: 1,
					Status:      protocol.StatusActive,
				})
			}()
		}
		wg.Wait()
	}
}

// BenchmarkWorkerLookup measures capability-based worker lookup in a 1000-worker registry.
func BenchmarkWorkerLookup(b *testing.B) {
	suppressRegistryLogs(b)
	reg, cleanup := newRegistryStack()
	defer cleanup()

	const n = 1000
	for i := 0; i < n; i++ {
		cap := "text-gen"
		if i%3 == 0 {
			cap = "image-gen"
		}
		if _, err := reg.Register(protocol.RegisterPayload{
			Name: fmt.Sprintf("worker-%d", i),
			Capabilities: []protocol.Capability{
				{Name: cap, EstCostPerCall: 0.001},
			},
			Endpoint: protocol.Endpoint{Type: "http", URL: fmt.Sprintf("http://worker-%d:8080", i)},
			Limits:   protocol.WorkerLimits{MaxConcurrentTasks: 10},
		}); err != nil {
			b.Fatalf("Register failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workers := reg.FindByCapability("text-gen")
		if len(workers) == 0 {
			b.Fatal("expected workers with text-gen capability")
		}
	}
}
