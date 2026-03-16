package store_test

import (
	"testing"

	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/store"
)

func TestMemoryStore_Workers(t *testing.T) {
	s := store.NewMemoryStore()

	w := &protocol.Worker{
		ID:     "worker_001",
		Name:   "TestBot",
		Status: protocol.StatusActive,
		Capabilities: []protocol.Capability{
			{Name: "greeting"},
		},
	}

	if err := s.AddWorker(w); err != nil {
		t.Fatalf("AddWorker: %v", err)
	}

	got, err := s.GetWorker("worker_001")
	if err != nil {
		t.Fatalf("GetWorker: %v", err)
	}
	if got.Name != "TestBot" {
		t.Errorf("Name: got %q, want TestBot", got.Name)
	}

	workers := s.ListWorkers()
	if len(workers) != 1 {
		t.Errorf("ListWorkers: got %d, want 1", len(workers))
	}

	found := s.FindWorkersByCapability("greeting")
	if len(found) != 1 {
		t.Errorf("FindByCapability: got %d, want 1", len(found))
	}

	found = s.FindWorkersByCapability("nonexistent")
	if len(found) != 0 {
		t.Errorf("FindByCapability nonexistent: got %d, want 0", len(found))
	}

	if err := s.RemoveWorker("worker_001"); err != nil {
		t.Fatalf("RemoveWorker: %v", err)
	}
	if _, err := s.GetWorker("worker_001"); err == nil {
		t.Error("GetWorker after remove should fail")
	}
}

func TestMemoryStore_Tasks(t *testing.T) {
	s := store.NewMemoryStore()

	task := &protocol.Task{
		ID:     "task_001",
		Type:   "greeting",
		Status: protocol.TaskPending,
	}

	if err := s.AddTask(task); err != nil {
		t.Fatalf("AddTask: %v", err)
	}

	got, err := s.GetTask("task_001")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Type != "greeting" {
		t.Errorf("Type: got %q, want greeting", got.Type)
	}

	task.Status = protocol.TaskCompleted
	if err := s.UpdateTask(task); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	got, _ = s.GetTask("task_001")
	if got.Status != protocol.TaskCompleted {
		t.Errorf("Status: got %q, want completed", got.Status)
	}
}
