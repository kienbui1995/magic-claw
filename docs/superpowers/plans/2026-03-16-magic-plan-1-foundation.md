# MagiC Plan 1: Foundation + Core Modules

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working MagiC server that can register AI workers, route tasks to them, and monitor all activity — with a Python SDK and hello-world example.

**Architecture:** Go HTTP server with 4 core modules (Gateway, Registry, Router, Monitor) sharing protocol types and an event bus. Workers communicate via MCP² JSON messages over HTTP. Python SDK wraps the HTTP API for easy worker development.

**Tech Stack:** Go 1.22+, Python 3.11+, SQLite (dev storage), structured JSON logging

**Spec:** `docs/superpowers/specs/2026-03-16-magic-framework-design.md`

---

## File Map

```
magic-claw/
├── core/
│   ├── cmd/magic/
│   │   └── main.go                    # CLI entrypoint
│   ├── internal/
│   │   ├── protocol/
│   │   │   ├── types.go               # Entity types (Worker, Task, Team, etc.)
│   │   │   ├── messages.go            # MCP² message types
│   │   │   ├── messages_test.go
│   │   │   └── validate.go            # JSON schema validation
│   │   ├── store/
│   │   │   ├── store.go               # Store interface
│   │   │   ├── memory.go              # In-memory implementation
│   │   │   └── memory_test.go
│   │   ├── events/
│   │   │   ├── bus.go                 # Event bus (pub/sub)
│   │   │   └── bus_test.go
│   │   ├── gateway/
│   │   │   ├── gateway.go             # HTTP server + router setup
│   │   │   ├── middleware.go          # Auth, request ID, logging
│   │   │   ├── handlers.go           # HTTP handlers
│   │   │   └── gateway_test.go
│   │   ├── registry/
│   │   │   ├── registry.go           # Worker registration + discovery
│   │   │   ├── health.go             # Heartbeat monitoring
│   │   │   └── registry_test.go
│   │   ├── router/
│   │   │   ├── router.go             # Task routing engine
│   │   │   ├── strategy.go           # Routing strategies
│   │   │   ├── scorer.go             # Capability matching score
│   │   │   └── router_test.go
│   │   └── monitor/
│   │       ├── monitor.go            # Event listener + metrics
│   │       ├── logger.go             # Structured JSON logger
│   │       └── monitor_test.go
│   ├── go.mod
│   └── go.sum
├── sdk/
│   └── python/
│       ├── magic_claw/
│       │   ├── __init__.py
│       │   ├── worker.py             # Base Worker class
│       │   ├── client.py             # HTTP client for MagiC API
│       │   ├── protocol.py           # Message types (Python)
│       │   └── decorators.py         # @capability decorator
│       ├── pyproject.toml
│       └── tests/
│           └── test_worker.py
├── examples/
│   └── hello-worker/
│       ├── main.py                   # < 20 lines hello world
│       └── README.md
├── Makefile
└── .gitignore
```

---

## Chunk 1: Go Project Scaffold + Protocol Types

### Task 1: Initialize Go project

**Files:**
- Create: `core/go.mod`
- Create: `core/cmd/magic/main.go`
- Create: `Makefile`
- Create: `.gitignore`

- [ ] **Step 1: Init Go module**

```bash
cd /home/kienbm/magic-claw
mkdir -p core/cmd/magic
cd core
go mod init github.com/kienbm/magic-claw/core
```

- [ ] **Step 2: Create minimal main.go**

Create `core/cmd/magic/main.go`:
```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("MagiC — Where AI becomes a Company")
		fmt.Println("Usage: magic <command>")
		fmt.Println("Commands: serve")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "serve":
		fmt.Println("Starting MagiC server...")
		// Will be implemented in Gateway task
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: build test run dev clean

build:
	cd core && go build -o ../bin/magic ./cmd/magic

test:
	cd core && go test ./... -v

run: build
	./bin/magic serve

dev:
	cd core && go run ./cmd/magic serve

clean:
	rm -rf bin/
```

- [ ] **Step 4: Create .gitignore**

Create `.gitignore`:
```
bin/
*.exe
.env
.env.*
*.db
__pycache__/
*.pyc
.venv/
dist/
*.egg-info/
node_modules/
.DS_Store
feedback.json
spec-feedback.json
```

- [ ] **Step 5: Verify build**

Run: `make build`
Expected: binary at `bin/magic`

Run: `./bin/magic`
Expected: prints usage message

- [ ] **Step 6: Commit**

```bash
git init
git add core/go.mod core/cmd/magic/main.go Makefile .gitignore
git commit -m "feat: initialize Go project scaffold"
```

---

### Task 2: Protocol types — entities

**Files:**
- Create: `core/internal/protocol/types.go`
- Create: `core/internal/protocol/types_test.go`

- [ ] **Step 1: Write test for entity serialization**

Create `core/internal/protocol/types_test.go`:
```go
package protocol_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kienbm/magic-claw/core/internal/protocol"
)

func TestWorkerSerialization(t *testing.T) {
	w := protocol.Worker{
		ID:   "worker_001",
		Name: "TestBot",
		Capabilities: []protocol.Capability{
			{
				Name:        "greeting",
				Description: "Says hello",
			},
		},
		Endpoint: protocol.Endpoint{
			Type: "http",
			URL:  "http://localhost:9000/mcp2",
		},
		Status: protocol.StatusActive,
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal worker: %v", err)
	}

	var w2 protocol.Worker
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatalf("unmarshal worker: %v", err)
	}

	if w2.ID != w.ID {
		t.Errorf("ID: got %q, want %q", w2.ID, w.ID)
	}
	if w2.Name != w.Name {
		t.Errorf("Name: got %q, want %q", w2.Name, w.Name)
	}
	if len(w2.Capabilities) != 1 {
		t.Fatalf("Capabilities: got %d, want 1", len(w2.Capabilities))
	}
	if w2.Status != protocol.StatusActive {
		t.Errorf("Status: got %q, want %q", w2.Status, protocol.StatusActive)
	}
}

func TestTaskSerialization(t *testing.T) {
	task := protocol.Task{
		ID:       "task_001",
		Type:     "greeting",
		Priority: protocol.PriorityNormal,
		Status:   protocol.TaskPending,
		Input:    json.RawMessage(`{"name": "Kien"}`),
		Contract: protocol.Contract{
			TimeoutMs: 30000,
			MaxCost:   0.50,
		},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	var task2 protocol.Task
	if err := json.Unmarshal(data, &task2); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}

	if task2.ID != "task_001" {
		t.Errorf("ID: got %q, want %q", task2.ID, "task_001")
	}
	if task2.Contract.MaxCost != 0.50 {
		t.Errorf("MaxCost: got %f, want 0.50", task2.Contract.MaxCost)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := protocol.GenerateID("worker")
	id2 := protocol.GenerateID("worker")
	if id1 == id2 {
		t.Error("GenerateID should return unique IDs")
	}
	if len(id1) < 10 {
		t.Errorf("ID too short: %q", id1)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/protocol/ -v`
Expected: FAIL — package not found

- [ ] **Step 3: Implement protocol types**

