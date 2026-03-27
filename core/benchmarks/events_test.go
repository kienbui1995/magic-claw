package benchmarks

import (
	"io"
	"log"
	"sync"
	"testing"

	"github.com/kienbui1995/magic/core/internal/events"
)

// suppressLogs silences the standard logger for the duration of the benchmark.
// The event bus logs "WARNING: event buffer full" via log.Printf; under extreme
// throughput (parallel publish with no consumers) this is expected behavior and
// only creates noise in benchmark output.
func suppressLogs(b *testing.B) {
	b.Helper()
	orig := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(orig) })
}

// BenchmarkEventBus_Publish measures publish throughput with 10 subscribers.
// Uses a large buffer so the benchmark measures Publish latency, not buffer
// contention — the buffer is intentionally oversized for benchmarking purposes.
func BenchmarkEventBus_Publish(b *testing.B) {
	suppressLogs(b)
	bus := events.NewBusWithConfig(64, 1<<20) // 1M event buffer
	defer bus.Stop()

	const numSubscribers = 10
	var mu sync.Mutex
	received := 0

	for i := 0; i < numSubscribers; i++ {
		bus.Subscribe("bench.event", func(e events.Event) {
			mu.Lock()
			received++
			mu.Unlock()
		})
	}

	evt := events.Event{
		Type:   "bench.event",
		Source: "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(evt)
	}
}

// BenchmarkEventBus_PublishParallel measures concurrent publish throughput.
// No subscribers are registered so that the benchmark isolates the cost of
// enqueuing an event onto the channel (the channel-send path), not handler
// execution. Subscriber dispatch throughput is covered by BenchmarkEventBus_Publish.
func BenchmarkEventBus_PublishParallel(b *testing.B) {
	suppressLogs(b)
	// Size the buffer to b.N to avoid drops; cap at 4M for memory safety.
	bufSize := b.N
	if bufSize < 4096 {
		bufSize = 4096
	}
	if bufSize > 1<<22 {
		bufSize = 1 << 22
	}
	bus := events.NewBusWithConfig(64, bufSize)
	defer bus.Stop()

	evt := events.Event{
		Type:   "bench.parallel",
		Source: "benchmark",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bus.Publish(evt)
		}
	})
}

// BenchmarkEventBus_FanOut measures publish latency with 100 subscribers (fan-out pattern).
func BenchmarkEventBus_FanOut(b *testing.B) {
	suppressLogs(b)
	bus := events.NewBusWithConfig(64, 1<<20)
	defer bus.Stop()

	const numSubscribers = 100
	for i := 0; i < numSubscribers; i++ {
		bus.Subscribe("bench.fanout", func(e events.Event) {})
	}

	evt := events.Event{
		Type:    "bench.fanout",
		Source:  "benchmark",
		Payload: map[string]any{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(evt)
	}
}
