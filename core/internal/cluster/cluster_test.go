package cluster

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestInMemoryElector(t *testing.T) {
	e := InMemoryElector()
	if !e.IsLeader() {
		t.Error("in-memory elector should always be leader")
	}
	if e.InstanceID() == "" {
		t.Error("should have instance ID")
	}
}

func TestElection(t *testing.T) {
	var lockHeld atomic.Bool

	lockFn := func(ctx context.Context, id int64) (bool, error) {
		return lockHeld.CompareAndSwap(false, true), nil
	}
	unlockFn := func(ctx context.Context, id int64) error {
		lockHeld.Store(false)
		return nil
	}

	elected := make(chan struct{}, 1)
	e := NewElector(lockFn, unlockFn)
	e.OnElected(func() { elected <- struct{}{} })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go e.Run(ctx)

	select {
	case <-elected:
		if !e.IsLeader() {
			t.Error("should be leader after election")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for election")
	}
}

func TestTwoInstancesOnlyOneLeader(t *testing.T) {
	var lockHeld atomic.Bool

	lockFn := func(ctx context.Context, id int64) (bool, error) {
		return lockHeld.CompareAndSwap(false, true), nil
	}
	unlockFn := func(ctx context.Context, id int64) error {
		lockHeld.Store(false)
		return nil
	}

	e1 := NewElector(lockFn, unlockFn)
	e2 := NewElector(lockFn, unlockFn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go e1.Run(ctx)
	time.Sleep(100 * time.Millisecond) // let e1 win
	go e2.Run(ctx)
	time.Sleep(200 * time.Millisecond)

	leaders := 0
	if e1.IsLeader() {
		leaders++
	}
	if e2.IsLeader() {
		leaders++
	}
	if leaders != 1 {
		t.Errorf("expected exactly 1 leader, got %d", leaders)
	}
}
