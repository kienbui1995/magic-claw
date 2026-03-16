# MagiC Plan 2: Tier 2 Modules (Orchestrator, Evaluator, Cost Controller, Org Manager)

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the 4 "differentiator" modules that give MagiC its competitive edge — multi-step workflow orchestration, output quality evaluation, cost tracking with budgets, and team/org management.

**Architecture:** Each module is a standalone Go package in `core/internal/` that receives dependencies (store, event bus, other modules) via constructor injection. Modules communicate through the event bus. The Gateway gets new HTTP endpoints. The Store interface is extended for Workflow and Team persistence.

**Tech Stack:** Go 1.22+, existing protocol types and event bus from Plan 1

**Spec:** `docs/superpowers/specs/2026-03-16-magic-framework-design.md` (sections 7.5–7.8, 8.2)

---

## File Map

```
magic-claw/core/
├── internal/
│   ├── protocol/
│   │   └── types.go               # ADD: Workflow, WorkflowStep, WorkflowStatus constants
│   ├── store/
│   │   ├── store.go               # EXTEND: add Workflow + Team methods to interface
│   │   └── memory.go              # EXTEND: implement new methods
│   ├── costctrl/
│   │   ├── controller.go          # Cost tracking, budget checking
│   │   └── controller_test.go
│   ├── evaluator/
│   │   ├── evaluator.go           # Schema validation + quality check
│   │   └── evaluator_test.go
│   ├── orgmgr/
│   │   ├── manager.go             # Team CRUD, worker assignment
│   │   └── manager_test.go
│   ├── orchestrator/
│   │   ├── orchestrator.go        # Workflow submission + step management
│   │   ├── dag.go                 # DAG execution engine
│   │   ├── orchestrator_test.go
│   │   └── dag_test.go
│   └── gateway/
│       ├── gateway.go             # MODIFY: add new module deps
│       └── handlers.go            # MODIFY: add new endpoints
└── cmd/magic/
    └── main.go                    # MODIFY: wire up new modules
```

---

## Chunk 1: Protocol + Store Extensions

### Task 1: Add Workflow types to protocol

**Files:**
- Modify: `core/internal/protocol/types.go`
- Modify: `core/internal/protocol/types_test.go`

- [ ] **Step 1: Write test for Workflow serialization**

Append to `core/internal/protocol/types_test.go`:
```go
func TestWorkflowSerialization(t *testing.T) {
	wf := protocol.Workflow{
		ID:   "wf_001",
		Name: "Product Launch",
		Steps: []protocol.WorkflowStep{
			{
				ID:       "research",
				TaskType: "market_research",
				Input:    json.RawMessage(`{"topic": "AI"}`),
			},
			{
				ID:        "content",
				TaskType:  "content_writing",
				DependsOn: []string{"research"},
				OnFailure: "retry",
			},
		},
		Status: protocol.WorkflowPending,
	}

	data, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var wf2 protocol.Workflow
	if err := json.Unmarshal(data, &wf2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if wf2.Name != "Product Launch" {
		t.Errorf("Name: got %q", wf2.Name)
	}
	if len(wf2.Steps) != 2 {
		t.Fatalf("Steps: got %d, want 2", len(wf2.Steps))
	}
	if wf2.Steps[1].DependsOn[0] != "research" {
		t.Errorf("DependsOn: got %v", wf2.Steps[1].DependsOn)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/protocol/ -v -run TestWorkflow`
Expected: FAIL — Workflow not defined

- [ ] **Step 3: Add Workflow and WorkflowStep types**

Append to `core/internal/protocol/types.go`:
```go
// Workflow statuses
const (
	WorkflowPending    = "pending"
	WorkflowRunning    = "running"
	WorkflowCompleted  = "completed"
	WorkflowFailed     = "failed"
	WorkflowAborted    = "aborted"
)

// Step statuses
const (
	StepPending   = "pending"
	StepRunning   = "running"
	StepCompleted = "completed"
	StepFailed    = "failed"
	StepSkipped   = "skipped"
	StepBlocked   = "blocked"
)

type WorkflowStep struct {
	ID        string          `json:"id"`
	TaskType  string          `json:"task_type"`
	Input     json.RawMessage `json:"input,omitempty"`
	DependsOn []string        `json:"depends_on,omitempty"`
	OnFailure string          `json:"on_failure,omitempty"` // retry | skip | abort | reassign
	Status    string          `json:"status,omitempty"`
	TaskID    string          `json:"task_id,omitempty"`    // assigned task ID
	Output    json.RawMessage `json:"output,omitempty"`
	Error     *TaskError      `json:"error,omitempty"`
}

type Workflow struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Steps     []WorkflowStep  `json:"steps"`
	Status    string          `json:"status"`
	Context   TaskContext     `json:"context"`
	CreatedAt time.Time       `json:"created_at"`
	DoneAt    *time.Time      `json:"done_at,omitempty"`
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/protocol/ -v`
Expected: PASS (6 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/protocol/
git commit -m "feat(protocol): add Workflow and WorkflowStep types"
```

---

### Task 2: Extend Store interface for Workflows and Teams

**Files:**
- Modify: `core/internal/store/store.go`
- Modify: `core/internal/store/memory.go`
- Modify: `core/internal/store/memory_test.go`

- [ ] **Step 1: Write tests for new store methods**

Append to `core/internal/store/memory_test.go`:
```go
func TestMemoryStore_Workflows(t *testing.T) {
	s := store.NewMemoryStore()

	wf := &protocol.Workflow{
		ID:     "wf_001",
		Name:   "Test Workflow",
		Status: protocol.WorkflowPending,
		Steps: []protocol.WorkflowStep{
			{ID: "step1", TaskType: "greeting", Status: protocol.StepPending},
		},
	}

	if err := s.AddWorkflow(wf); err != nil {
		t.Fatalf("AddWorkflow: %v", err)
	}

	got, err := s.GetWorkflow("wf_001")
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.Name != "Test Workflow" {
		t.Errorf("Name: got %q", got.Name)
	}

	wf.Status = protocol.WorkflowRunning
	if err := s.UpdateWorkflow(wf); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}

	got, _ = s.GetWorkflow("wf_001")
	if got.Status != protocol.WorkflowRunning {
		t.Errorf("Status: got %q", got.Status)
	}

	list := s.ListWorkflows()
	if len(list) != 1 {
		t.Errorf("ListWorkflows: got %d, want 1", len(list))
	}
}

