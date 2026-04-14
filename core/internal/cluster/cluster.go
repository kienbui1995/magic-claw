// Package cluster provides leader election and coordination for running
// multiple MagiC server instances. Uses PostgreSQL advisory locks for
// leader election — no external dependencies (etcd, Consul, etc.) needed.
//
// Only the leader runs singleton tasks: health checks, webhook delivery,
// workflow orchestration. All instances handle API requests.
package cluster

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync/atomic"
	"time"
)

const (
	// AdvisoryLockID is the PostgreSQL advisory lock key for leader election.
	// Chosen to be unlikely to conflict with application locks.
	AdvisoryLockID int64 = 0x4D414749_43000001 // "MAGIC" + 1

	// ElectionInterval is how often non-leaders attempt to acquire the lock.
	ElectionInterval = 5 * time.Second

	// HeartbeatInterval is how often the leader renews its lock.
	HeartbeatInterval = 3 * time.Second
)

// LockFunc tries to acquire a PostgreSQL advisory lock. Returns true if acquired.
type LockFunc func(ctx context.Context, lockID int64) (bool, error)

// UnlockFunc releases a PostgreSQL advisory lock.
type UnlockFunc func(ctx context.Context, lockID int64) error

// Elector manages leader election for a cluster of MagiC instances.
type Elector struct {
	instanceID string
	isLeader   atomic.Bool
	lockFn     LockFunc
	unlockFn   UnlockFunc
	onElected  func()
	onDemoted  func()
}

func generateInstanceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "magic-" + hex.EncodeToString(b)
}

// NewElector creates a new leader elector.
// lockFn/unlockFn should wrap pg_try_advisory_lock / pg_advisory_unlock.
func NewElector(lockFn LockFunc, unlockFn UnlockFunc) *Elector {
	return &Elector{
		instanceID: generateInstanceID(),
		lockFn:     lockFn,
		unlockFn:   unlockFn,
	}
}

// OnElected sets a callback for when this instance becomes leader.
func (e *Elector) OnElected(fn func()) { e.onElected = fn }

// OnDemoted sets a callback for when this instance loses leadership.
func (e *Elector) OnDemoted(fn func()) { e.onDemoted = fn }

// IsLeader returns whether this instance is currently the leader.
func (e *Elector) IsLeader() bool { return e.isLeader.Load() }

// InstanceID returns this instance's unique identifier.
func (e *Elector) InstanceID() string { return e.instanceID }

// Run starts the election loop. Blocks until ctx is cancelled.
func (e *Elector) Run(ctx context.Context) {
	log.Printf("[cluster] instance %s starting election loop", e.instanceID)
	ticker := time.NewTicker(ElectionInterval)
	defer ticker.Stop()

	for {
		e.tryElection(ctx)

		select {
		case <-ctx.Done():
			// Release lock on shutdown
			if e.isLeader.Load() {
				e.unlockFn(context.Background(), AdvisoryLockID) //nolint:errcheck
				log.Printf("[cluster] instance %s released leadership", e.instanceID)
			}
			return
		case <-ticker.C:
		}
	}
}

func (e *Elector) tryElection(ctx context.Context) {
	acquired, err := e.lockFn(ctx, AdvisoryLockID)
	if err != nil {
		return // DB error, skip this round
	}

	wasLeader := e.isLeader.Load()

	if acquired && !wasLeader {
		e.isLeader.Store(true)
		log.Printf("[cluster] instance %s elected as leader", e.instanceID)
		if e.onElected != nil {
			e.onElected()
		}
	} else if !acquired && wasLeader {
		e.isLeader.Store(false)
		log.Printf("[cluster] instance %s demoted from leader", e.instanceID)
		if e.onDemoted != nil {
			e.onDemoted()
		}
	}
}

// InMemoryElector is a simple single-instance elector for dev/testing.
// Always elects itself as leader immediately.
func InMemoryElector() *Elector {
	e := &Elector{
		instanceID: generateInstanceID(),
		lockFn:     func(ctx context.Context, id int64) (bool, error) { return true, nil },
		unlockFn:   func(ctx context.Context, id int64) error { return nil },
	}
	e.isLeader.Store(true)
	return e
}