Create `core/internal/protocol/types.go`:
```go
package protocol

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Worker statuses
const (
	StatusActive  = "active"
	StatusPaused  = "paused"
	StatusOffline = "offline"
)

// Task statuses
const (
	TaskPending    = "pending"
	TaskAssigned   = "assigned"
	TaskAccepted   = "accepted"
	TaskInProgress = "in_progress"
	TaskCompleted  = "completed"
	TaskFailed     = "failed"
)

// Task priorities
const (
	PriorityLow      = "low"
	PriorityNormal   = "normal"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// GenerateID creates a unique prefixed ID.
func GenerateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

type Capability struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	InputSchema    json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema   json.RawMessage `json:"output_schema,omitempty"`
	EstCostPerCall float64         `json:"est_cost_per_call,omitempty"`
	AvgResponseMs  int64           `json:"avg_response_ms,omitempty"`
}

type Endpoint struct {
	Type   string       `json:"type"` // http | grpc | ws
	URL    string       `json:"url"`
	Auth   *EndpointAuth `json:"auth,omitempty"`
}

type EndpointAuth struct {
	Type   string `json:"type"`   // api_key | bearer
	Header string `json:"header"` // header name
}

type WorkerLimits struct {
	MaxConcurrentTasks int     `json:"max_concurrent_tasks"`
	RateLimit          string  `json:"rate_limit,omitempty"`
	MaxCostPerDay      float64 `json:"max_cost_per_day,omitempty"`
}

type Worker struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	TeamID        string            `json:"team_id,omitempty"`
	Capabilities  []Capability      `json:"capabilities"`
	Endpoint      Endpoint          `json:"endpoint"`
	Limits        WorkerLimits      `json:"limits"`
	Status        string            `json:"status"`
	CurrentLoad   int               `json:"current_load"`
	TotalCostToday float64          `json:"total_cost_today"`
	RegisteredAt  time.Time         `json:"registered_at"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}

type Contract struct {
	OutputSchema    json.RawMessage   `json:"output_schema,omitempty"`
	QualityCriteria []QualityCriterion `json:"quality_criteria,omitempty"`
	TimeoutMs       int64             `json:"timeout_ms"`
	MaxCost         float64           `json:"max_cost"`
	RetryPolicy     *RetryPolicy      `json:"retry_policy,omitempty"`
}

type QualityCriterion struct {
	Metric    string  `json:"metric"`
	Threshold float64 `json:"threshold"`
}

type RetryPolicy struct {
	MaxRetries int   `json:"max_retries"`
	BackoffMs  int64 `json:"backoff_ms,omitempty"`
}

type RoutingConfig struct {
	Strategy             string   `json:"strategy"` // best_match | round_robin | cheapest | fastest | specific
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	PreferredWorkers     []string `json:"preferred_workers,omitempty"`
	ExcludedWorkers      []string `json:"excluded_workers,omitempty"`
}

type TaskContext struct {
	OrgID      string `json:"org_id,omitempty"`
	TeamID     string `json:"team_id,omitempty"`
	Requester  string `json:"requester,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
}

type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type Task struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Priority       string          `json:"priority"`
	Status         string          `json:"status"`
	Input          json.RawMessage `json:"input"`
	Output         json.RawMessage `json:"output,omitempty"`
	Contract       Contract        `json:"contract"`
	Routing        RoutingConfig   `json:"routing"`
	AssignedWorker string          `json:"assigned_worker,omitempty"`
	WorkflowID     string          `json:"workflow_id,omitempty"`
	Context        TaskContext     `json:"context"`
	Cost           float64         `json:"cost"`
	Progress       int             `json:"progress"` // 0-100
	CreatedAt      time.Time       `json:"created_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	Error          *TaskError      `json:"error,omitempty"`
}