func TestMemoryStore_Teams(t *testing.T) {
	s := store.NewMemoryStore()

	team := &protocol.Team{
		ID:          "team_001",
		Name:        "Marketing",
		OrgID:       "org_magic",
		DailyBudget: 10.0,
	}

	if err := s.AddTeam(team); err != nil {
		t.Fatalf("AddTeam: %v", err)
	}

	got, err := s.GetTeam("team_001")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.Name != "Marketing" {
		t.Errorf("Name: got %q", got.Name)
	}

	team.Workers = []string{"worker_001"}
	if err := s.UpdateTeam(team); err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}

	list := s.ListTeams()
	if len(list) != 1 {
		t.Errorf("ListTeams: got %d, want 1", len(list))
	}

	if err := s.RemoveTeam("team_001"); err != nil {
		t.Fatalf("RemoveTeam: %v", err)
	}
	if _, err := s.GetTeam("team_001"); err == nil {
		t.Error("GetTeam after remove should fail")
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/store/ -v -run TestMemoryStore_Workflows`
Expected: FAIL

- [ ] **Step 3: Extend Store interface**

Add to `core/internal/store/store.go` (after existing Task methods):
```go
	// Workflows
	AddWorkflow(w *protocol.Workflow) error
	GetWorkflow(id string) (*protocol.Workflow, error)
	UpdateWorkflow(w *protocol.Workflow) error
	ListWorkflows() []*protocol.Workflow

	// Teams
	AddTeam(t *protocol.Team) error
	GetTeam(id string) (*protocol.Team, error)
	UpdateTeam(t *protocol.Team) error
	RemoveTeam(id string) error
	ListTeams() []*protocol.Team
```

- [ ] **Step 4: Extend MemoryStore implementation**

Add to `core/internal/store/memory.go` — add `workflows` and `teams` maps to struct, update `NewMemoryStore`, and implement all new methods:

MemoryStore struct update:
```go
type MemoryStore struct {
	mu        sync.RWMutex
	workers   map[string]*protocol.Worker
	tasks     map[string]*protocol.Task
	workflows map[string]*protocol.Workflow
	teams     map[string]*protocol.Team
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workers:   make(map[string]*protocol.Worker),
		tasks:     make(map[string]*protocol.Task),
		workflows: make(map[string]*protocol.Workflow),
		teams:     make(map[string]*protocol.Team),
	}
}
```

New methods (follow same pattern as existing Worker/Task methods):
```go
func (s *MemoryStore) AddWorkflow(w *protocol.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[w.ID] = w
	return nil
}

func (s *MemoryStore) GetWorkflow(id string) (*protocol.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workflows[id]
	if !ok {
		return nil, ErrNotFound
	}
	return w, nil
}

func (s *MemoryStore) UpdateWorkflow(w *protocol.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workflows[w.ID]; !ok {
		return ErrNotFound
	}
	s.workflows[w.ID] = w
	return nil
}

func (s *MemoryStore) ListWorkflows() []*protocol.Workflow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*protocol.Workflow, 0, len(s.workflows))
	for _, w := range s.workflows {
		result = append(result, w)
	}
	return result
}

func (s *MemoryStore) AddTeam(t *protocol.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.teams[t.ID] = t
	return nil
}

func (s *MemoryStore) GetTeam(id string) (*protocol.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.teams[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *MemoryStore) UpdateTeam(t *protocol.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.teams[t.ID]; !ok {
		return ErrNotFound
	}
	s.teams[t.ID] = t
	return nil
}

func (s *MemoryStore) RemoveTeam(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.teams[id]; !ok {
		return ErrNotFound
	}
	delete(s.teams, id)
	return nil
}

func (s *MemoryStore) ListTeams() []*protocol.Team {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*protocol.Team, 0, len(s.teams))
	for _, t := range s.teams {
		result = append(result, t)
	}
	return result
}
```

- [ ] **Step 5: Run test — verify it passes**

Run: `cd core && go test ./internal/store/ -v`
Expected: PASS (4 tests)

- [ ] **Step 6: Commit**

```bash
git add core/internal/store/
git commit -m "feat(store): extend Store interface with Workflow and Team persistence"
```

---

## Chunk 2: Cost Controller

### Task 3: Cost Controller module

**Files:**
- Create: `core/internal/costctrl/controller.go`
- Create: `core/internal/costctrl/controller_test.go`

- [ ] **Step 1: Write test for cost controller**

Create `core/internal/costctrl/controller_test.go`:
```go
package costctrl_test

import (
	"testing"
	"time"

	"github.com/kienbm/magic-claw/core/internal/costctrl"
	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/store"
)

func TestCostController_RecordCost(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	cc := costctrl.New(s, bus)

	// Register a worker
	w := &protocol.Worker{ID: "worker_001", Name: "Bot", Status: protocol.StatusActive}
	s.AddWorker(w)

	// Record cost
	cc.RecordCost("worker_001", "task_001", 0.15)

	report := cc.WorkerReport("worker_001")
	if report.TotalCost != 0.15 {
		t.Errorf("TotalCost: got %f, want 0.15", report.TotalCost)
	}
	if report.TaskCount != 1 {
		t.Errorf("TaskCount: got %d, want 1", report.TaskCount)
	}
}

func TestCostController_BudgetAlert(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	cc := costctrl.New(s, bus)

	// Track budget alerts
	var alerts []events.Event
	bus.Subscribe("budget.threshold", func(e events.Event) {
		alerts = append(alerts, e)
	})

	w := &protocol.Worker{
		ID:     "worker_001",
		Name:   "Bot",
		Status: protocol.StatusActive,
		Limits: protocol.WorkerLimits{MaxCostPerDay: 1.0},
	}
	s.AddWorker(w)

	// Record costs that exceed 80% threshold
	cc.RecordCost("worker_001", "task_001", 0.85)

	time.Sleep(50 * time.Millisecond)

	if len(alerts) == 0 {
		t.Error("should have received budget alert at 80%")
	}
}

func TestCostController_AutoPause(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	cc := costctrl.New(s, bus)

	w := &protocol.Worker{
		ID:     "worker_001",
		Name:   "Bot",
		Status: protocol.StatusActive,
		Limits: protocol.WorkerLimits{MaxCostPerDay: 1.0},
	}
	s.AddWorker(w)

	// Exceed budget
	cc.RecordCost("worker_001", "task_001", 1.10)

	time.Sleep(50 * time.Millisecond)

	got, _ := s.GetWorker("worker_001")
	if got.Status != protocol.StatusPaused {
		t.Errorf("Status: got %q, want paused", got.Status)
	}
}

