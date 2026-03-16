package store

import (
	"fmt"

	"github.com/kienbm/magic-claw/core/internal/protocol"
)

var ErrNotFound = fmt.Errorf("not found")

type Store interface {
	AddWorker(w *protocol.Worker) error
	GetWorker(id string) (*protocol.Worker, error)
	UpdateWorker(w *protocol.Worker) error
	RemoveWorker(id string) error
	ListWorkers() []*protocol.Worker
	FindWorkersByCapability(capability string) []*protocol.Worker

	AddTask(t *protocol.Task) error
	GetTask(id string) (*protocol.Task, error)
	UpdateTask(t *protocol.Task) error
	ListTasks() []*protocol.Task
}