type Team struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	OrgID            string  `json:"org_id"`
	Workers          []string `json:"workers"`
	DailyBudget      float64 `json:"daily_budget"`
	ApprovalRequired bool    `json:"approval_required"`
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/protocol/ -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/protocol/
git commit -m "feat(protocol): add core entity types (Worker, Task, Team, Capability)"
```

---

### Task 3: Protocol types — MCP² messages

**Files:**
- Create: `core/internal/protocol/messages.go`
- Modify: `core/internal/protocol/types_test.go` (add message tests)

- [ ] **Step 1: Write test for message serialization**

Append to `core/internal/protocol/types_test.go`:
```go
func TestMessageSerialization(t *testing.T) {
	msg := protocol.Message{
		Protocol:  "mcp2",
		Version:   "1.0",
		Type:      protocol.MsgWorkerRegister,
		ID:        "msg_001",
		Timestamp: time.Now(),
		Source:    "worker_001",
		Target:   "org_magic",
		Payload:   json.RawMessage(`{"name": "TestBot"}`),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var msg2 protocol.Message
	if err := json.Unmarshal(data, &msg2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if msg2.Type != protocol.MsgWorkerRegister {
		t.Errorf("Type: got %q, want %q", msg2.Type, protocol.MsgWorkerRegister)
	}
}

func TestNewMessage(t *testing.T) {
	msg := protocol.NewMessage(protocol.MsgTaskAssign, "org", "worker_001", json.RawMessage(`{}`))
	if msg.Protocol != "mcp2" {
		t.Errorf("Protocol: got %q, want mcp2", msg.Protocol)
	}
	if msg.ID == "" {
		t.Error("ID should not be empty")
	}
	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/protocol/ -v -run TestMessage`
Expected: FAIL — MsgWorkerRegister not defined

- [ ] **Step 3: Implement message types**

Create `core/internal/protocol/messages.go`:
```go
package protocol

import (
	"encoding/json"
	"time"
)

// Message types — MCP² protocol
const (
	// Worker lifecycle
	MsgWorkerRegister         = "worker.register"
	MsgWorkerHeartbeat        = "worker.heartbeat"
	MsgWorkerDeregister       = "worker.deregister"
	MsgWorkerUpdateCapabilities = "worker.update_capabilities"

	// Task lifecycle
	MsgTaskAssign    = "task.assign"
	MsgTaskAccept    = "task.accept"
	MsgTaskReject    = "task.reject"
	MsgTaskProgress  = "task.progress"
	MsgTaskComplete  = "task.complete"
	MsgTaskFail      = "task.fail"

	// Collaboration
	MsgWorkerDelegate   = "worker.delegate"
	MsgOrgBroadcast     = "org.broadcast"

	// Direct channel
	MsgWorkerOpenChannel  = "worker.open_channel"
	MsgWorkerCloseChannel = "worker.close_channel"
)

// Message is the top-level MCP² protocol envelope.
type Message struct {
	Protocol  string          `json:"protocol"`
	Version   string          `json:"version"`
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Target   string          `json:"target"`
	Payload   json.RawMessage `json:"payload"`
}

// NewMessage creates a new MCP² message with generated ID and current timestamp.
func NewMessage(msgType, source, target string, payload json.RawMessage) Message {
	return Message{
		Protocol:  "mcp2",
		Version:   "1.0",
		Type:      msgType,
		ID:        GenerateID("msg"),
		Timestamp: time.Now(),
		Source:    source,
		Target:   target,
		Payload:   payload,
	}
}

// Registration payload sent by worker when joining.
type RegisterPayload struct {
	Name         string            `json:"name"`
	Capabilities []Capability      `json:"capabilities"`
	Endpoint     Endpoint          `json:"endpoint"`
	Limits       WorkerLimits      `json:"limits"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}

// TaskAssignPayload sent by org when assigning a task.
type TaskAssignPayload struct {
	TaskID   string          `json:"task_id"`
	TaskType string          `json:"task_type"`
	Priority string          `json:"priority"`
	Input    json.RawMessage `json:"input"`
	Contract Contract        `json:"contract"`
	Context  TaskContext     `json:"context"`
}

// TaskCompletePayload sent by worker when task is done.
type TaskCompletePayload struct {
	TaskID string          `json:"task_id"`
	Output json.RawMessage `json:"output"`
	Cost   float64         `json:"cost"`
}

// TaskFailPayload sent by worker when task fails.
type TaskFailPayload struct {
	TaskID string    `json:"task_id"`
	Error  TaskError `json:"error"`
}

// TaskProgressPayload sent by worker to report progress.
type TaskProgressPayload struct {
	TaskID   string          `json:"task_id"`
	Progress int             `json:"progress"` // 0-100
	Output   json.RawMessage `json:"output,omitempty"` // intermediate result
}

// HeartbeatPayload sent periodically by worker.
type HeartbeatPayload struct {
	WorkerID    string `json:"worker_id"`
	CurrentLoad int    `json:"current_load"`
	Status      string `json:"status"`
}

// DelegatePayload sent when worker needs another worker's help.
type DelegatePayload struct {
	FromTaskID          string          `json:"from_task_id"`
	RequiredCapability  string          `json:"required_capability"`
	Input               json.RawMessage `json:"input"`
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/protocol/ -v`
Expected: PASS (5 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/protocol/messages.go core/internal/protocol/types_test.go
git commit -m "feat(protocol): add MCP² message types and payloads"
```

---

### Task 4: In-memory store

**Files:**
- Create: `core/internal/store/store.go`
- Create: `core/internal/store/memory.go`
- Create: `core/internal/store/memory_test.go`

- [ ] **Step 1: Write test for store operations**

Create `core/internal/store/memory_test.go`:
```go
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

	// Add
	if err := s.AddWorker(w); err != nil {
		t.Fatalf("AddWorker: %v", err)
	}

	// Get
	got, err := s.GetWorker("worker_001")
	if err != nil {
		t.Fatalf("GetWorker: %v", err)
	}
	if got.Name != "TestBot" {
		t.Errorf("Name: got %q, want TestBot", got.Name)
	}

	// List
	workers := s.ListWorkers()
	if len(workers) != 1 {
		t.Errorf("ListWorkers: got %d, want 1", len(workers))
	}

	// Find by capability
	found := s.FindWorkersByCapability("greeting")
	if len(found) != 1 {
		t.Errorf("FindByCapability: got %d, want 1", len(found))
	}

	found = s.FindWorkersByCapability("nonexistent")
	if len(found) != 0 {
		t.Errorf("FindByCapability nonexistent: got %d, want 0", len(found))
	}

	// Remove
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
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/store/ -v`
Expected: FAIL

- [ ] **Step 3: Implement store interface and memory store**

Create `core/internal/store/store.go`:
```go
package store

import (
	"fmt"

	"github.com/kienbm/magic-claw/core/internal/protocol"
)

var ErrNotFound = fmt.Errorf("not found")

type Store interface {
	// Workers
	AddWorker(w *protocol.Worker) error
	GetWorker(id string) (*protocol.Worker, error)
	UpdateWorker(w *protocol.Worker) error
	RemoveWorker(id string) error
	ListWorkers() []*protocol.Worker
	FindWorkersByCapability(capability string) []*protocol.Worker

	// Tasks
	AddTask(t *protocol.Task) error
	GetTask(id string) (*protocol.Task, error)
	UpdateTask(t *protocol.Task) error
	ListTasks() []*protocol.Task
}
```

Create `core/internal/store/memory.go`:
```go
package store

import (
	"sync"

	"github.com/kienbm/magic-claw/core/internal/protocol"
)

type MemoryStore struct {
	mu      sync.RWMutex
	workers map[string]*protocol.Worker
	tasks   map[string]*protocol.Task
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workers: make(map[string]*protocol.Worker),
		tasks:   make(map[string]*protocol.Task),
	}
}

func (s *MemoryStore) AddWorker(w *protocol.Worker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workers[w.ID] = w
	return nil
}

func (s *MemoryStore) GetWorker(id string) (*protocol.Worker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workers[id]
	if !ok {
		return nil, ErrNotFound
	}
	return w, nil
}

func (s *MemoryStore) UpdateWorker(w *protocol.Worker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workers[w.ID]; !ok {
		return ErrNotFound
	}
	s.workers[w.ID] = w
	return nil
}

func (s *MemoryStore) RemoveWorker(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workers[id]; !ok {
		return ErrNotFound
	}
	delete(s.workers, id)
	return nil
}

func (s *MemoryStore) ListWorkers() []*protocol.Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*protocol.Worker, 0, len(s.workers))
	for _, w := range s.workers {
		result = append(result, w)
	}
	return result
}

func (s *MemoryStore) FindWorkersByCapability(capability string) []*protocol.Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*protocol.Worker
	for _, w := range s.workers {
		if w.Status != protocol.StatusActive {
			continue
		}
		for _, cap := range w.Capabilities {
			if cap.Name == capability {
				result = append(result, w)
				break
			}
		}
	}
	return result
}

func (s *MemoryStore) AddTask(t *protocol.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
	return nil
}

func (s *MemoryStore) GetTask(id string) (*protocol.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *MemoryStore) UpdateTask(t *protocol.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[t.ID]; !ok {
		return ErrNotFound
	}
	s.tasks[t.ID] = t
	return nil
}

func (s *MemoryStore) ListTasks() []*protocol.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*protocol.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, t)
	}
	return result
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/store/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add core/internal/store/
git commit -m "feat(store): add Store interface and in-memory implementation"
```

---

### Task 5: Event bus

**Files:**
- Create: `core/internal/events/bus.go`
- Create: `core/internal/events/bus_test.go`

- [ ] **Step 1: Write test for event bus**

Create `core/internal/events/bus_test.go`:
```go
package events_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kienbm/magic-claw/core/internal/events"
)