func TestCostController_OrgReport(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	cc := costctrl.New(s, bus)

	cc.RecordCost("w1", "t1", 0.10)
	cc.RecordCost("w2", "t2", 0.20)
	cc.RecordCost("w1", "t3", 0.05)

	report := cc.OrgReport()
	if report.TotalCost != 0.35 {
		t.Errorf("TotalCost: got %f, want 0.35", report.TotalCost)
	}
	if report.TaskCount != 3 {
		t.Errorf("TaskCount: got %d, want 3", report.TaskCount)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/costctrl/ -v`
Expected: FAIL

- [ ] **Step 3: Implement cost controller**

Create `core/internal/costctrl/controller.go`:
```go
package costctrl

import (
	"fmt"
	"sync"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/store"
)

type CostRecord struct {
	WorkerID string
	TaskID   string
	Cost     float64
}

type CostReport struct {
	TotalCost float64 `json:"total_cost"`
	TaskCount int     `json:"task_count"`
}

type Controller struct {
	store   store.Store
	bus     *events.Bus
	mu      sync.RWMutex
	records []CostRecord
}

func New(s store.Store, bus *events.Bus) *Controller {
	return &Controller{store: s, bus: bus}
}

func (c *Controller) RecordCost(workerID, taskID string, cost float64) {
	c.mu.Lock()
	c.records = append(c.records, CostRecord{
		WorkerID: workerID,
		TaskID:   taskID,
		Cost:     cost,
	})
	c.mu.Unlock()

	// Update worker's daily cost
	w, err := c.store.GetWorker(workerID)
	if err == nil {
		w.TotalCostToday += cost
		c.store.UpdateWorker(w)
		c.checkBudget(w)
	}

	c.bus.Publish(events.Event{
		Type:   "cost.recorded",
		Source: "costctrl",
		Payload: map[string]any{
			"worker_id": workerID,
			"task_id":   taskID,
			"cost":      cost,
		},
	})
}

func (c *Controller) checkBudget(w *protocol.Worker) {
	if w.Limits.MaxCostPerDay <= 0 {
		return
	}
	ratio := w.TotalCostToday / w.Limits.MaxCostPerDay

	if ratio >= 1.0 {
		// Auto-pause
		w.Status = protocol.StatusPaused
		c.store.UpdateWorker(w)
		c.bus.Publish(events.Event{
			Type:     "budget.exceeded",
			Source:   "costctrl",
			Severity: "error",
			Payload: map[string]any{
				"worker_id": w.ID,
				"spent":     w.TotalCostToday,
				"budget":    w.Limits.MaxCostPerDay,
			},
		})
	} else if ratio >= 0.8 {
		c.bus.Publish(events.Event{
			Type:     "budget.threshold",
			Source:   "costctrl",
			Severity: "warn",
			Payload: map[string]any{
				"worker_id": w.ID,
				"percent":   fmt.Sprintf("%.0f%%", ratio*100),
				"spent":     w.TotalCostToday,
				"budget":    w.Limits.MaxCostPerDay,
			},
		})
	}
}

func (c *Controller) WorkerReport(workerID string) CostReport {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var report CostReport
	for _, r := range c.records {
		if r.WorkerID == workerID {
			report.TotalCost += r.Cost
			report.TaskCount++
		}
	}
	return report
}

func (c *Controller) OrgReport() CostReport {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var report CostReport
	for _, r := range c.records {
		report.TotalCost += r.Cost
		report.TaskCount++
	}
	return report
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/costctrl/ -v`
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/costctrl/
git commit -m "feat(costctrl): add cost tracking, budget alerts, auto-pause"
```

---

## Chunk 3: Evaluator

### Task 4: Evaluator module

**Files:**
- Create: `core/internal/evaluator/evaluator.go`
- Create: `core/internal/evaluator/evaluator_test.go`

- [ ] **Step 1: Write test for evaluator**

Create `core/internal/evaluator/evaluator_test.go`:
```go
package evaluator_test

import (
	"encoding/json"
	"testing"

	"github.com/kienbm/magic-claw/core/internal/evaluator"
	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
)

func TestEvaluator_SchemaValidation_Pass(t *testing.T) {
	bus := events.NewBus()
	ev := evaluator.New(bus)

	schema := json.RawMessage(`{
		"type": "object",
		"required": ["title", "body"],
		"properties": {
			"title": {"type": "string"},
			"body": {"type": "string"}
		}
	}`)

	output := json.RawMessage(`{"title": "Hello", "body": "World"}`)

	result := ev.Evaluate(output, protocol.Contract{
		OutputSchema: schema,
	})

	if !result.Pass {
		t.Errorf("should pass, got errors: %v", result.Errors)
	}
}

func TestEvaluator_SchemaValidation_MissingRequired(t *testing.T) {
	bus := events.NewBus()
	ev := evaluator.New(bus)

	schema := json.RawMessage(`{
		"type": "object",
		"required": ["title", "body"],
		"properties": {
			"title": {"type": "string"},
			"body": {"type": "string"}
		}
	}`)

	output := json.RawMessage(`{"title": "Hello"}`)

	result := ev.Evaluate(output, protocol.Contract{
		OutputSchema: schema,
	})

	if result.Pass {
		t.Error("should fail — missing required field 'body'")
	}
	if len(result.Errors) == 0 {
		t.Error("should have at least one error")
	}
}

func TestEvaluator_SchemaValidation_WrongType(t *testing.T) {
	bus := events.NewBus()
	ev := evaluator.New(bus)

	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"count": {"type": "number"}
		}
	}`)

	output := json.RawMessage(`{"count": "not a number"}`)

	result := ev.Evaluate(output, protocol.Contract{
		OutputSchema: schema,
	})

	if result.Pass {
		t.Error("should fail — wrong type for 'count'")
	}
}

func TestEvaluator_NoSchema(t *testing.T) {
	bus := events.NewBus()
	ev := evaluator.New(bus)

	output := json.RawMessage(`{"anything": "goes"}`)

	result := ev.Evaluate(output, protocol.Contract{})

	if !result.Pass {
		t.Error("should pass when no schema specified")
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/evaluator/ -v`
Expected: FAIL

- [ ] **Step 3: Implement evaluator**

Create `core/internal/evaluator/evaluator.go`:
```go
package evaluator

import (
	"encoding/json"
	"fmt"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
)

type Result struct {
	Pass   bool     `json:"pass"`
	Errors []string `json:"errors,omitempty"`
}

type Evaluator struct {
	bus *events.Bus
}

func New(bus *events.Bus) *Evaluator {
	return &Evaluator{bus: bus}
}

func (e *Evaluator) Evaluate(output json.RawMessage, contract protocol.Contract) Result {
	var result Result
	result.Pass = true

	if len(contract.OutputSchema) > 0 {
		schemaErrors := validateSchema(output, contract.OutputSchema)
		if len(schemaErrors) > 0 {
			result.Pass = false
			result.Errors = append(result.Errors, schemaErrors...)
		}
	}

	if !result.Pass {
		e.bus.Publish(events.Event{
			Type:     "evaluation.failed",
			Source:   "evaluator",
			Severity: "warn",
			Payload:  map[string]any{"errors": result.Errors},
		})
	}

	return result
}

// validateSchema performs basic JSON schema validation.
// Supports: type checking (object, string, number, boolean, array),
// required fields, and properties type checking.
func validateSchema(data json.RawMessage, schema json.RawMessage) []string {
	var schemaMap map[string]any
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return []string{fmt.Sprintf("invalid schema: %v", err)}
	}

	var dataVal any
	if err := json.Unmarshal(data, &dataVal); err != nil {
		return []string{fmt.Sprintf("invalid JSON output: %v", err)}
	}

	return validateValue(dataVal, schemaMap)
}

func validateValue(val any, schema map[string]any) []string {
	var errors []string

	// Type check
	if expectedType, ok := schema["type"].(string); ok {
		if !checkType(val, expectedType) {
			return []string{fmt.Sprintf("expected type %q, got %T", expectedType, val)}
		}
	}

	// Object-specific: required fields and properties
	if obj, ok := val.(map[string]any); ok {
		// Check required
		if reqRaw, ok := schema["required"].([]any); ok {
			for _, r := range reqRaw {
				field := fmt.Sprint(r)
				if _, exists := obj[field]; !exists {
					errors = append(errors, fmt.Sprintf("missing required field %q", field))
				}
			}
		}

		// Check property types
		if props, ok := schema["properties"].(map[string]any); ok {
			for fieldName, propSchema := range props {
				fieldVal, exists := obj[fieldName]
				if !exists {
					continue // not required, skip
				}
				if propMap, ok := propSchema.(map[string]any); ok {
					fieldErrors := validateValue(fieldVal, propMap)
					for _, e := range fieldErrors {
						errors = append(errors, fmt.Sprintf("field %q: %s", fieldName, e))
					}
				}
			}
		}
	}

	return errors
}

func checkType(val any, expected string) bool {
	switch expected {
	case "object":
		_, ok := val.(map[string]any)
		return ok
	case "array":
		_, ok := val.([]any)
		return ok
	case "string":
		_, ok := val.(string)
		return ok
	case "number":
		_, ok := val.(float64)
		return ok
	case "boolean":
		_, ok := val.(bool)
		return ok
	case "null":
		return val == nil
	}
	return true
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/evaluator/ -v`
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/evaluator/
git commit -m "feat(evaluator): add JSON schema validation for task output"
```

---

## Chunk 4: Org Manager

### Task 5: Org Manager module

**Files:**
- Create: `core/internal/orgmgr/manager.go`
- Create: `core/internal/orgmgr/manager_test.go`

- [ ] **Step 1: Write test for org manager**

Create `core/internal/orgmgr/manager_test.go`:
```go
package orgmgr_test

import (
	"testing"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/orgmgr"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/store"
)

func TestOrgManager_CreateTeam(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	mgr := orgmgr.New(s, bus)

	team, err := mgr.CreateTeam("Marketing", "org_magic", 10.0)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if team.ID == "" {
		t.Error("team ID should not be empty")
	}
	if team.Name != "Marketing" {
		t.Errorf("Name: got %q", team.Name)
	}
	if team.DailyBudget != 10.0 {
		t.Errorf("DailyBudget: got %f", team.DailyBudget)
	}
}

func TestOrgManager_AssignWorker(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	mgr := orgmgr.New(s, bus)

	team, _ := mgr.CreateTeam("Marketing", "org_magic", 10.0)

	w := &protocol.Worker{ID: "worker_001", Name: "Bot", Status: protocol.StatusActive}
	s.AddWorker(w)

	err := mgr.AssignWorker(team.ID, "worker_001")
	if err != nil {
		t.Fatalf("AssignWorker: %v", err)
	}

	got, _ := s.GetTeam(team.ID)
	if len(got.Workers) != 1 || got.Workers[0] != "worker_001" {
		t.Errorf("Workers: got %v", got.Workers)
	}

	gotW, _ := s.GetWorker("worker_001")
	if gotW.TeamID != team.ID {
		t.Errorf("TeamID: got %q", gotW.TeamID)
	}
}

func TestOrgManager_RemoveWorker(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	mgr := orgmgr.New(s, bus)

	team, _ := mgr.CreateTeam("Marketing", "org_magic", 10.0)
	w := &protocol.Worker{ID: "worker_001", Name: "Bot", Status: protocol.StatusActive}
	s.AddWorker(w)
	mgr.AssignWorker(team.ID, "worker_001")

	err := mgr.RemoveWorker(team.ID, "worker_001")
	if err != nil {
		t.Fatalf("RemoveWorker: %v", err)
	}

	got, _ := s.GetTeam(team.ID)
	if len(got.Workers) != 0 {
		t.Errorf("Workers: got %v, want empty", got.Workers)
	}

	gotW, _ := s.GetWorker("worker_001")
	if gotW.TeamID != "" {
		t.Errorf("TeamID: got %q, want empty", gotW.TeamID)
	}
}

func TestOrgManager_ListTeams(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	mgr := orgmgr.New(s, bus)

	mgr.CreateTeam("Marketing", "org_magic", 10.0)
	mgr.CreateTeam("Sales", "org_magic", 15.0)

	teams := mgr.ListTeams()
	if len(teams) != 2 {
		t.Errorf("ListTeams: got %d, want 2", len(teams))
	}
}

func TestOrgManager_DeleteTeam(t *testing.T) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	mgr := orgmgr.New(s, bus)

	team, _ := mgr.CreateTeam("Marketing", "org_magic", 10.0)

	err := mgr.DeleteTeam(team.ID)
	if err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}

	teams := mgr.ListTeams()
	if len(teams) != 0 {
		t.Errorf("ListTeams after delete: got %d", len(teams))
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/orgmgr/ -v`
Expected: FAIL

- [ ] **Step 3: Implement org manager**

Create `core/internal/orgmgr/manager.go`:
```go
package orgmgr

import (
	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/store"
)

type Manager struct {
	store store.Store
	bus   *events.Bus
}

func New(s store.Store, bus *events.Bus) *Manager {
	return &Manager{store: s, bus: bus}
}

func (m *Manager) CreateTeam(name, orgID string, dailyBudget float64) (*protocol.Team, error) {
	team := &protocol.Team{
		ID:          protocol.GenerateID("team"),
		Name:        name,
		OrgID:       orgID,
		DailyBudget: dailyBudget,
	}

	if err := m.store.AddTeam(team); err != nil {
		return nil, err
	}

	m.bus.Publish(events.Event{
		Type:   "team.created",
		Source: "orgmgr",
		Payload: map[string]any{
			"team_id":   team.ID,
			"team_name": team.Name,
		},
	})

	return team, nil
}

func (m *Manager) DeleteTeam(teamID string) error {
	if err := m.store.RemoveTeam(teamID); err != nil {
		return err
	}

	m.bus.Publish(events.Event{
		Type:   "team.deleted",
		Source: "orgmgr",
		Payload: map[string]any{"team_id": teamID},
	})

	return nil
}

func (m *Manager) ListTeams() []*protocol.Team {
	return m.store.ListTeams()
}

func (m *Manager) GetTeam(id string) (*protocol.Team, error) {
	return m.store.GetTeam(id)
}

func (m *Manager) AssignWorker(teamID, workerID string) error {
	team, err := m.store.GetTeam(teamID)
	if err != nil {
		return err
	}

	worker, err := m.store.GetWorker(workerID)
	if err != nil {
		return err
	}

	// Add worker to team
	team.Workers = append(team.Workers, workerID)
	if err := m.store.UpdateTeam(team); err != nil {
		return err
	}

	// Set worker's team
	worker.TeamID = teamID
	if err := m.store.UpdateWorker(worker); err != nil {
		return err
	}

	m.bus.Publish(events.Event{
		Type:   "team.worker_assigned",
		Source: "orgmgr",
		Payload: map[string]any{
			"team_id":   teamID,
			"worker_id": workerID,
		},
	})

	return nil
}

func (m *Manager) RemoveWorker(teamID, workerID string) error {
	team, err := m.store.GetTeam(teamID)
	if err != nil {
		return err
	}

	// Remove worker from team list
	var updated []string
	for _, id := range team.Workers {
		if id != workerID {
			updated = append(updated, id)
		}
	}
	team.Workers = updated
	if err := m.store.UpdateTeam(team); err != nil {
		return err
	}

	// Clear worker's team
	worker, err := m.store.GetWorker(workerID)
	if err != nil {
		return err
	}
	worker.TeamID = ""
	if err := m.store.UpdateWorker(worker); err != nil {
		return err
	}

	m.bus.Publish(events.Event{
		Type:   "team.worker_removed",
		Source: "orgmgr",
		Payload: map[string]any{
			"team_id":   teamID,
			"worker_id": workerID,
		},
	})

	return nil
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/orgmgr/ -v`
Expected: PASS (5 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/orgmgr/
git commit -m "feat(orgmgr): add team CRUD and worker assignment"
```

---

## Chunk 5: Orchestrator

### Task 6: DAG execution engine

**Files:**
- Create: `core/internal/orchestrator/dag.go`
- Create: `core/internal/orchestrator/dag_test.go`

- [ ] **Step 1: Write test for DAG**

Create `core/internal/orchestrator/dag_test.go`:
```go
package orchestrator_test

import (
	"testing"

	"github.com/kienbm/magic-claw/core/internal/orchestrator"
	"github.com/kienbm/magic-claw/core/internal/protocol"
)

func TestDAG_FindReady_NoDeps(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a", TaskType: "task_a", Status: protocol.StepPending},
		{ID: "b", TaskType: "task_b", Status: protocol.StepPending, DependsOn: []string{"a"}},
	}

	ready := orchestrator.FindReadySteps(steps)
	if len(ready) != 1 || ready[0] != "a" {
		t.Errorf("ready: got %v, want [a]", ready)
	}
}

func TestDAG_FindReady_DepCompleted(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a", TaskType: "task_a", Status: protocol.StepCompleted},
		{ID: "b", TaskType: "task_b", Status: protocol.StepPending, DependsOn: []string{"a"}},
		{ID: "c", TaskType: "task_c", Status: protocol.StepPending, DependsOn: []string{"a"}},
	}

	ready := orchestrator.FindReadySteps(steps)
	if len(ready) != 2 {
		t.Errorf("ready: got %v, want [b, c]", ready)
	}
}

func TestDAG_FindReady_PartialDeps(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a", TaskType: "task_a", Status: protocol.StepCompleted},
		{ID: "b", TaskType: "task_b", Status: protocol.StepRunning},
		{ID: "c", TaskType: "task_c", Status: protocol.StepPending, DependsOn: []string{"a", "b"}},
	}

	ready := orchestrator.FindReadySteps(steps)
	if len(ready) != 0 {
		t.Errorf("ready: got %v, want [] (b not done)", ready)
	}
}

func TestDAG_FindReady_SkippedDep(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a", TaskType: "task_a", Status: protocol.StepSkipped},
		{ID: "b", TaskType: "task_b", Status: protocol.StepPending, DependsOn: []string{"a"}},
	}

	// Skipped counts as resolved
	ready := orchestrator.FindReadySteps(steps)
	if len(ready) != 1 || ready[0] != "b" {
		t.Errorf("ready: got %v, want [b]", ready)
	}
}

