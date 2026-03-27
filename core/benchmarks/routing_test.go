package benchmarks

import (
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/kienbui1995/magic/core/internal/events"
	"github.com/kienbui1995/magic/core/internal/protocol"
	"github.com/kienbui1995/magic/core/internal/registry"
	"github.com/kienbui1995/magic/core/internal/router"
	"github.com/kienbui1995/magic/core/internal/store"
)

// newRoutingStack creates a fresh registry+router stack with n registered workers.
// Workers all have the "text-gen" capability with MaxConcurrentTasks set high
// so none are filtered out during routing.
// Uses a large event buffer to prevent buffer saturation during high-throughput benchmarks.
func newRoutingStack(n int) (*router.Router, func()) {
	bus := events.NewBusWithConfig(64, 1<<20)
	s := store.NewMemoryStore()
	reg := registry.New(s, bus)
	rtr := router.New(reg, s, bus)

	for i := 0; i < n; i++ {
		reg.Register(protocol.RegisterPayload{ //nolint:errcheck
			Name: fmt.Sprintf("worker-%d", i),
			Capabilities: []protocol.Capability{
				{
					Name:           "text-gen",
					EstCostPerCall: float64(i+1) * 0.001,
					AvgResponseMs:  int64(100 + i),
				},
			},
			Endpoint: protocol.Endpoint{Type: "http", URL: fmt.Sprintf("http://worker-%d:8080", i)},
			Limits:   protocol.WorkerLimits{MaxConcurrentTasks: 0}, // 0 = unlimited
		})
	}

	cleanup := func() { bus.Stop() }
	return rtr, cleanup
}

// newTask returns a minimal Task that targets the "text-gen" capability.
func newTask() *protocol.Task {
	return &protocol.Task{
		ID:       protocol.GenerateID("task"),
		Type:     "text-gen",
		Priority: protocol.PriorityNormal,
		Status:   protocol.TaskPending,
		Routing: protocol.RoutingConfig{
			Strategy:             "best_match",
			RequiredCapabilities: []string{"text-gen"},
		},
		CreatedAt: time.Now(),
	}
}

// suppressRoutingLogs silences log output for routing benchmarks.
func suppressRoutingLogs(b *testing.B) {
	b.Helper()
	orig := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(orig) })
}

// BenchmarkTaskRouting_10Workers measures route latency with 10 registered workers.
func BenchmarkTaskRouting_10Workers(b *testing.B) {
	suppressRoutingLogs(b)
	rtr, cleanup := newRoutingStack(10)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := newTask()
		if _, err := rtr.RouteTask(task); err != nil {
			b.Fatalf("RouteTask failed: %v", err)
		}
	}
}

// BenchmarkTaskRouting_100Workers measures route latency with 100 registered workers.
func BenchmarkTaskRouting_100Workers(b *testing.B) {
	suppressRoutingLogs(b)
	rtr, cleanup := newRoutingStack(100)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := newTask()
		if _, err := rtr.RouteTask(task); err != nil {
			b.Fatalf("RouteTask failed: %v", err)
		}
	}
}

// BenchmarkTaskRouting_1000Workers measures route latency with 1000 registered workers.
func BenchmarkTaskRouting_1000Workers(b *testing.B) {
	suppressRoutingLogs(b)
	rtr, cleanup := newRoutingStack(1000)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := newTask()
		if _, err := rtr.RouteTask(task); err != nil {
			b.Fatalf("RouteTask failed: %v", err)
		}
	}
}

// BenchmarkTaskRoutingParallel_100Workers measures concurrent routing throughput with 100 workers.
func BenchmarkTaskRoutingParallel_100Workers(b *testing.B) {
	suppressRoutingLogs(b)
	rtr, cleanup := newRoutingStack(100)
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := newTask()
			if _, err := rtr.RouteTask(task); err != nil {
				b.Errorf("RouteTask failed: %v", err)
			}
		}
	})
}

// BenchmarkTaskRoutingParallel_1000Workers measures concurrent routing throughput with 1000 workers.
func BenchmarkTaskRoutingParallel_1000Workers(b *testing.B) {
	suppressRoutingLogs(b)
	rtr, cleanup := newRoutingStack(1000)
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := newTask()
			if _, err := rtr.RouteTask(task); err != nil {
				b.Errorf("RouteTask failed: %v", err)
			}
		}
	})
}