func TestEventBus_PubSub(t *testing.T) {
	bus := events.NewBus()
	var received []events.Event
	var mu sync.Mutex

	bus.Subscribe("task.completed", func(e events.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	bus.Publish(events.Event{
		Type:    "task.completed",
		Source:  "router",
		Payload: map[string]any{"task_id": "task_001"},
	})

	time.Sleep(50 * time.Millisecond) // async delivery

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received: got %d, want 1", len(received))
	}
	if received[0].Type != "task.completed" {
		t.Errorf("type: got %q", received[0].Type)
	}
}

func TestEventBus_WildcardSubscribe(t *testing.T) {
	bus := events.NewBus()
	var received []events.Event
	var mu sync.Mutex

	bus.Subscribe("*", func(e events.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	bus.Publish(events.Event{Type: "task.completed"})
	bus.Publish(events.Event{Type: "worker.registered"})

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Errorf("received: got %d, want 2", len(received))
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/events/ -v`
Expected: FAIL

- [ ] **Step 3: Implement event bus**

Create `core/internal/events/bus.go`:
```go
package events

import (
	"sync"
	"time"
)

type Event struct {
	Type      string         `json:"type"`
	Source    string         `json:"source"`
	Payload   map[string]any `json:"payload,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Severity  string         `json:"severity"` // info | warn | error | critical
}

type Handler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

func (b *Bus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *Bus) Publish(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	if e.Severity == "" {
		e.Severity = "info"
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Deliver to specific subscribers
	for _, h := range b.handlers[e.Type] {
		go h(e)
	}
	// Deliver to wildcard subscribers
	if e.Type != "*" {
		for _, h := range b.handlers["*"] {
			go h(e)
		}
	}
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/events/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add core/internal/events/
git commit -m "feat(events): add event bus with pub/sub and wildcard support"
```

---

## Chunk 2: Core Modules (Gateway, Registry, Router, Monitor)

### Task 6: Registry module

**Files:**
- Create: `core/internal/registry/registry.go`
- Create: `core/internal/registry/health.go`
- Create: `core/internal/registry/registry_test.go`

- [ ] **Step 1: Write test for registry**

Create `core/internal/registry/registry_test.go`:
```go
package registry_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/registry"
	"github.com/kienbm/magic-claw/core/internal/store"
)

func TestRegistry_Register(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)

	payload := protocol.RegisterPayload{
		Name: "TestBot",
		Capabilities: []protocol.Capability{
			{Name: "greeting", Description: "Says hello"},
		},
		Endpoint: protocol.Endpoint{Type: "http", URL: "http://localhost:9000"},
		Limits:   protocol.WorkerLimits{MaxConcurrentTasks: 5},
	}

	worker, err := reg.Register(payload)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if worker.ID == "" {
		t.Error("worker ID should not be empty")
	}
	if worker.Status != protocol.StatusActive {
		t.Errorf("status: got %q, want active", worker.Status)
	}

	// Verify stored
	got, err := s.GetWorker(worker.ID)
	if err != nil {
		t.Fatalf("GetWorker: %v", err)
	}
	if got.Name != "TestBot" {
		t.Errorf("Name: got %q", got.Name)
	}
}

func TestRegistry_Heartbeat(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)

	payload := protocol.RegisterPayload{
		Name:     "TestBot",
		Endpoint: protocol.Endpoint{Type: "http", URL: "http://localhost:9000"},
	}
	worker, _ := reg.Register(payload)

	hb := protocol.HeartbeatPayload{
		WorkerID:    worker.ID,
		CurrentLoad: 2,
		Status:      protocol.StatusActive,
	}

	err := reg.Heartbeat(hb)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	got, _ := s.GetWorker(worker.ID)
	if got.CurrentLoad != 2 {
		t.Errorf("CurrentLoad: got %d, want 2", got.CurrentLoad)
	}
}

func TestRegistry_Deregister(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)

	payload := protocol.RegisterPayload{
		Name:     "TestBot",
		Endpoint: protocol.Endpoint{Type: "http", URL: "http://localhost:9000"},
	}
	worker, _ := reg.Register(payload)

	err := reg.Deregister(worker.ID)
	if err != nil {
		t.Fatalf("Deregister: %v", err)
	}

	_, err = s.GetWorker(worker.ID)
	if err == nil {
		t.Error("worker should be removed")
	}
}

func TestRegistry_FindByCapability(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)

	reg.Register(protocol.RegisterPayload{
		Name:         "ContentBot",
		Capabilities: []protocol.Capability{{Name: "content_writing"}},
		Endpoint:     protocol.Endpoint{Type: "http", URL: "http://localhost:9001"},
	})
	reg.Register(protocol.RegisterPayload{
		Name:         "DataBot",
		Capabilities: []protocol.Capability{{Name: "data_analysis"}},
		Endpoint:     protocol.Endpoint{Type: "http", URL: "http://localhost:9002"},
	})

	writers := reg.FindByCapability("content_writing")
	if len(writers) != 1 {
		t.Errorf("content_writing: got %d, want 1", len(writers))
	}
	if writers[0].Name != "ContentBot" {
		t.Errorf("Name: got %q", writers[0].Name)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/registry/ -v`
Expected: FAIL

- [ ] **Step 3: Implement registry**

Create `core/internal/registry/registry.go`:
```go
package registry

import (
	"time"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/store"
)

type Registry struct {
	store store.Store
	bus   *events.Bus
}

func New(s store.Store, bus *events.Bus) *Registry {
	return &Registry{store: s, bus: bus}
}

func (r *Registry) Register(p protocol.RegisterPayload) (*protocol.Worker, error) {
	w := &protocol.Worker{
		ID:            protocol.GenerateID("worker"),
		Name:          p.Name,
		Capabilities:  p.Capabilities,
		Endpoint:      p.Endpoint,
		Limits:        p.Limits,
		Status:        protocol.StatusActive,
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Metadata:      p.Metadata,
	}

	if err := r.store.AddWorker(w); err != nil {
		return nil, err
	}

	r.bus.Publish(events.Event{
		Type:   "worker.registered",
		Source: "registry",
		Payload: map[string]any{
			"worker_id":   w.ID,
			"worker_name": w.Name,
		},
	})

	return w, nil
}

func (r *Registry) Deregister(workerID string) error {
	if err := r.store.RemoveWorker(workerID); err != nil {
		return err
	}

	r.bus.Publish(events.Event{
		Type:   "worker.deregistered",
		Source: "registry",
		Payload: map[string]any{"worker_id": workerID},
	})

	return nil
}

func (r *Registry) Heartbeat(p protocol.HeartbeatPayload) error {
	w, err := r.store.GetWorker(p.WorkerID)
	if err != nil {
		return err
	}
	w.LastHeartbeat = time.Now()
	w.CurrentLoad = p.CurrentLoad
	if p.Status != "" {
		w.Status = p.Status
	}
	return r.store.UpdateWorker(w)
}

func (r *Registry) GetWorker(id string) (*protocol.Worker, error) {
	return r.store.GetWorker(id)
}

func (r *Registry) ListWorkers() []*protocol.Worker {
	return r.store.ListWorkers()
}

func (r *Registry) FindByCapability(capability string) []*protocol.Worker {
	return r.store.FindWorkersByCapability(capability)
}
```

Create `core/internal/registry/health.go`:
```go
package registry

import (
	"time"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
)

const HeartbeatTimeout = 60 * time.Second

// StartHealthCheck runs a goroutine that marks workers offline if no heartbeat.
func (r *Registry) StartHealthCheck(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			r.checkHealth()
		}
	}()
}

func (r *Registry) checkHealth() {
	workers := r.store.ListWorkers()
	now := time.Now()
	for _, w := range workers {
		if w.Status == protocol.StatusActive && now.Sub(w.LastHeartbeat) > HeartbeatTimeout {
			w.Status = protocol.StatusOffline
			r.store.UpdateWorker(w)
			r.bus.Publish(events.Event{
				Type:     "worker.offline",
				Source:   "registry",
				Severity: "warn",
				Payload:  map[string]any{"worker_id": w.ID, "worker_name": w.Name},
			})
		}
	}
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/registry/ -v`
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/registry/
git commit -m "feat(registry): add worker registration, heartbeat, health check"
```

---

### Task 7: Router module

**Files:**
- Create: `core/internal/router/router.go`
- Create: `core/internal/router/strategy.go`
- Create: `core/internal/router/router_test.go`

- [ ] **Step 1: Write test for router**

Create `core/internal/router/router_test.go`:
```go
package router_test

import (
	"encoding/json"
	"testing"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/registry"
	"github.com/kienbm/magic-claw/core/internal/router"
	"github.com/kienbm/magic-claw/core/internal/store"
)

func setupRouter(t *testing.T) (*router.Router, *registry.Registry) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)
	rt := router.New(reg, s, bus)

	reg.Register(protocol.RegisterPayload{
		Name:         "ContentBot",
		Capabilities: []protocol.Capability{{Name: "content_writing", EstCostPerCall: 0.05}},
		Endpoint:     protocol.Endpoint{Type: "http", URL: "http://localhost:9001"},
		Limits:       protocol.WorkerLimits{MaxConcurrentTasks: 5},
	})
	reg.Register(protocol.RegisterPayload{
		Name:         "CheapBot",
		Capabilities: []protocol.Capability{{Name: "content_writing", EstCostPerCall: 0.01}},
		Endpoint:     protocol.Endpoint{Type: "http", URL: "http://localhost:9002"},
		Limits:       protocol.WorkerLimits{MaxConcurrentTasks: 5},
	})

	return rt, reg
}

func TestRouter_RouteTask_BestMatch(t *testing.T) {
	rt, _ := setupRouter(t)

	task := &protocol.Task{
		ID:    protocol.GenerateID("task"),
		Type:  "content_writing",
		Input: json.RawMessage(`{"topic": "test"}`),
		Routing: protocol.RoutingConfig{
			Strategy:             "best_match",
			RequiredCapabilities: []string{"content_writing"},
		},
		Contract: protocol.Contract{TimeoutMs: 30000, MaxCost: 1.0},
	}

	worker, err := rt.RouteTask(task)
	if err != nil {
		t.Fatalf("RouteTask: %v", err)
	}
	if worker == nil {
		t.Fatal("worker should not be nil")
	}
}

func TestRouter_RouteTask_NoCapableWorker(t *testing.T) {
	rt, _ := setupRouter(t)

	task := &protocol.Task{
		ID:   protocol.GenerateID("task"),
		Type: "data_analysis",
		Routing: protocol.RoutingConfig{
			Strategy:             "best_match",
			RequiredCapabilities: []string{"data_analysis"},
		},
	}

	_, err := rt.RouteTask(task)
	if err == nil {
		t.Error("should fail — no worker with data_analysis capability")
	}
}

func TestRouter_RouteTask_Cheapest(t *testing.T) {
	rt, _ := setupRouter(t)

	task := &protocol.Task{
		ID:   protocol.GenerateID("task"),
		Type: "content_writing",
		Routing: protocol.RoutingConfig{
			Strategy:             "cheapest",
			RequiredCapabilities: []string{"content_writing"},
		},
	}

	worker, err := rt.RouteTask(task)
	if err != nil {
		t.Fatalf("RouteTask: %v", err)
	}
	if worker.Name != "CheapBot" {
		t.Errorf("cheapest should pick CheapBot, got %q", worker.Name)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/router/ -v`
Expected: FAIL

- [ ] **Step 3: Implement router**

Create `core/internal/router/strategy.go`:
```go
package router

import (
	"github.com/kienbm/magic-claw/core/internal/protocol"
)

type WorkerScore struct {
	Worker *protocol.Worker
	Score  float64
}

// filterByCapability returns workers that have all required capabilities.
func filterByCapability(workers []*protocol.Worker, required []string) []*protocol.Worker {
	var result []*protocol.Worker
	for _, w := range workers {
		if w.Status != protocol.StatusActive {
			continue
		}
		if hasAllCapabilities(w, required) {
			result = append(result, w)
		}
	}
	return result
}

func hasAllCapabilities(w *protocol.Worker, required []string) bool {
	capSet := make(map[string]bool)
	for _, c := range w.Capabilities {
		capSet[c.Name] = true
	}
	for _, r := range required {
		if !capSet[r] {
			return false
		}
	}
	return true
}

func scoreBestMatch(w *protocol.Worker) float64 {
	availability := 1.0
	if w.Limits.MaxConcurrentTasks > 0 {
		availability = 1.0 - float64(w.CurrentLoad)/float64(w.Limits.MaxConcurrentTasks)
	}
	if availability < 0 {
		availability = 0
	}
	return availability
}

func findCheapest(workers []*protocol.Worker, capName string) *protocol.Worker {
	var cheapest *protocol.Worker
	minCost := float64(999999)
	for _, w := range workers {
		for _, c := range w.Capabilities {
			if c.Name == capName && c.EstCostPerCall < minCost {
				minCost = c.EstCostPerCall
				cheapest = w
			}
		}
	}
	return cheapest
}
```

Create `core/internal/router/router.go`:
```go
package router

import (
	"fmt"
	"sort"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/registry"
	"github.com/kienbm/magic-claw/core/internal/store"
)

var ErrNoWorkerAvailable = fmt.Errorf("no worker available for task")

type Router struct {
	registry *registry.Registry
	store    store.Store
	bus      *events.Bus
}

func New(reg *registry.Registry, s store.Store, bus *events.Bus) *Router {
	return &Router{registry: reg, store: s, bus: bus}
}

// RouteTask finds the best worker for a task and assigns it.
func (r *Router) RouteTask(task *protocol.Task) (*protocol.Worker, error) {
	allWorkers := r.registry.ListWorkers()
	capable := filterByCapability(allWorkers, task.Routing.RequiredCapabilities)

	if len(capable) == 0 {
		return nil, ErrNoWorkerAvailable
	}

	// Apply excluded workers
	if len(task.Routing.ExcludedWorkers) > 0 {
		excluded := make(map[string]bool)
		for _, id := range task.Routing.ExcludedWorkers {
			excluded[id] = true
		}
		var filtered []*protocol.Worker
		for _, w := range capable {
			if !excluded[w.ID] {
				filtered = append(filtered, w)
			}
		}
		capable = filtered
		if len(capable) == 0 {
			return nil, ErrNoWorkerAvailable
		}
	}

	var selected *protocol.Worker

	switch task.Routing.Strategy {
	case "cheapest":
		capName := ""
		if len(task.Routing.RequiredCapabilities) > 0 {
			capName = task.Routing.RequiredCapabilities[0]
		}
		selected = findCheapest(capable, capName)

	case "specific":
		if len(task.Routing.PreferredWorkers) > 0 {
			targetID := task.Routing.PreferredWorkers[0]
			for _, w := range capable {
				if w.ID == targetID {
					selected = w
					break
				}
			}
		}

	default: // best_match, round_robin (fallback to best_match for now)
		scores := make([]WorkerScore, len(capable))
		for i, w := range capable {
			scores[i] = WorkerScore{Worker: w, Score: scoreBestMatch(w)}
		}
		sort.Slice(scores, func(i, j int) bool {
			return scores[i].Score > scores[j].Score
		})
		selected = scores[0].Worker
	}

	if selected == nil {
		return nil, ErrNoWorkerAvailable
	}

	// Update task
	task.AssignedWorker = selected.ID
	task.Status = protocol.TaskAssigned

	r.bus.Publish(events.Event{
		Type:   "task.routed",
		Source: "router",
		Payload: map[string]any{
			"task_id":     task.ID,
			"worker_id":   selected.ID,
			"worker_name": selected.Name,
			"strategy":    task.Routing.Strategy,
		},
	})

	return selected, nil
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/router/ -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/router/
git commit -m "feat(router): add task routing with best_match and cheapest strategies"
```

---

### Task 8: Monitor module

**Files:**
- Create: `core/internal/monitor/monitor.go`
- Create: `core/internal/monitor/logger.go`
- Create: `core/internal/monitor/monitor_test.go`

- [ ] **Step 1: Write test for monitor**

Create `core/internal/monitor/monitor_test.go`:
```go
package monitor_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/monitor"
)

func TestMonitor_CapturesEvents(t *testing.T) {
	bus := events.NewBus()
	var buf bytes.Buffer
	mon := monitor.New(bus, &buf)
	mon.Start()

	bus.Publish(events.Event{
		Type:   "task.completed",
		Source: "router",
		Payload: map[string]any{"task_id": "task_001"},
	})

	time.Sleep(50 * time.Millisecond)

	stats := mon.Stats()
	if stats.TotalEvents == 0 {
		t.Error("should have captured at least 1 event")
	}
}

func TestMonitor_WritesJSON(t *testing.T) {
	bus := events.NewBus()
	var buf bytes.Buffer
	mon := monitor.New(bus, &buf)
	mon.Start()

	bus.Publish(events.Event{
		Type:   "worker.registered",
		Source: "registry",
	})

	time.Sleep(50 * time.Millisecond)

	// Check buffer has valid JSON
	output := buf.String()
	if output == "" {
		t.Fatal("no output written")
	}

	var logEntry map[string]any
	if err := json.Unmarshal([]byte(output[:len(output)-1]), &logEntry); err != nil {
		// Try first line only
		lines := bytes.Split(buf.Bytes(), []byte("\n"))
		if err := json.Unmarshal(lines[0], &logEntry); err != nil {
			t.Fatalf("invalid JSON log: %v\nOutput: %s", err, output)
		}
	}

	if logEntry["event_type"] != "worker.registered" {
		t.Errorf("event_type: got %v", logEntry["event_type"])
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/monitor/ -v`
Expected: FAIL

- [ ] **Step 3: Implement monitor**

Create `core/internal/monitor/logger.go`:
```go
package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/kienbm/magic-claw/core/internal/events"
)

type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	EventType string         `json:"event_type"`
	Source    string         `json:"source"`
	Payload   map[string]any `json:"payload,omitempty"`
}

func writeLogEntry(w io.Writer, e events.Event) {
	level := "info"
	switch e.Severity {
	case "warn":
		level = "warn"
	case "error", "critical":
		level = "error"
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		EventType: e.Type,
		Source:    e.Source,
		Payload:   e.Payload,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "%s\n", data)
}
```

Create `core/internal/monitor/monitor.go`:
```go
package monitor

import (
	"io"
	"sync/atomic"

	"github.com/kienbm/magic-claw/core/internal/events"
)

type Stats struct {
	TotalEvents  int64 `json:"total_events"`
	TasksRouted  int64 `json:"tasks_routed"`
	TasksDone    int64 `json:"tasks_done"`
	TasksFailed  int64 `json:"tasks_failed"`
	WorkersCount int64 `json:"workers_count"`
}

type Monitor struct {
	bus    *events.Bus
	writer io.Writer
	stats  Stats
}

func New(bus *events.Bus, writer io.Writer) *Monitor {
	return &Monitor{bus: bus, writer: writer}
}

func (m *Monitor) Start() {
	m.bus.Subscribe("*", func(e events.Event) {
		atomic.AddInt64(&m.stats.TotalEvents, 1)

		switch e.Type {
		case "task.routed":
			atomic.AddInt64(&m.stats.TasksRouted, 1)
		case "task.completed":
			atomic.AddInt64(&m.stats.TasksDone, 1)
		case "task.failed":
			atomic.AddInt64(&m.stats.TasksFailed, 1)
		case "worker.registered":
			atomic.AddInt64(&m.stats.WorkersCount, 1)
		case "worker.deregistered":
			atomic.AddInt64(&m.stats.WorkersCount, -1)
		}

		writeLogEntry(m.writer, e)
	})
}

func (m *Monitor) Stats() Stats {
	return Stats{
		TotalEvents:  atomic.LoadInt64(&m.stats.TotalEvents),
		TasksRouted:  atomic.LoadInt64(&m.stats.TasksRouted),
		TasksDone:    atomic.LoadInt64(&m.stats.TasksDone),
		TasksFailed:  atomic.LoadInt64(&m.stats.TasksFailed),
		WorkersCount: atomic.LoadInt64(&m.stats.WorkersCount),
	}
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/monitor/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add core/internal/monitor/
git commit -m "feat(monitor): add event monitoring with JSON logging and stats"
```

---

### Task 9: Gateway — HTTP server + handlers

**Files:**
- Create: `core/internal/gateway/gateway.go`
- Create: `core/internal/gateway/middleware.go`
- Create: `core/internal/gateway/handlers.go`
- Create: `core/internal/gateway/gateway_test.go`
- Modify: `core/cmd/magic/main.go`

- [ ] **Step 1: Write test for gateway HTTP handlers**

Create `core/internal/gateway/gateway_test.go`:
```go
package gateway_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/gateway"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/registry"
	"github.com/kienbm/magic-claw/core/internal/router"
	"github.com/kienbm/magic-claw/core/internal/store"
	"github.com/kienbm/magic-claw/core/internal/monitor"
)

func setupGateway() *gateway.Gateway {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)
	rt := router.New(reg, s, bus)
	mon := monitor.New(bus, os.Stderr)
	mon.Start()
	return gateway.New(reg, rt, s, bus, mon)
}

func TestGateway_Health(t *testing.T) {
	gw := setupGateway()
	srv := httptest.NewServer(gw.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestGateway_RegisterWorker(t *testing.T) {
	gw := setupGateway()
	srv := httptest.NewServer(gw.Handler())
	defer srv.Close()

	payload := protocol.RegisterPayload{
		Name:         "TestBot",
		Capabilities: []protocol.Capability{{Name: "greeting"}},
		Endpoint:     protocol.Endpoint{Type: "http", URL: "http://localhost:9000"},
		Limits:       protocol.WorkerLimits{MaxConcurrentTasks: 5},
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(srv.URL+"/api/v1/workers/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("status: got %d, want 201", resp.StatusCode)
	}

	var result protocol.Worker
	json.NewDecoder(resp.Body).Decode(&result)
	if result.ID == "" {
		t.Error("worker ID should not be empty")
	}
}

func TestGateway_ListWorkers(t *testing.T) {
	gw := setupGateway()
	srv := httptest.NewServer(gw.Handler())
	defer srv.Close()

	// Register a worker first
	payload := protocol.RegisterPayload{
		Name:     "TestBot",
		Endpoint: protocol.Endpoint{Type: "http", URL: "http://localhost:9000"},
	}
	body, _ := json.Marshal(payload)
	http.Post(srv.URL+"/api/v1/workers/register", "application/json", bytes.NewReader(body))

	resp, _ := http.Get(srv.URL + "/api/v1/workers")
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d", resp.StatusCode)
	}

	var workers []*protocol.Worker
	json.NewDecoder(resp.Body).Decode(&workers)
	if len(workers) != 1 {
		t.Errorf("workers count: got %d, want 1", len(workers))
	}
}

func TestGateway_SubmitTask(t *testing.T) {
	gw := setupGateway()
	srv := httptest.NewServer(gw.Handler())
	defer srv.Close()

	// Register a capable worker
	regPayload := protocol.RegisterPayload{
		Name:         "GreetBot",
		Capabilities: []protocol.Capability{{Name: "greeting"}},
		Endpoint:     protocol.Endpoint{Type: "http", URL: "http://localhost:9000"},
		Limits:       protocol.WorkerLimits{MaxConcurrentTasks: 5},
	}
	body, _ := json.Marshal(regPayload)
	http.Post(srv.URL+"/api/v1/workers/register", "application/json", bytes.NewReader(body))

	// Submit task
	taskReq := map[string]any{
		"type":  "greeting",
		"input": map[string]string{"name": "Kien"},
		"routing": map[string]any{
			"strategy":              "best_match",
			"required_capabilities": []string{"greeting"},
		},
		"contract": map[string]any{
			"timeout_ms": 30000,
			"max_cost":   1.0,
		},
	}
	body, _ = json.Marshal(taskReq)
	resp, err := http.Post(srv.URL+"/api/v1/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("status: got %d, want 201", resp.StatusCode)
	}

	var task protocol.Task
	json.NewDecoder(resp.Body).Decode(&task)
	if task.Status != protocol.TaskAssigned {
		t.Errorf("status: got %q, want assigned", task.Status)
	}
	if task.AssignedWorker == "" {
		t.Error("assigned_worker should not be empty")
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/gateway/ -v`
Expected: FAIL

- [ ] **Step 3: Implement gateway**

Create `core/internal/gateway/middleware.go`:
```go
package gateway

import (
	"net/http"

	"github.com/kienbm/magic-claw/core/internal/protocol"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = protocol.GenerateID("req")
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

Create `core/internal/gateway/handlers.go`:
```go
package gateway

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kienbm/magic-claw/core/internal/protocol"
)

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"version": "0.1.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (g *Gateway) handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	var payload protocol.RegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	worker, err := g.registry.Register(payload)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(worker)
}

func (g *Gateway) handleListWorkers(w http.ResponseWriter, r *http.Request) {
	workers := g.registry.ListWorkers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}

func (g *Gateway) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var payload protocol.HeartbeatPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}
	if err := g.registry.Heartbeat(payload); err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (g *Gateway) handleSubmitTask(w http.ResponseWriter, r *http.Request) {
	var task protocol.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	task.ID = protocol.GenerateID("task")
	task.Status = protocol.TaskPending
	task.CreatedAt = time.Now()

	if task.Priority == "" {
		task.Priority = protocol.PriorityNormal
	}

	// Route task to best worker
	worker, err := g.router.RouteTask(&task)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusServiceUnavailable)
		return
	}

	_ = worker // Will be used to send task.assign in future

	// Store task
	g.store.AddTask(&task)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (g *Gateway) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := g.monitor.Stats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
```

Create `core/internal/gateway/gateway.go`:
```go
package gateway

import (
	"net/http"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/monitor"
	"github.com/kienbm/magic-claw/core/internal/registry"
	"github.com/kienbm/magic-claw/core/internal/router"
	"github.com/kienbm/magic-claw/core/internal/store"
)

type Gateway struct {
	registry *registry.Registry
	router   *router.Router
	store    store.Store
	bus      *events.Bus
	monitor  *monitor.Monitor
}

func New(reg *registry.Registry, rt *router.Router, s store.Store, bus *events.Bus, mon *monitor.Monitor) *Gateway {
	return &Gateway{
		registry: reg,
		router:   rt,
		store:    s,
		bus:      bus,
		monitor:  mon,
	}
}

func (g *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", g.handleHealth)

	// Workers
	mux.HandleFunc("POST /api/v1/workers/register", g.handleRegisterWorker)
	mux.HandleFunc("POST /api/v1/workers/heartbeat", g.handleHeartbeat)
	mux.HandleFunc("GET /api/v1/workers", g.handleListWorkers)

	// Tasks
	mux.HandleFunc("POST /api/v1/tasks", g.handleSubmitTask)

	// Metrics
	mux.HandleFunc("GET /api/v1/metrics", g.handleGetStats)

	// Apply middleware
	var handler http.Handler = mux
	handler = requestIDMiddleware(handler)
	handler = corsMiddleware(handler)

	return handler
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/gateway/ -v`
Expected: PASS (4 tests)

- [ ] **Step 5: Wire up main.go**

Update `core/cmd/magic/main.go`:
```go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/gateway"
	"github.com/kienbm/magic-claw/core/internal/monitor"
	"github.com/kienbm/magic-claw/core/internal/registry"
	"github.com/kienbm/magic-claw/core/internal/router"
	"github.com/kienbm/magic-claw/core/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("MagiC — Where AI becomes a Company")
		fmt.Println("Usage: magic <command>")
		fmt.Println("Commands: serve")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "serve":
		runServer()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runServer() {
	port := os.Getenv("MAGIC_PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize components
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)
	rt := router.New(reg, s, bus)
	mon := monitor.New(bus, os.Stdout)
	mon.Start()
	reg.StartHealthCheck(30_000_000_000) // 30s

	gw := gateway.New(reg, rt, s, bus, mon)

	fmt.Printf("MagiC server starting on :%s\n", port)
	fmt.Println("  POST /api/v1/workers/register  — Register a worker")
	fmt.Println("  GET  /api/v1/workers           — List workers")
	fmt.Println("  POST /api/v1/tasks             — Submit a task")
	fmt.Println("  GET  /api/v1/metrics           — View stats")
	fmt.Println("  GET  /health                   — Health check")

	if err := http.ListenAndServe(":"+port, gw.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Run all tests**

Run: `cd core && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 7: Build and test manually**

```bash
make build
./bin/magic serve &
# In another terminal:
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/v1/workers/register \
  -H "Content-Type: application/json" \
  -d '{"name":"TestBot","capabilities":[{"name":"greeting"}],"endpoint":{"type":"http","url":"http://localhost:9000"},"limits":{"max_concurrent_tasks":5}}'
curl http://localhost:8080/api/v1/workers
kill %1
```

- [ ] **Step 8: Commit**

```bash
git add core/internal/gateway/ core/cmd/magic/main.go
git commit -m "feat(gateway): add HTTP server with worker registration, task routing, health check"
```

---

## Chunk 3: Python SDK + Hello Worker

### Task 10: Python SDK

**Files:**
- Create: `sdk/python/magic_claw/__init__.py`
- Create: `sdk/python/magic_claw/client.py`
- Create: `sdk/python/magic_claw/worker.py`
- Create: `sdk/python/magic_claw/protocol.py`
- Create: `sdk/python/magic_claw/decorators.py`
- Create: `sdk/python/pyproject.toml`
- Create: `sdk/python/tests/test_worker.py`

- [ ] **Step 1: Create pyproject.toml**

Create `sdk/python/pyproject.toml`:
```toml
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "magic-claw"
version = "0.1.0"
description = "MagiC Python SDK — Build AI workers for the MagiC framework"
requires-python = ">=3.11"
dependencies = ["httpx>=0.27"]

[project.optional-dependencies]
dev = ["pytest>=8.0"]
```

- [ ] **Step 2: Implement protocol types**

Create `sdk/python/magic_claw/__init__.py`:
```python
from magic_claw.worker import Worker
from magic_claw.decorators import capability

__all__ = ["Worker", "capability"]
```

Create `sdk/python/magic_claw/protocol.py`:
```python
from dataclasses import dataclass, field
from typing import Any

@dataclass
class Capability:
    name: str
    description: str = ""
    est_cost_per_call: float = 0.0

@dataclass
class RegisterPayload:
    name: str
    capabilities: list[dict]
    endpoint: dict
    limits: dict = field(default_factory=lambda: {"max_concurrent_tasks": 5})
    metadata: dict = field(default_factory=dict)
```

- [ ] **Step 3: Implement HTTP client**

Create `sdk/python/magic_claw/client.py`:
```python
import httpx

class MagiCClient:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip("/")
        self._client = httpx.Client(base_url=self.base_url, timeout=30)

    def register_worker(self, payload: dict) -> dict:
        resp = self._client.post("/api/v1/workers/register", json=payload)
        resp.raise_for_status()
        return resp.json()

    def heartbeat(self, worker_id: str, current_load: int = 0) -> dict:
        resp = self._client.post("/api/v1/workers/heartbeat", json={
            "worker_id": worker_id,
            "current_load": current_load,
            "status": "active",
        })
        resp.raise_for_status()
        return resp.json()

    def health(self) -> dict:
        resp = self._client.get("/health")
        resp.raise_for_status()
        return resp.json()
```

- [ ] **Step 4: Implement Worker class and decorators**

Create `sdk/python/magic_claw/decorators.py`:
```python
def capability(name: str, description: str = "", est_cost: float = 0.0):
    def decorator(func):
        func._magic_capability = {
            "name": name,
            "description": description or func.__doc__ or "",
            "est_cost_per_call": est_cost,
        }
        return func
    return decorator
```

Create `sdk/python/magic_claw/worker.py`:
```python
import json
import threading
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from typing import Callable

from magic_claw.client import MagiCClient

class Worker:
    def __init__(self, name: str, endpoint: str = "http://localhost:9000"):
        self.name = name
        self.endpoint = endpoint
        self._capabilities: dict[str, dict] = {}
        self._handlers: dict[str, Callable] = {}
        self._worker_id: str | None = None
        self._client: MagiCClient | None = None

    def capability(self, name: str, description: str = "", est_cost: float = 0.0):
        def decorator(func):
            self._capabilities[name] = {
                "name": name,
                "description": description or func.__doc__ or "",
                "est_cost_per_call": est_cost,
            }
            self._handlers[name] = func
            return func
        return decorator

    def register(self, magic_url: str):
        self._client = MagiCClient(magic_url)
        payload = {
            "name": self.name,
            "capabilities": list(self._capabilities.values()),
            "endpoint": {"type": "http", "url": self.endpoint},
            "limits": {"max_concurrent_tasks": 5},
        }
        result = self._client.register_worker(payload)
        self._worker_id = result.get("id")
        print(f"Registered as {self._worker_id}")
        return self

    def _start_heartbeat(self, interval: int = 30):
        def loop():
            while True:
                time.sleep(interval)
                if self._client and self._worker_id:
                    try:
                        self._client.heartbeat(self._worker_id)
                    except Exception:
                        pass
        t = threading.Thread(target=loop, daemon=True)
        t.start()

    def handle_task(self, task_type: str, input_data: dict) -> dict:
        handler = self._handlers.get(task_type)
        if not handler:
            raise ValueError(f"No handler for {task_type}")
        result = handler(**input_data)
        if isinstance(result, str):
            return {"result": result}
        return result

    def serve(self, host: str = "0.0.0.0", port: int = 9000):
        worker = self
        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                length = int(self.headers.get("Content-Length", 0))
                body = json.loads(self.rfile.read(length))
                msg_type = body.get("type", "")
                payload = body.get("payload", {})

                if msg_type == "task.assign":
                    try:
                        result = worker.handle_task(payload.get("task_type", ""), payload.get("input", {}))
                        response = {"type": "task.complete", "payload": {"task_id": payload.get("task_id"), "output": result}}
                    except Exception as e:
                        response = {"type": "task.fail", "payload": {"task_id": payload.get("task_id"), "error": {"message": str(e)}}}
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.end_headers()
                    self.wfile.write(json.dumps(response).encode())
                else:
                    self.send_response(404)
                    self.end_headers()

            def log_message(self, format, *args):
                pass  # suppress default logging

        self._start_heartbeat()
        parsed = self.endpoint.split(":")
        port = int(parsed[-1].split("/")[0]) if len(parsed) > 2 else port
        server = HTTPServer((host, port), Handler)
        print(f"{self.name} serving on {host}:{port}")
        server.serve_forever()
```

- [ ] **Step 5: Write test**

Create `sdk/python/tests/test_worker.py`:
```python
from magic_claw import Worker

def test_worker_capability_registration():
    w = Worker(name="TestBot")

    @w.capability("greeting", description="Says hello")
    def greet(name: str) -> str:
        return f"Hello, {name}!"

    assert "greeting" in w._capabilities
    assert w._capabilities["greeting"]["name"] == "greeting"

def test_worker_handle_task():
    w = Worker(name="TestBot")

    @w.capability("greeting")
    def greet(name: str) -> str:
        return f"Hello, {name}!"

    result = w.handle_task("greeting", {"name": "Kien"})
    assert result == {"result": "Hello, Kien!"}

def test_worker_handle_unknown_task():
    w = Worker(name="TestBot")
    try:
        w.handle_task("nonexistent", {})
        assert False, "should raise"
    except ValueError:
        pass
```

- [ ] **Step 6: Run tests**

```bash
cd sdk/python
pip install -e ".[dev]"
pytest tests/ -v
```
Expected: PASS (3 tests)

- [ ] **Step 7: Commit**

```bash
git add sdk/python/
git commit -m "feat(sdk): add Python SDK with Worker class, capability decorator, MagiC client"
```

---

### Task 11: Hello Worker example

**Files:**
- Create: `examples/hello-worker/main.py`

- [ ] **Step 1: Create hello worker (< 20 lines)**

Create `examples/hello-worker/main.py`:
```python
from magic_claw import Worker

worker = Worker(name="HelloBot", endpoint="http://localhost:9000")

@worker.capability("greeting", description="Says hello to anyone")
def greet(name: str) -> str:
    return f"Hello, {name}! I'm managed by MagiC."

if __name__ == "__main__":
    worker.register("http://localhost:8080")
    worker.serve()
```

- [ ] **Step 2: Test end-to-end manually**

Terminal 1:
```bash
make run  # starts MagiC server on :8080
```

Terminal 2:
```bash
cd examples/hello-worker
python main.py  # registers + starts worker on :9000
```

Terminal 3:
```bash
# Check worker registered
curl http://localhost:8080/api/v1/workers | python -m json.tool

# Submit task
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"type":"greeting","input":{"name":"Kien"},"routing":{"strategy":"best_match","required_capabilities":["greeting"]},"contract":{"timeout_ms":30000,"max_cost":1.0}}'
```

- [ ] **Step 3: Commit**

```bash
git add examples/hello-worker/
git commit -m "feat(examples): add hello-worker — 10 line Python worker example"
```

---

### Task 12: Run all tests + final commit

- [ ] **Step 1: Run all Go tests**

```bash
cd core && go test ./... -v -count=1
```
Expected: ALL PASS (14+ tests across 5 packages)

- [ ] **Step 2: Run Python tests**

```bash
cd sdk/python && pytest tests/ -v
```
Expected: ALL PASS (3 tests)

- [ ] **Step 3: Final commit**

```bash
git add -A
git commit -m "feat: MagiC v0.1.0 — Foundation + Core Modules (Gateway, Registry, Router, Monitor) + Python SDK"
```

---

## Summary

After completing this plan, you have:

- **MagiC server** (Go) running on port 8080 with:
  - Gateway: HTTP API with health check, CORS, request IDs
  - Registry: Worker registration, heartbeat, health monitoring
  - Router: Task routing with best_match and cheapest strategies
  - Monitor: Event bus, structured JSON logging, stats
  - Store: In-memory storage with thread-safe operations
  - Protocol: Full MCP² message types and entity definitions

- **Python SDK** (`pip install magic-claw`) with:
  - Worker class with `@capability` decorator
  - MagiC HTTP client
  - Auto-heartbeat

- **Hello Worker example** in 10 lines of Python

**Next:** Plan 2 will add Tier 2 modules (Orchestrator, Evaluator, Cost Controller, Org Manager).