func TestDAG_IsComplete(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a", Status: protocol.StepCompleted},
		{ID: "b", Status: protocol.StepCompleted},
	}
	if !orchestrator.IsWorkflowDone(steps) {
		t.Error("should be done")
	}
}

func TestDAG_IsComplete_WithSkipped(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a", Status: protocol.StepCompleted},
		{ID: "b", Status: protocol.StepSkipped},
	}
	if !orchestrator.IsWorkflowDone(steps) {
		t.Error("should be done (skipped counts)")
	}
}

func TestDAG_IsComplete_NotYet(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a", Status: protocol.StepCompleted},
		{ID: "b", Status: protocol.StepRunning},
	}
	if orchestrator.IsWorkflowDone(steps) {
		t.Error("should not be done")
	}
}

func TestDAG_ValidateNoCycle(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a", DependsOn: []string{"b"}},
		{ID: "b", DependsOn: []string{"a"}},
	}
	if err := orchestrator.ValidateDAG(steps); err == nil {
		t.Error("should detect cycle")
	}
}

func TestDAG_ValidateValid(t *testing.T) {
	steps := []protocol.WorkflowStep{
		{ID: "a"},
		{ID: "b", DependsOn: []string{"a"}},
		{ID: "c", DependsOn: []string{"a"}},
		{ID: "d", DependsOn: []string{"b", "c"}},
	}
	if err := orchestrator.ValidateDAG(steps); err != nil {
		t.Errorf("valid DAG should pass: %v", err)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/orchestrator/ -v -run TestDAG`
Expected: FAIL

- [ ] **Step 3: Implement DAG engine**

Create `core/internal/orchestrator/dag.go`:
```go
package orchestrator

import (
	"fmt"

	"github.com/kienbm/magic-claw/core/internal/protocol"
)

// FindReadySteps returns step IDs that are pending and have all dependencies resolved.
func FindReadySteps(steps []protocol.WorkflowStep) []string {
	resolved := make(map[string]bool)
	for _, s := range steps {
		if s.Status == protocol.StepCompleted || s.Status == protocol.StepSkipped {
			resolved[s.ID] = true
		}
	}

	var ready []string
	for _, s := range steps {
		if s.Status != protocol.StepPending {
			continue
		}
		allDepsOK := true
		for _, dep := range s.DependsOn {
			if !resolved[dep] {
				allDepsOK = false
				break
			}
		}
		if allDepsOK {
			ready = append(ready, s.ID)
		}
	}
	return ready
}

// IsWorkflowDone returns true if all steps are completed, skipped, or failed.
func IsWorkflowDone(steps []protocol.WorkflowStep) bool {
	for _, s := range steps {
		switch s.Status {
		case protocol.StepCompleted, protocol.StepSkipped, protocol.StepFailed:
			continue
		default:
			return false
		}
	}
	return true
}

// HasFailed returns true if any step has failed (excluding skipped).
func HasFailed(steps []protocol.WorkflowStep) bool {
	for _, s := range steps {
		if s.Status == protocol.StepFailed {
			return true
		}
	}
	return false
}

// ValidateDAG checks for cycles using topological sort (Kahn's algorithm).
func ValidateDAG(steps []protocol.WorkflowStep) error {
	// Build adjacency and in-degree
	ids := make(map[string]bool)
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dep -> [steps that depend on it]

	for _, s := range steps {
		ids[s.ID] = true
		inDegree[s.ID] = len(s.DependsOn)
		for _, dep := range s.DependsOn {
			if !ids[dep] {
				ids[dep] = true
			}
			dependents[dep] = append(dependents[dep], s.ID)
		}
	}

	// Find nodes with no incoming edges
	var queue []string
	for _, s := range steps {
		if inDegree[s.ID] == 0 {
			queue = append(queue, s.ID)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++

		for _, dep := range dependents[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if visited != len(steps) {
		return fmt.Errorf("workflow contains a cycle")
	}
	return nil
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/orchestrator/ -v -run TestDAG`
Expected: PASS (9 tests)

- [ ] **Step 5: Commit**

```bash
git add core/internal/orchestrator/
git commit -m "feat(orchestrator): add DAG engine with cycle detection and step resolution"
```

---

### Task 7: Orchestrator — workflow execution

**Files:**
- Create: `core/internal/orchestrator/orchestrator.go`
- Create: `core/internal/orchestrator/orchestrator_test.go`

- [ ] **Step 1: Write test for orchestrator**

Create `core/internal/orchestrator/orchestrator_test.go`:
```go
package orchestrator_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/orchestrator"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/registry"
	"github.com/kienbm/magic-claw/core/internal/router"
	"github.com/kienbm/magic-claw/core/internal/store"
)

func setupOrchestrator(t *testing.T) (*orchestrator.Orchestrator, store.Store) {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)
	rt := router.New(reg, s, bus)

	// Register a worker with multiple capabilities
	reg.Register(protocol.RegisterPayload{
		Name: "MultiBot",
		Capabilities: []protocol.Capability{
			{Name: "market_research"},
			{Name: "content_writing"},
			{Name: "seo_optimization"},
		},
		Endpoint: protocol.Endpoint{Type: "http", URL: "http://localhost:9000"},
		Limits:   protocol.WorkerLimits{MaxConcurrentTasks: 10},
	})

	orch := orchestrator.New(s, rt, bus)
	return orch, s
}

func TestOrchestrator_SubmitWorkflow(t *testing.T) {
	orch, s := setupOrchestrator(t)

	wf, err := orch.Submit("Campaign", []protocol.WorkflowStep{
		{ID: "research", TaskType: "market_research", Input: json.RawMessage(`{"topic": "AI"}`)},
		{ID: "content", TaskType: "content_writing", DependsOn: []string{"research"}},
	}, protocol.TaskContext{OrgID: "org_magic"})

	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if wf.Status != protocol.WorkflowRunning {
		t.Errorf("Status: got %q, want running", wf.Status)
	}

	// Should have stored workflow
	got, err := s.GetWorkflow(wf.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.Name != "Campaign" {
		t.Errorf("Name: got %q", got.Name)
	}
}

func TestOrchestrator_InvalidDAG(t *testing.T) {
	orch, _ := setupOrchestrator(t)

	_, err := orch.Submit("Bad", []protocol.WorkflowStep{
		{ID: "a", TaskType: "t", DependsOn: []string{"b"}},
		{ID: "b", TaskType: "t", DependsOn: []string{"a"}},
	}, protocol.TaskContext{})

	if err == nil {
		t.Error("should reject cyclic workflow")
	}
}

func TestOrchestrator_CompleteStep(t *testing.T) {
	orch, s := setupOrchestrator(t)

	wf, _ := orch.Submit("Campaign", []protocol.WorkflowStep{
		{ID: "research", TaskType: "market_research", Input: json.RawMessage(`{}`)},
		{ID: "content", TaskType: "content_writing", DependsOn: []string{"research"}, Input: json.RawMessage(`{}`)},
	}, protocol.TaskContext{})

	// Find the task for "research" step
	got, _ := s.GetWorkflow(wf.ID)
	researchTaskID := got.Steps[0].TaskID

	// Complete the research step
	err := orch.CompleteStep(wf.ID, researchTaskID, json.RawMessage(`{"data": "results"}`))
	if err != nil {
		t.Fatalf("CompleteStep: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // allow async step dispatch

	// Verify research is completed and content is now running
	got, _ = s.GetWorkflow(wf.ID)
	if got.Steps[0].Status != protocol.StepCompleted {
		t.Errorf("research status: got %q", got.Steps[0].Status)
	}
	if got.Steps[1].Status != protocol.StepRunning {
		t.Errorf("content status: got %q, want running", got.Steps[1].Status)
	}
}

func TestOrchestrator_WorkflowCompletion(t *testing.T) {
	orch, s := setupOrchestrator(t)

	wf, _ := orch.Submit("Simple", []protocol.WorkflowStep{
		{ID: "only", TaskType: "market_research", Input: json.RawMessage(`{}`)},
	}, protocol.TaskContext{})

	got, _ := s.GetWorkflow(wf.ID)
	taskID := got.Steps[0].TaskID

	orch.CompleteStep(wf.ID, taskID, json.RawMessage(`{"done": true}`))

	time.Sleep(50 * time.Millisecond)

	got, _ = s.GetWorkflow(wf.ID)
	if got.Status != protocol.WorkflowCompleted {
		t.Errorf("workflow status: got %q, want completed", got.Status)
	}
}

func TestOrchestrator_FailStepSkip(t *testing.T) {
	orch, s := setupOrchestrator(t)

	wf, _ := orch.Submit("WithSkip", []protocol.WorkflowStep{
		{ID: "a", TaskType: "market_research", Input: json.RawMessage(`{}`)},
		{ID: "b", TaskType: "content_writing", DependsOn: []string{"a"}, OnFailure: "skip", Input: json.RawMessage(`{}`)},
	}, protocol.TaskContext{})

	got, _ := s.GetWorkflow(wf.ID)
	taskIDA := got.Steps[0].TaskID

	// Complete step A
	orch.CompleteStep(wf.ID, taskIDA, json.RawMessage(`{}`))
	time.Sleep(100 * time.Millisecond)

	// Now fail step B
	got, _ = s.GetWorkflow(wf.ID)
	taskIDB := got.Steps[1].TaskID
	orch.FailStep(wf.ID, taskIDB, protocol.TaskError{Code: "err", Message: "failed"})

	time.Sleep(50 * time.Millisecond)

	// B should be skipped (on_failure: skip), workflow should complete
	got, _ = s.GetWorkflow(wf.ID)
	if got.Steps[1].Status != protocol.StepSkipped {
		t.Errorf("step B status: got %q, want skipped", got.Steps[1].Status)
	}
	if got.Status != protocol.WorkflowCompleted {
		t.Errorf("workflow status: got %q, want completed", got.Status)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/orchestrator/ -v -run TestOrchestrator`
Expected: FAIL

- [ ] **Step 3: Implement orchestrator**

Create `core/internal/orchestrator/orchestrator.go`:
```go
package orchestrator

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/protocol"
	"github.com/kienbm/magic-claw/core/internal/router"
	"github.com/kienbm/magic-claw/core/internal/store"
)

type Orchestrator struct {
	store  store.Store
	router *router.Router
	bus    *events.Bus
}

func New(s store.Store, rt *router.Router, bus *events.Bus) *Orchestrator {
	return &Orchestrator{store: s, router: rt, bus: bus}
}

func (o *Orchestrator) Submit(name string, steps []protocol.WorkflowStep, ctx protocol.TaskContext) (*protocol.Workflow, error) {
	if err := ValidateDAG(steps); err != nil {
		return nil, fmt.Errorf("invalid workflow: %w", err)
	}

	// Initialize step statuses
	for i := range steps {
		steps[i].Status = protocol.StepPending
	}

	wf := &protocol.Workflow{
		ID:        protocol.GenerateID("wf"),
		Name:      name,
		Steps:     steps,
		Status:    protocol.WorkflowRunning,
		Context:   ctx,
		CreatedAt: time.Now(),
	}

	if err := o.store.AddWorkflow(wf); err != nil {
		return nil, err
	}

	o.bus.Publish(events.Event{
		Type:   "workflow.started",
		Source: "orchestrator",
		Payload: map[string]any{
			"workflow_id": wf.ID,
			"name":        wf.Name,
			"steps":       len(wf.Steps),
		},
	})

	// Dispatch initial ready steps
	o.advanceWorkflow(wf)

	return wf, nil
}

func (o *Orchestrator) CompleteStep(workflowID, taskID string, output json.RawMessage) error {
	wf, err := o.store.GetWorkflow(workflowID)
	if err != nil {
		return err
	}

	for i := range wf.Steps {
		if wf.Steps[i].TaskID == taskID {
			wf.Steps[i].Status = protocol.StepCompleted
			wf.Steps[i].Output = output
			break
		}
	}

	if err := o.store.UpdateWorkflow(wf); err != nil {
		return err
	}

	o.bus.Publish(events.Event{
		Type:   "workflow.step_completed",
		Source: "orchestrator",
		Payload: map[string]any{
			"workflow_id": workflowID,
			"task_id":     taskID,
		},
	})

	go o.advanceWorkflow(wf)
	return nil
}

func (o *Orchestrator) FailStep(workflowID, taskID string, taskErr protocol.TaskError) error {
	wf, err := o.store.GetWorkflow(workflowID)
	if err != nil {
		return err
	}

	for i := range wf.Steps {
		if wf.Steps[i].TaskID == taskID {
			step := &wf.Steps[i]
			step.Error = &taskErr

			switch step.OnFailure {
			case "skip":
				step.Status = protocol.StepSkipped
			case "abort":
				step.Status = protocol.StepFailed
				wf.Status = protocol.WorkflowAborted
				o.store.UpdateWorkflow(wf)
				o.bus.Publish(events.Event{
					Type:     "workflow.aborted",
					Source:   "orchestrator",
					Severity: "error",
					Payload:  map[string]any{"workflow_id": workflowID, "failed_step": step.ID},
				})
				return nil
			default: // retry or no policy — mark failed
				step.Status = protocol.StepFailed
			}
			break
		}
	}

	if err := o.store.UpdateWorkflow(wf); err != nil {
		return err
	}

	go o.advanceWorkflow(wf)
	return nil
}

func (o *Orchestrator) advanceWorkflow(wf *protocol.Workflow) {
	// Check if done
	if IsWorkflowDone(wf.Steps) {
		if HasFailed(wf.Steps) {
			wf.Status = protocol.WorkflowFailed
		} else {
			wf.Status = protocol.WorkflowCompleted
		}
		now := time.Now()
		wf.DoneAt = &now
		o.store.UpdateWorkflow(wf)

		o.bus.Publish(events.Event{
			Type:   "workflow.completed",
			Source: "orchestrator",
			Payload: map[string]any{
				"workflow_id": wf.ID,
				"status":      wf.Status,
			},
		})
		return
	}

	// Find and dispatch ready steps
	ready := FindReadySteps(wf.Steps)
	for _, stepID := range ready {
		for i := range wf.Steps {
			if wf.Steps[i].ID == stepID {
				o.dispatchStep(wf, &wf.Steps[i])
				break
			}
		}
	}

	o.store.UpdateWorkflow(wf)
}

func (o *Orchestrator) dispatchStep(wf *protocol.Workflow, step *protocol.WorkflowStep) {
	task := &protocol.Task{
		ID:       protocol.GenerateID("task"),
		Type:     step.TaskType,
		Priority: protocol.PriorityNormal,
		Status:   protocol.TaskPending,
		Input:    step.Input,
		Routing: protocol.RoutingConfig{
			Strategy:             "best_match",
			RequiredCapabilities: []string{step.TaskType},
		},
		WorkflowID: wf.ID,
		Context:    wf.Context,
		CreatedAt:  time.Now(),
	}

	worker, err := o.router.RouteTask(task)
	if err != nil {
		step.Status = protocol.StepFailed
		step.Error = &protocol.TaskError{Code: "no_worker", Message: err.Error()}
		return
	}

	_ = worker
	o.store.AddTask(task)
	step.Status = protocol.StepRunning
	step.TaskID = task.ID
}

func (o *Orchestrator) GetWorkflow(id string) (*protocol.Workflow, error) {
	return o.store.GetWorkflow(id)
}

func (o *Orchestrator) ListWorkflows() []*protocol.Workflow {
	return o.store.ListWorkflows()
}
```

- [ ] **Step 4: Run test — verify it passes**

Run: `cd core && go test ./internal/orchestrator/ -v`
Expected: PASS (14 tests total: 9 DAG + 5 Orchestrator)

- [ ] **Step 5: Commit**

```bash
git add core/internal/orchestrator/
git commit -m "feat(orchestrator): add workflow execution with DAG scheduling and failure handling"
```

---

## Chunk 6: Integration (Gateway + main.go)

### Task 8: Add new API endpoints to Gateway

**Files:**
- Modify: `core/internal/gateway/gateway.go`
- Modify: `core/internal/gateway/handlers.go`
- Modify: `core/internal/gateway/gateway_test.go`
- Modify: `core/cmd/magic/main.go`

- [ ] **Step 1: Write tests for new endpoints**

Append to `core/internal/gateway/gateway_test.go`:
```go
func TestGateway_SubmitWorkflow(t *testing.T) {
	gw := setupGateway()
	srv := httptest.NewServer(gw.Handler())
	defer srv.Close()

	// Register a worker
	regPayload := protocol.RegisterPayload{
		Name:         "MultiBot",
		Capabilities: []protocol.Capability{{Name: "market_research"}, {Name: "content_writing"}},
		Endpoint:     protocol.Endpoint{Type: "http", URL: "http://localhost:9000"},
		Limits:       protocol.WorkerLimits{MaxConcurrentTasks: 10},
	}
	body, _ := json.Marshal(regPayload)
	http.Post(srv.URL+"/api/v1/workers/register", "application/json", bytes.NewReader(body))

	wfReq := map[string]any{
		"name": "Test Workflow",
		"steps": []map[string]any{
			{"id": "step1", "task_type": "market_research", "input": map[string]string{"topic": "AI"}},
			{"id": "step2", "task_type": "content_writing", "depends_on": []string{"step1"}, "input": map[string]string{}},
		},
	}
	body, _ = json.Marshal(wfReq)

	resp, err := http.Post(srv.URL+"/api/v1/workflows", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("status: got %d, want 201", resp.StatusCode)
	}

	var wf protocol.Workflow
	json.NewDecoder(resp.Body).Decode(&wf)
	if wf.Status != protocol.WorkflowRunning {
		t.Errorf("status: got %q, want running", wf.Status)
	}
}

func TestGateway_CreateTeam(t *testing.T) {
	gw := setupGateway()
	srv := httptest.NewServer(gw.Handler())
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"name":         "Marketing",
		"org_id":       "org_magic",
		"daily_budget": 10.0,
	})
	resp, err := http.Post(srv.URL+"/api/v1/teams", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("status: got %d, want 201", resp.StatusCode)
	}

	var team protocol.Team
	json.NewDecoder(resp.Body).Decode(&team)
	if team.Name != "Marketing" {
		t.Errorf("Name: got %q", team.Name)
	}
}

func TestGateway_ListTeams(t *testing.T) {
	gw := setupGateway()
	srv := httptest.NewServer(gw.Handler())
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{"name": "Sales", "org_id": "org", "daily_budget": 5.0})
	http.Post(srv.URL+"/api/v1/teams", "application/json", bytes.NewReader(body))

	resp, _ := http.Get(srv.URL + "/api/v1/teams")
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d", resp.StatusCode)
	}

	var teams []*protocol.Team
	json.NewDecoder(resp.Body).Decode(&teams)
	if len(teams) != 1 {
		t.Errorf("teams: got %d, want 1", len(teams))
	}
}

func TestGateway_CostReport(t *testing.T) {
	gw := setupGateway()
	srv := httptest.NewServer(gw.Handler())
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/v1/costs")
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d", resp.StatusCode)
	}

	var report map[string]any
	json.NewDecoder(resp.Body).Decode(&report)
	if _, ok := report["total_cost"]; !ok {
		t.Error("should have total_cost field")
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `cd core && go test ./internal/gateway/ -v -run TestGateway_SubmitWorkflow`
Expected: FAIL

- [ ] **Step 3: Update Gateway struct and setupGateway in test**

Update `setupGateway()` in `core/internal/gateway/gateway_test.go` to import and initialize new modules:
```go
import (
	// ... existing imports ...
	"github.com/kienbm/magic-claw/core/internal/costctrl"
	"github.com/kienbm/magic-claw/core/internal/evaluator"
	"github.com/kienbm/magic-claw/core/internal/orchestrator"
	"github.com/kienbm/magic-claw/core/internal/orgmgr"
)

func setupGateway() *gateway.Gateway {
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)
	rt := router.New(reg, s, bus)
	mon := monitor.New(bus, os.Stderr)
	mon.Start()
	cc := costctrl.New(s, bus)
	ev := evaluator.New(bus)
	orch := orchestrator.New(s, rt, bus)
	mgr := orgmgr.New(s, bus)
	return gateway.New(reg, rt, s, bus, mon, cc, ev, orch, mgr)
}
```

- [ ] **Step 4: Update Gateway struct**

Update `core/internal/gateway/gateway.go`:
```go
package gateway

import (
	"net/http"

	"github.com/kienbm/magic-claw/core/internal/costctrl"
	"github.com/kienbm/magic-claw/core/internal/evaluator"
	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/monitor"
	"github.com/kienbm/magic-claw/core/internal/orchestrator"
	"github.com/kienbm/magic-claw/core/internal/orgmgr"
	"github.com/kienbm/magic-claw/core/internal/registry"
	"github.com/kienbm/magic-claw/core/internal/router"
	"github.com/kienbm/magic-claw/core/internal/store"
)

type Gateway struct {
	registry     *registry.Registry
	router       *router.Router
	store        store.Store
	bus          *events.Bus
	monitor      *monitor.Monitor
	costCtrl     *costctrl.Controller
	evaluator    *evaluator.Evaluator
	orchestrator *orchestrator.Orchestrator
	orgMgr       *orgmgr.Manager
}

func New(reg *registry.Registry, rt *router.Router, s store.Store, bus *events.Bus, mon *monitor.Monitor, cc *costctrl.Controller, ev *evaluator.Evaluator, orch *orchestrator.Orchestrator, mgr *orgmgr.Manager) *Gateway {
	return &Gateway{
		registry:     reg,
		router:       rt,
		store:        s,
		bus:          bus,
		monitor:      mon,
		costCtrl:     cc,
		evaluator:    ev,
		orchestrator: orch,
		orgMgr:       mgr,
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

	// Workflows
	mux.HandleFunc("POST /api/v1/workflows", g.handleSubmitWorkflow)
	mux.HandleFunc("GET /api/v1/workflows", g.handleListWorkflows)

	// Teams
	mux.HandleFunc("POST /api/v1/teams", g.handleCreateTeam)
	mux.HandleFunc("GET /api/v1/teams", g.handleListTeams)

	// Costs
	mux.HandleFunc("GET /api/v1/costs", g.handleCostReport)

	// Metrics
	mux.HandleFunc("GET /api/v1/metrics", g.handleGetStats)

	var handler http.Handler = mux
	handler = requestIDMiddleware(handler)
	handler = corsMiddleware(handler)

	return handler
}
```

- [ ] **Step 5: Add new handlers**

Append to `core/internal/gateway/handlers.go`:
```go
type WorkflowRequest struct {
	Name    string                  `json:"name"`
	Steps   []protocol.WorkflowStep `json:"steps"`
	Context protocol.TaskContext    `json:"context"`
}

func (g *Gateway) handleSubmitWorkflow(w http.ResponseWriter, r *http.Request) {
	var req WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	wf, err := g.orchestrator.Submit(req.Name, req.Steps, req.Context)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wf)
}

func (g *Gateway) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows := g.orchestrator.ListWorkflows()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

type CreateTeamRequest struct {
	Name        string  `json:"name"`
	OrgID       string  `json:"org_id"`
	DailyBudget float64 `json:"daily_budget"`
}

func (g *Gateway) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	team, err := g.orgMgr.CreateTeam(req.Name, req.OrgID, req.DailyBudget)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(team)
}

func (g *Gateway) handleListTeams(w http.ResponseWriter, r *http.Request) {
	teams := g.orgMgr.ListTeams()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

func (g *Gateway) handleCostReport(w http.ResponseWriter, r *http.Request) {
	report := g.costCtrl.OrgReport()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
```

- [ ] **Step 6: Update main.go**

Update `core/cmd/magic/main.go`:
```go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kienbm/magic-claw/core/internal/costctrl"
	"github.com/kienbm/magic-claw/core/internal/evaluator"
	"github.com/kienbm/magic-claw/core/internal/events"
	"github.com/kienbm/magic-claw/core/internal/gateway"
	"github.com/kienbm/magic-claw/core/internal/monitor"
	"github.com/kienbm/magic-claw/core/internal/orchestrator"
	"github.com/kienbm/magic-claw/core/internal/orgmgr"
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

	// Core
	s := store.NewMemoryStore()
	bus := events.NewBus()
	reg := registry.New(s, bus)
	rt := router.New(reg, s, bus)
	mon := monitor.New(bus, os.Stdout)
	mon.Start()
	reg.StartHealthCheck(30_000_000_000) // 30s

	// Tier 2
	cc := costctrl.New(s, bus)
	ev := evaluator.New(bus)
	orch := orchestrator.New(s, rt, bus)
	mgr := orgmgr.New(s, bus)

	gw := gateway.New(reg, rt, s, bus, mon, cc, ev, orch, mgr)

	fmt.Printf("MagiC server starting on :%s\n", port)
	fmt.Println("  POST /api/v1/workers/register  — Register a worker")
	fmt.Println("  GET  /api/v1/workers           — List workers")
	fmt.Println("  POST /api/v1/tasks             — Submit a task")
	fmt.Println("  POST /api/v1/workflows         — Submit a workflow")
	fmt.Println("  GET  /api/v1/workflows         — List workflows")
	fmt.Println("  POST /api/v1/teams             — Create a team")
	fmt.Println("  GET  /api/v1/teams             — List teams")
	fmt.Println("  GET  /api/v1/costs             — Cost report")
	fmt.Println("  GET  /api/v1/metrics           — View stats")
	fmt.Println("  GET  /health                   — Health check")

	if err := http.ListenAndServe(":"+port, gw.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Run all tests**

Run: `cd core && go test ./... -v -count=1`
Expected: ALL PASS across all packages

- [ ] **Step 8: Build and verify**

Run: `make build && ./bin/magic`
Expected: prints usage with all endpoints

- [ ] **Step 9: Commit**

```bash
git add core/internal/gateway/ core/cmd/magic/main.go
git commit -m "feat(gateway): add workflow, team, cost endpoints and wire up Tier 2 modules"
```

---

### Task 9: Final integration test

- [ ] **Step 1: Run full test suite**

Run: `cd core && go test ./... -v -count=1`
Expected: ALL PASS (30+ tests across 9 packages)

- [ ] **Step 2: Run Python SDK tests**

Run: `cd sdk/python && pytest tests/ -v`
Expected: PASS (3 tests)

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat: MagiC v0.2.0 — Tier 2 modules (Orchestrator, Evaluator, Cost Controller, Org Manager)"
```

---

## Summary

After completing this plan, the MagiC server has all 9 modules operational:

**Tier 1 (Plan 1):** Gateway, Registry, Router, Monitor
**Tier 2 (Plan 2):** Orchestrator, Evaluator, Cost Controller, Org Manager

New capabilities:
- **Orchestrator:** Submit multi-step workflows (DAGs), parallel step execution, failure handling (skip/abort/retry)
- **Evaluator:** JSON schema validation for task output
- **Cost Controller:** Per-worker cost tracking, budget alerts at 80%, auto-pause at 100%
- **Org Manager:** Team CRUD, worker-to-team assignment

New API endpoints:
- `POST /api/v1/workflows` — Submit a workflow
- `GET  /api/v1/workflows` — List workflows
- `POST /api/v1/teams` — Create a team
- `GET  /api/v1/teams` — List teams
- `GET  /api/v1/costs` — Organization cost report

**Next:** Plan 3 will add the Knowledge Hub (Tier 3) and enhance existing modules.
