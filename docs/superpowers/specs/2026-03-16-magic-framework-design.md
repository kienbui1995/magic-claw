# MagiC Framework — Design Specification

> **"Don't build another AI. Manage the ones you have."**

**Version:** 1.0.0
**Date:** 2026-03-16
**Author:** Kien (kienbm) + Claude
**Status:** Draft

---

## Table of Contents

1. [Vision & Problem Statement](#1-vision--problem-statement)
2. [Product Definition](#2-product-definition)
3. [MagiC Protocol (MCP²)](#3-magic-protocol-mcp)
4. [Core Entities](#4-core-entities)
5. [Module Architecture](#5-module-architecture)
6. [Tech Stack & Project Structure](#6-tech-stack--project-structure)
7. [Module Specifications](#7-module-specifications)
8. [Data Flow & Sequences](#8-data-flow--sequences)
9. [First Use Case: CEO Division Orchestration](#9-first-use-case-ceo-division-orchestration)
10. [Open-Source & Viral Strategy](#10-open-source--viral-strategy)
11. [Risks & Mitigations](#11-risks--mitigations)

---

## 1. Vision & Problem Statement

### The Problem

The AI agent market is projected to grow from $7.84B (2025) to $52B (2030) at 45.8% CAGR. Every enterprise is deploying AI agents. But no one manages them well.

**5 pain points** enterprises face today (Azumo + Gartner 2026):

| Pain Point | Impact |
|---|---|
| Escalating costs | Multiple agents, multiple models, multiple API keys — no visibility into total cost |
| Unclear business value | No way to measure which agent is effective and which is useless |
| Lack of governance | No control over who can do what, which data agents can access |
| Weak risk controls | Agents fail silently, no rollback, no audit trail |
| AI skills gap | Teams lack expertise to build AND operate multiple agents |

### The Insight

Every competitor (CrewAI, AutoGen, MetaGPT, Agno) focuses on **building agents**. Nobody focuses on **managing a fleet of any agents**.

This is the **"Kubernetes moment"** for AI:
- Docker (build containers) came first → Kubernetes (manage containers) came after and won bigger
- LangChain/CrewAI (build agents) came first → **MagiC (manage any agents)** is the next layer

### The Vision

**MagiC** is a framework where **any AI worker** (OpenClaw, CrewAI agent, custom bot, GPT assistant...) can **join an organization** and work according to user requirements — through an open protocol.

---

## 2. Product Definition

### Name & Branding

- **Name:** MagiC (capital C)
- **C =** Company · Crew · Claw
- **Tagline (branding):** "Where AI becomes a Company"
- **Tagline (landing):** "Run your AI workforce like a real team"
- **Tagline (GitHub):** "Don't build another AI. Manage the ones you have."

### What MagiC Is

- An **open-source framework** (Go core + Python SDK)
- With an **open protocol** (MagiC Protocol) for AI worker integration
- That provides **fleet management** capabilities (registry, routing, monitoring, evaluation, cost control, orchestration)

### What MagiC Is NOT

- NOT an AI agent builder (use LangChain, CrewAI, Dify for that)
- NOT a chatbot platform (use Botpress, Voiceflow for that)
- NOT a workflow automation tool (use n8n for that)
- MagiC **manages** agents built with any of those tools

### Target Audience

| Phase | Audience | How they use MagiC |
|---|---|---|
| Phase 1-2 | Developers / Tech teams | `go get` or `pip install`, build workers, manage via CLI/API |
| Phase 3 | B2B SMB | SaaS dashboard, no-code worker integration |

### Business Model

- **Open-source core:** Apache 2.0 license. Framework, protocol spec, CLI, SDK — all free
- **SaaS platform (later):** Managed hosting, dashboard UI, team collaboration, enterprise SSO
- **Marketplace (later):** Community shares/sells worker templates, plugins, integrations

### Why Apache 2.0 (not MIT)?

| | MIT | Apache 2.0 |
|---|---|---|
| Freedom | Maximum | Nearly maximum |
| Patent protection | **NONE** | **YES** — protects against patent trolls |
| Requirements | Keep copyright notice | Keep copyright + note changes |
| Used by | React, Next.js, LangChain | Kubernetes, Docker, Dify, TensorFlow |

Apache 2.0 chosen because:
1. **Patent shield** — if someone uses MagiC then sues for patent infringement, their license auto-terminates
2. **Infra standard** — K8s, Docker, Dify, TensorFlow all use Apache 2.0. MagiC is infra, should follow convention
3. **Enterprise-friendly** — corporate legal teams approve Apache 2.0 faster due to explicit patent clause

---

## 3. MagiC Protocol (MCP²)

The protocol defines how **Organizations** communicate with **AI Workers**. Any AI system that implements MCP² can join a MagiC-managed organization.

### How MCP² differs from MCP (Anthropic)

| | MCP (Anthropic) | MagiC Protocol (MCP²) |
|---|---|---|
| **What it connects** | 1 AI model ↔ 1 tool | 1 Organization ↔ N workers |
| **Relationship** | Client-Server (1:1) | Org-Fleet (1:N) |
| **Lifecycle** | None. Call once, done | Full: register → heartbeat → assign → complete → deregister |
| **Routing** | None. Hardcoded tool | Smart: best_match, round_robin, cheapest, fastest |
| **Cost tracking** | None | Per task, per worker, budget alerts |
| **Quality eval** | None | Schema validation, LLM-as-judge |
| **Collaboration** | None | worker.delegate, org.broadcast |
| **Fail-safe** | None | retry → delegate → notify human |
| **Health monitoring** | None | Heartbeat, offline detection |

**In short:** MCP = "AI calls 1 tool" (single, unmanaged). MCP² = "Organization manages a fleet of AI workers" (lifecycle, routing, monitoring, cost, quality). Same difference as calling 1 API vs using API Gateway + Load Balancer + Monitoring.

### 3.1 Design Principles

| Principle | Description |
|---|---|
| **Transport-agnostic** | HTTP, WebSocket, gRPC — workers choose their transport. Protocol defines message format only |
| **Capability-based** | Workers self-declare capabilities. Organization routes tasks based on capability match |
| **Contract-driven** | Every task has a contract: input schema, output schema, quality criteria, timeout, max cost |
| **Human-in-the-loop** | Humans are always "boss". Can approve, reject, redirect any task at any point |
| **Fail-safe** | Worker failure triggers fallback: retry → delegate to another worker → notify human. No silent failures |

### 3.2 Message Format

All messages use JSON. Transport layer wraps them (HTTP body, WebSocket frame, gRPC message).

```json
{
  "protocol": "mcp2",
  "version": "1.0",
  "type": "task.assign",
  "id": "msg_abc123",
  "timestamp": "2026-03-16T10:00:00Z",
  "source": "org_magic",
  "target": "worker_openclaw_001",
  "payload": { ... }
}
```

### 3.3 Communication Patterns

**Pattern 1: Worker ↔ Organization (default — all traffic goes through Org)**

```
Worker A → Org → Router → Worker B    (delegation, task routing)
Worker A → Org → Monitor              (logging, metrics)
Worker A → Org → CostCtrl             (cost tracking)
```

**Pattern 2: Worker ↔ Worker (direct channel — opt-in for real-time)**

```
Worker A ←→ Worker B                  (streaming, real-time collab)
Org monitors but doesn't route        (via worker.open_channel)
```

Use cases: Worker A streams text → Worker B translates real-time. Two workers co-edit a document.

**Pattern 3: Worker ↔ Environment (via MCP or native)**

```
Worker → MCP Server → Database/API/Files   (structured tool access)
Worker → REST API directly                  (native integration)
Org can gate access via env.access_request  (policy enforcement)
```

MCP (Anthropic) and MCP² (MagiC) are **complementary**: MCP connects workers to tools, MCP² connects organization to workers.

### 3.4 Message Types (14 total)

**Worker Lifecycle (4):**

| Message | Direction | Description |
|---|---|---|
| `worker.register` | Worker → Org | Worker joins organization. Declares name, capabilities, endpoint, auth, limits |
| `worker.heartbeat` | Worker → Org | Periodic health check. Includes current load, availability |
| `worker.deregister` | Worker → Org | Worker leaves organization gracefully |
| `worker.update_capabilities` | Worker → Org | Worker updates its capabilities without re-registering |

**Task Lifecycle (5):**

| Message | Direction | Description |
|---|---|---|
| `task.assign` | Org → Worker | Organization assigns a task to a worker. Includes full contract |
| `task.accept` | Worker → Org | Worker accepts the task (optional — auto-accept by default) |
| `task.reject` | Worker → Org | Worker rejects the task with reason. Org re-routes to another worker |
| `task.progress` | Worker → Org | Worker reports progress (0-100%) with optional intermediate results |
| `task.complete` | Worker → Org | Worker delivers final result. Includes output, metadata, cost |
| `task.fail` | Worker → Org | Worker reports failure with error details. Triggers fail-safe |

**Collaboration (2):**

| Message | Direction | Description |
|---|---|---|
| `worker.delegate` | Worker → Org | Worker requests another worker's help via the organization |
| `org.broadcast` | Org → All Workers | Organization sends notification to all workers (policy change, maintenance) |

**Direct Channel (2):**

| Message | Direction | Description |
|---|---|---|
| `worker.open_channel` | Worker → Org | Request direct P2P connection with another worker |
| `worker.close_channel` | Worker → Org | Close direct connection |

**Environment Access (2):**

| Message | Direction | Description |
|---|---|---|
| `env.access_request` | Worker → Org | Worker requests access to external resource (if policy requires approval) |
| `env.access_granted` | Org → Worker | Organization grants access |

### 3.5 Worker Registration

```json
{
  "type": "worker.register",
  "payload": {
    "name": "OpenClaw Instance #1",
    "capabilities": [
      {
        "name": "content_writing",
        "description": "Write blog posts, articles, social media content",
        "input_schema": { "type": "object", "properties": { "topic": { "type": "string" }, "tone": { "type": "string" } } },
        "output_schema": { "type": "object", "properties": { "title": { "type": "string" }, "body": { "type": "string" } } },
        "estimated_cost_per_call": 0.05,
        "avg_response_time_ms": 15000
      }
    ],
    "endpoint": {
      "type": "http",
      "url": "http://openclaw-1:8000/mcp2",
      "auth": { "type": "api_key", "header": "X-Worker-Key" }
    },
    "limits": {
      "max_concurrent_tasks": 5,
      "rate_limit": "100/min",
      "max_cost_per_day": 10.00
    },
    "metadata": {
      "version": "1.2.0",
      "runtime": "python",
      "model": "claude-sonnet-4-20250514"
    }
  }
}
```

### 3.5 Task Contract

```json
{
  "type": "task.assign",
  "payload": {
    "task_id": "task_001",
    "task_type": "content_writing",
    "priority": "normal",
    "input": {
      "topic": "AI agent trends in 2026",
      "tone": "professional",
      "word_count": 1500
    },
    "contract": {
      "output_schema": {
        "type": "object",
        "required": ["title", "body"],
        "properties": {
          "title": { "type": "string", "maxLength": 100 },
          "body": { "type": "string", "minLength": 1000 }
        }
      },
      "quality_criteria": [
        { "metric": "no_hallucination", "threshold": true },
        { "metric": "seo_score", "threshold": 80 }
      ],
      "timeout_ms": 300000,
      "max_cost": 0.50,
      "retry_policy": { "max_retries": 2, "backoff_ms": 5000 }
    },
    "routing": {
      "strategy": "best_match",
      "required_capabilities": ["content_writing"],
      "preferred_workers": [],
      "excluded_workers": []
    },
    "context": {
      "org_id": "org_magic",
      "team_id": "team_content",
      "requester": "user_kien",
      "workflow_id": null
    }
  }
}
```

---

## 4. Core Entities

### 4.1 Entity Relationship

```
Organization (1)
  ├── Teams (N)
  │     ├── Workers (N)
  │     │     └── Tools (N)        ← NEW
  │     └── Policies
  ├── Workflows (N)
  │     └── Tasks (N)
  ├── Channels (N)
  ├── Tools Registry (1)           ← NEW: shared tools
  ├── Memory Store (1)             ← NEW: persistent state
  ├── Event Bus (1)                ← NEW: system events
  ├── Plugin Registry (1)          ← NEW: extensions
  └── Knowledge Hub (1)
```

### Entities comparison with industry (CrewAI, AutoGen, LangGraph, Agno)

| Entity | CrewAI | AutoGen | LangGraph | Agno | **MagiC** |
|---|---|---|---|---|---|
| Agent/Worker | Agent | Agent | Node | Agent | **Worker** |
| Team/Group | Crew | GroupChat | Graph | Team | **Team** |
| Task | Task | — | — | — | **Task** |
| Workflow | Flow | Event-driven | Graph+Edges | Runtime | **Workflow** |
| Tool | Tool | Tool (MCP) | Tool | Tool | **Tool** ← added |
| Memory | Knowledge | — | Checkpoint | Memory | **Memory** ← added |
| Event | — | Event | — | — | **Event** ← added |
| Plugin | — | Custom | — | MCP | **Plugin** ← added |
| Channel | — | — | — | — | **Channel** (unique) |
| Organization | — | — | — | — | **Organization** (unique) |

MagiC has **10 entities** — 6 original + 4 new from best practice analysis. Organization and Channel are unique to MagiC (no competitor has these).

### 4.2 Entity Definitions

#### Organization

The top-level entity. One MagiC instance = one organization.

```go
type Organization struct {
    ID              string            `json:"id"`
    Name            string            `json:"name"`
    Teams           []string          `json:"teams"`           // team IDs
    GlobalPolicies  Policies          `json:"global_policies"`
    LLMProxy        LLMProxyConfig    `json:"llm_proxy"`       // LiteLLM config
    CostBudget      Budget            `json:"cost_budget"`
    KnowledgeHubID  string            `json:"knowledge_hub_id"`
    CreatedAt       time.Time         `json:"created_at"`
}

type LLMProxyConfig struct {
    BaseURL string `json:"base_url"` // e.g., "http://litellm:4000"
    APIKey  string `json:"api_key"`
}
```

#### Team

A group of workers organized by function.

```go
type Team struct {
    ID              string    `json:"id"`
    Name            string    `json:"name"`
    OrgID           string    `json:"org_id"`
    Workers         []string  `json:"workers"`          // worker IDs
    Policies        Policies  `json:"policies"`
    DailyBudget     float64   `json:"daily_budget"`
    ApprovalRequired bool     `json:"approval_required"`
}
```

#### Worker

An AI agent that has joined the organization.

```go
type Worker struct {
    ID              string          `json:"id"`
    Name            string          `json:"name"`
    TeamID          string          `json:"team_id"`
    Capabilities    []Capability    `json:"capabilities"`
    Endpoint        Endpoint        `json:"endpoint"`
    Limits          WorkerLimits    `json:"limits"`
    Status          WorkerStatus    `json:"status"`          // active | paused | offline
    CurrentLoad     int             `json:"current_load"`    // active tasks count
    TotalCostToday  float64         `json:"total_cost_today"`
    RegisteredAt    time.Time       `json:"registered_at"`
    LastHeartbeat   time.Time       `json:"last_heartbeat"`
    Metadata        map[string]any  `json:"metadata"`
}

type Capability struct {
    Name            string          `json:"name"`
    Description     string          `json:"description"`
    InputSchema     json.RawMessage `json:"input_schema"`
    OutputSchema    json.RawMessage `json:"output_schema"`
    EstCostPerCall  float64         `json:"est_cost_per_call"`
    AvgResponseMs   int64           `json:"avg_response_ms"`
}
```

#### Task

A unit of work with a contract.

```go
type Task struct {
    ID              string          `json:"id"`
    Type            string          `json:"type"`
    Priority        string          `json:"priority"`        // low | normal | high | critical
    Status          TaskStatus      `json:"status"`          // pending | assigned | accepted | in_progress | completed | failed
    Input           json.RawMessage `json:"input"`
    Output          json.RawMessage `json:"output,omitempty"`
    Contract        Contract        `json:"contract"`
    Routing         RoutingConfig   `json:"routing"`
    AssignedWorker  string          `json:"assigned_worker,omitempty"`
    WorkflowID      string          `json:"workflow_id,omitempty"`
    Context         TaskContext     `json:"context"`
    Cost            float64         `json:"cost"`
    CreatedAt       time.Time       `json:"created_at"`
    CompletedAt     *time.Time      `json:"completed_at,omitempty"`
    Error           *TaskError      `json:"error,omitempty"`
}
```

#### Workflow

A chain of tasks (DAG) executed sequentially or in parallel.

```go
type Workflow struct {
    ID              string          `json:"id"`
    Name            string          `json:"name"`
    Steps           []WorkflowStep  `json:"steps"`
    Status          string          `json:"status"`
    Context         TaskContext     `json:"context"`
    CreatedAt       time.Time       `json:"created_at"`
}

type WorkflowStep struct {
    ID              string          `json:"id"`
    TaskType        string          `json:"task_type"`
    Input           json.RawMessage `json:"input"`
    DependsOn       []string        `json:"depends_on"`     // step IDs that must complete first
    Condition       *StepCondition  `json:"condition,omitempty"`
    OnFailure       string          `json:"on_failure"`     // skip | abort | retry
}
```

#### Channel

The interface through which users interact with the organization.

```go
type Channel struct {
    ID              string          `json:"id"`
    Type            string          `json:"type"`            // api | slack | telegram | web | zalo
    Config          json.RawMessage `json:"config"`
    InputAdapter    string          `json:"input_adapter"`   // transforms channel input to task
    OutputFormatter string          `json:"output_formatter"`
    AuthConfig      json.RawMessage `json:"auth_config"`
}
```

#### Tool

A reusable capability that workers can use. Registered centrally, shared across workers.

```go
type Tool struct {
    ID              string          `json:"id"`
    Name            string          `json:"name"`           // e.g., "web_search", "sql_query"
    Description     string          `json:"description"`
    Type            string          `json:"type"`           // mcp | api | function
    Config          json.RawMessage `json:"config"`         // MCP server URL, API endpoint, etc.
    InputSchema     json.RawMessage `json:"input_schema"`
    OutputSchema    json.RawMessage `json:"output_schema"`
    RequiresApproval bool           `json:"requires_approval"` // human must approve before use
    AllowedWorkers  []string        `json:"allowed_workers"`   // empty = all workers can use
}
```

#### Memory

Persistent state across tasks and sessions. Workers can read/write shared memory.

```go
type Memory struct {
    ID              string          `json:"id"`
    Scope           string          `json:"scope"`          // org | team | worker | workflow
    ScopeID         string          `json:"scope_id"`       // which org/team/worker/workflow
    Key             string          `json:"key"`
    Value           json.RawMessage `json:"value"`
    TTL             *time.Duration  `json:"ttl,omitempty"`  // auto-expire
    CreatedBy       string          `json:"created_by"`     // worker ID
    UpdatedAt       time.Time       `json:"updated_at"`
}
```

#### Event

First-class system event for monitoring, triggers, and event-driven workflows.

```go
type Event struct {
    ID              string          `json:"id"`
    Type            string          `json:"type"`           // task.completed, worker.registered, budget.exceeded, etc.
    Source          string          `json:"source"`         // module or worker that emitted
    Payload         json.RawMessage `json:"payload"`
    Timestamp       time.Time       `json:"timestamp"`
    Severity        string          `json:"severity"`       // info | warn | error | critical
}
```

#### Plugin

Extension point for custom modules, middleware, and integrations.

```go
type Plugin struct {
    ID              string          `json:"id"`
    Name            string          `json:"name"`
    Type            string          `json:"type"`           // middleware | router_strategy | evaluator | storage | channel
    Version         string          `json:"version"`
    EntryPoint      string          `json:"entry_point"`    // Go plugin path or gRPC endpoint
    Config          json.RawMessage `json:"config"`
    Hooks           []string        `json:"hooks"`          // events this plugin subscribes to
}
```

---

## 5. Module Architecture

```
                    ┌──────────────────────────────────────────┐
                    │              MagiC Core (Go)             │
                    ├──────────────────────────────────────────┤
  User Request ──►  │  Gateway                                 │
                    │    │                                      │
                    │    ▼                                      │
                    │  Router ──► Registry (find best worker)   │
                    │    │                                      │
                    │    ▼                                      │
                    │  Orchestrator (multi-step workflows)      │
                    │    │                                      │
                    │    ▼                                      │
                    │  ┌─────────────────────────────────┐     │
                    │  │  Worker A  │ Worker B │ Worker C │     │
                    │  └─────────────────────────────────┘     │
                    │    │                                      │
                    │    ▼                                      │
                    │  Monitor ◄── all events flow here        │
                    │  Evaluator ◄── quality check on output   │
                    │  Cost Controller ◄── track spending      │
                    │  Org Manager ◄── enforce policies        │
                    │  Knowledge Hub ◄── shared context        │
                    └──────────────────────────────────────────┘
```

### Module Tiers

| Tier | Modules | Build Order | Purpose |
|---|---|---|---|
| **Core** | Gateway, Registry, Router, Monitor | Weeks 1-2 | Framework doesn't run without these |
| **Differentiator** | Orchestrator, Evaluator, Cost Controller, Org Manager | Weeks 3-5 | These create the "wow" factor |
| **Bonus** | Knowledge Hub | Week 6 | Shared knowledge across workers |

---

## 6. Tech Stack & Project Structure

### Language Choice

| Component | Language | Rationale |
|---|---|---|
| Core framework (9 modules) | **Go** | Fast, goroutines for concurrency, small binary, infra precedent (K8s, Docker) |
| Worker SDK | **Python** | AI/ML ecosystem (LangChain, LiteLLM), user familiarity |
| Worker SDK | **Go** | For Go developers who want native performance |
| Protocol spec | **Language-agnostic** | JSON schema, any language can implement |

### Project Structure

```
magic-claw/                          # repo root (keep name for git)
├── core/                            # Go — MagiC framework
│   ├── cmd/
│   │   └── magic/                   # CLI entrypoint: `magic serve`, `magic worker list`
│   │       └── main.go
│   ├── internal/
│   │   ├── gateway/                 # HTTP/gRPC entry point
│   │   │   ├── gateway.go
│   │   │   ├── middleware.go        # auth, rate limit, request ID
│   │   │   └── routes.go
│   │   ├── registry/                # Worker registration & discovery
│   │   │   ├── registry.go
│   │   │   ├── store.go             # in-memory + persistent store
│   │   │   └── health.go            # heartbeat monitoring
│   │   ├── router/                  # Task routing engine
│   │   │   ├── router.go
│   │   │   ├── strategy.go          # best_match, round_robin, cheapest
│   │   │   └── scorer.go            # capability matching score
│   │   ├── monitor/                 # Observability
│   │   │   ├── monitor.go
│   │   │   ├── events.go            # event bus
│   │   │   ├── metrics.go           # prometheus metrics
│   │   │   └── logger.go            # structured JSON logging
│   │   ├── orchestrator/            # Multi-worker workflow execution
│   │   │   ├── orchestrator.go
│   │   │   ├── dag.go               # DAG execution engine
│   │   │   └── workflow.go
│   │   ├── evaluator/               # Output quality assessment
│   │   │   ├── evaluator.go
│   │   │   ├── criteria.go          # quality criteria checks
│   │   │   └── scorer.go
│   │   ├── costctrl/                # Budget & cost management
│   │   │   ├── controller.go
│   │   │   ├── budget.go
│   │   │   └── alerts.go
│   │   ├── orgmgr/                  # Organization, teams, RBAC
│   │   │   ├── manager.go
│   │   │   ├── team.go
│   │   │   ├── policy.go
│   │   │   └── rbac.go
│   │   ├── knowledge/               # Shared knowledge hub
│   │   │   ├── hub.go
│   │   │   └── store.go
│   │   └── protocol/                # MCP² message types & serialization
│   │       ├── messages.go
│   │       ├── types.go
│   │       └── validate.go
│   ├── pkg/                         # Public Go packages
│   │   └── mcp2/                    # Go SDK for building workers
│   │       ├── client.go
│   │       ├── worker.go
│   │       └── types.go
│   ├── go.mod
│   └── go.sum
│
├── sdk/
│   └── python/                      # Python SDK
│       ├── magic_claw/
│       │   ├── __init__.py
│       │   ├── worker.py            # Base worker class
│       │   ├── client.py            # MagiC API client
│       │   ├── protocol.py          # MCP² message types
│       │   └── decorators.py        # @capability decorator
│       ├── pyproject.toml
│       └── README.md
│
├── protocol/                        # Protocol specification
│   ├── spec.md                      # Human-readable spec
│   ├── schema/                      # JSON schemas
│   │   ├── worker.register.json
│   │   ├── task.assign.json
│   │   └── ...
│   └── conformance/                 # Conformance test suite
│       └── test_worker.py
│
├── examples/
│   ├── hello-worker/                # Simplest possible worker (5 min setup)
│   │   ├── main.py
│   │   └── README.md
│   ├── ceo-division/                # First use case
│   │   ├── docker-compose.yml
│   │   ├── workers/
│   │   └── README.md
│   └── openclaw-integration/        # OpenClaw as MagiC worker
│       └── README.md
│
├── docs/                            # Documentation
│   ├── getting-started.md
│   ├── architecture.md
│   ├── protocol.md
│   └── superpowers/specs/           # Design specs
│
├── docker-compose.yml               # Production
├── docker-compose.dev.yml           # Development overrides
├── Dockerfile                       # Multi-stage (dev + prod)
├── Makefile                         # make dev, make build, make test
└── README.md                        # GitHub README (viral-optimized)
```

### Storage

| Data | Storage | Rationale |
|---|---|---|
| Worker registry | In-memory (sync to disk) | Fast lookup, simple. SQLite for persistence |
| Tasks & results | SQLite (dev) / PostgreSQL (prod) | Structured, queryable |
| Metrics & events | Event bus (in-memory) → optional Prometheus export | Real-time monitoring |
| Knowledge base | File-based + vector store (later) | Start simple |
| Configuration | YAML files + env vars | Standard practice |
| Memory store | In-memory + SQLite | Persistent worker state |

### Extensibility Architecture

MagiC is designed for high extensibility via a plugin system. Every major component can be extended or replaced.

| Extension Point | What you can customize | How |
|---|---|---|
| **Router Strategy** | Custom task routing logic | Implement `RouterStrategy` interface, register via Plugin |
| **Evaluator Criteria** | Custom quality checks | Implement `EvaluationCriteria` interface |
| **Storage Backend** | Custom persistence (Redis, MongoDB...) | Implement `Store` interface |
| **Channel Adapter** | New communication channels | Implement `ChannelAdapter` interface |
| **Middleware** | Custom Gateway middleware | Standard HTTP middleware chain |
| **Event Hooks** | React to system events | Subscribe to Event Bus via Plugin hooks |
| **Tool Provider** | Custom tool integrations | Register Tool with MCP or native config |

```go
// Example: custom router strategy plugin
type RouterStrategy interface {
    Name() string
    Score(task Task, workers []Worker) []WorkerScore
}

// Example: custom evaluator
type EvaluationCriteria interface {
    Name() string
    Evaluate(task Task, output json.RawMessage) (score float64, pass bool, err error)
}

// Example: custom storage backend
type Store interface {
    Get(key string) ([]byte, error)
    Set(key string, value []byte, ttl time.Duration) error
    Delete(key string) error
    List(prefix string) ([]string, error)
}
```

---

## 7. Module Specifications

### 7.1 Gateway

**Purpose:** Single entry point for all requests. Auth, rate limiting, request routing.

**Responsibilities:**
- Accept HTTP/gRPC requests from users and channels
- Authenticate requests (API key, JWT)
- Rate limiting per user/team
- Inject request ID for tracing
- Forward to Router or Orchestrator
- Serve MCP² protocol endpoints for worker communication

**Key APIs:**

```
POST   /api/v1/tasks              # Submit a task
GET    /api/v1/tasks/{id}         # Get task status/result
POST   /api/v1/workflows          # Submit a workflow
GET    /api/v1/workers             # List registered workers
POST   /api/v1/workers/register    # Worker registration (MCP²)
POST   /api/v1/workers/heartbeat   # Worker heartbeat (MCP²)
GET    /api/v1/metrics             # Prometheus metrics
GET    /health                     # Liveness
GET    /health/ready               # Readiness (all modules up)
```

### 7.2 Registry

**Purpose:** Track all registered workers, their capabilities, and health status.

**Responsibilities:**
- Store worker registrations
- Match workers by capability
- Monitor heartbeats, mark workers as offline after timeout
- Provide worker discovery API to Router

**Key Logic:**

```go
func (r *Registry) FindWorkers(capability string) []Worker
func (r *Registry) BestMatch(requirements []string) (Worker, error)
func (r *Registry) HealthCheck() // goroutine: check heartbeats every 30s
```

### 7.3 Router

**Purpose:** Intelligently assign tasks to the best available worker.

**Routing Strategies:**

| Strategy | Description |
|---|---|
| `best_match` | Highest capability match score + lowest load |
| `round_robin` | Distribute evenly across capable workers |
| `cheapest` | Lowest estimated cost per call |
| `fastest` | Lowest average response time |
| `specific` | Route to a specific worker by ID |

**Scoring Algorithm:**

```
score = capability_match * 0.4
      + availability * 0.3
      + performance_history * 0.2
      + cost_efficiency * 0.1
```

### 7.4 Monitor

**Purpose:** Real-time observability for all events in the system.

**What it tracks:**
- All MCP² messages (with structured JSON logging)
- Task durations, success/failure rates
- Worker load, response times
- Cost per task, per worker, per team
- System health metrics

**Output formats:**
- Structured JSON logs (stdout)
- Prometheus metrics endpoint (`/api/v1/metrics`)
- Event bus (internal, for other modules to subscribe)

### 7.5 Orchestrator

**Purpose:** Execute multi-step workflows (DAGs) across multiple workers.

**Capabilities:**
- Parse workflow definition (DAG of steps)
- Execute steps respecting dependencies
- Parallel execution where possible
- Pass output of step N as input to step N+1
- Handle step failures (skip, abort, retry)
- Human approval gates between steps

### 7.6 Evaluator

**Purpose:** Assess output quality of worker responses.

**Evaluation Types:**

| Type | Description | How |
|---|---|---|
| Schema validation | Output matches expected schema | JSON Schema validation |
| Quality criteria | Custom criteria (no hallucination, SEO score) | LLM-as-judge via LiteLLM |
| Historical comparison | Compare with previous outputs | Statistical analysis |

### 7.7 Cost Controller

**Purpose:** Track and control spending across the organization.

**Features:**
- Real-time cost tracking per task, worker, team, org
- Budget alerts (50%, 80%, 100% thresholds)
- Auto-pause workers that exceed daily budget
- Cost reports (daily, weekly, monthly)

### 7.8 Org Manager

**Purpose:** Manage organizational structure, teams, roles, and policies.

**Features:**
- CRUD for teams
- Worker-to-team assignment
- Policy enforcement (approval workflows, budget limits)
- RBAC: who can create/modify/delete workers, teams, workflows
- Audit log for all organizational changes

### 7.9 Knowledge Hub

**Purpose:** Shared knowledge base accessible by all workers.

**Initial implementation (simple):**
- File-based document storage
- Workers can query knowledge via a special capability
- Metadata tagging and search

**Future:**
- RAG pipeline with vector embeddings
- Per-team knowledge scoping

---

## 8. Data Flow & Sequences

### 8.1 Simple Task Flow

```
User                    Gateway     Router      Registry    Worker
  │                        │           │           │          │
  │── POST /tasks ────────►│           │           │          │
  │                        │──route──►│           │          │
  │                        │           │──find────►│          │
  │                        │           │◄──worker──│          │
  │                        │◄─worker──│           │          │
  │                        │                                  │
  │                        │──── task.assign ────────────────►│
  │                        │◄─── task.accept ────────────────│
  │                        │◄─── task.progress (50%) ────────│
  │                        │◄─── task.complete ──────────────│
  │                        │                                  │
  │                        │── Evaluator: check quality       │
  │                        │── CostCtrl: record cost          │
  │                        │── Monitor: log event             │
  │                        │                                  │
  │◄── 200 { result } ────│                                  │
```

### 8.2 Complex Workflow — DAG Execution (Parallel + Dependencies)

**Example:** "Launch product campaign" — 5 steps, 2 parallel branches

```yaml
workflow: "Product Launch Campaign"
steps:
  - id: research
    type: market_research
    team: marketing
  - id: content
    type: content_writing
    team: marketing
    depends_on: [research]
  - id: seo
    type: seo_optimization
    team: marketing
    depends_on: [content]
    on_failure: skip
  - id: leads
    type: lead_generation
    team: sales
    depends_on: [research]        # PARALLEL with content
  - id: outreach
    type: email_outreach
    team: sales
    depends_on: [leads, content]  # WAITS for BOTH branches
    approval_required: true       # CEO must approve before execution
```

**DAG visualization:**

```
        research (step 1)
       /         \
      v           v
  content       leads        ← 2 branches run IN PARALLEL
  (step 2)     (step 4)
      |           |
      v           |
    seo           |
  (step 3)        |
      \          /
       v        v
   [CEO APPROVAL GATE]
          |
          v
     outreach (step 5)       ← WAITS for BOTH branches to complete
```

**Orchestrator execution algorithm:**

```
1. Parse workflow definition → build DAG (directed acyclic graph)

2. Find steps with NO dependencies → step 1 (research)
   Router finds best worker → Marketing.ContentBot
   Send task.assign → worker executes

3. Step 1 COMPLETE → Orchestrator checks dependents:
   - Step 2 (content): depends_on [research ✓] → ALL deps met → ASSIGN
   - Step 4 (leads):   depends_on [research ✓] → ALL deps met → ASSIGN
   → TWO tasks dispatched IN PARALLEL (goroutines)

4. Step 2 COMPLETE → check dependents:
   - Step 3 (seo):     depends_on [content ✓] → ASSIGN
   - Step 5 (outreach): depends_on [content ✓, leads ?]
     → Step 4 NOT done yet → WAIT (do not assign)

5. Step 4 COMPLETE → check dependents:
   - Step 5 (outreach): depends_on [content ✓, leads ✓] → ALL deps met
     → approval_required: true → PAUSE, notify CEO

6. CEO reviews outreach input → APPROVES → Orchestrator ASSIGNS step 5

7. Step 5 COMPLETE → No more steps → Workflow DONE
   → Monitor logs total duration, cost
   → Evaluator scores overall quality
   → CostCtrl records total spend
```

**Error handling per step:**

```
on_failure options:
  "retry"  → Retry up to max_retries (default 2), with backoff
  "skip"   → Mark step as skipped, continue downstream steps
  "abort"  → Stop entire workflow, notify user
  "reassign" → Router picks a DIFFERENT worker, try again

Example: Step 3 (seo) fails
  on_failure: "skip"
  → Step 3 marked SKIPPED
  → Step 5 (outreach) can still proceed (seo was not in its depends_on)
  → CEO notified: "SEO optimization was skipped due to failure"

Example: Step 2 (content) fails
  on_failure: "retry" (2 attempts)
  → Attempt 1 fails → Attempt 2 fails
  → Fallback: "reassign" → Router picks different worker
  → Still fails → "abort"
  → Steps 3, 5 are BLOCKED
  → User notified: "Content writing failed. Steps 3, 5 blocked. Action required."
  → User can: manually retry, assign specific worker, or cancel workflow
```

### 8.3 Worker Delegation Flow

When a worker needs help from another worker mid-task:

```
Worker A (content_writing) is assigned "Write data-driven blog"
  │
  │ Needs statistics data → sends worker.delegate
  │   { delegate_to_capability: "data_analysis",
  │     input: { query: "AI market stats 2026" } }
  │
  ▼
Organization receives delegate request
  │
  Router finds Worker B (data_analysis capability)
  │
  ├── task.assign → Worker B
  │       │
  │   task.complete → { stats: [...] }
  │       │
  └── Forward result back to Worker A
      │
  Worker A continues with stats data
  │
  task.complete → final blog post
```

---

## 9. First Use Case: CEO Division Orchestration

### Scenario

CEO manages a company with multiple divisions. Each division has 1+ AI workers.

```
CEO (user)
├── Marketing Division
│   ├── ContentBot (writes content)
│   └── SEOBot (optimizes for search)
├── Sales Division
│   ├── LeadBot (qualifies leads)
│   └── OutreachBot (sends personalized emails)
├── Engineering Division
│   ├── CodeReviewBot (reviews PRs)
│   └── DocBot (generates documentation)
└── Finance Division
    └── ReportBot (generates financial reports)
```

### Example Workflows

**1. "Launch new product campaign"** (cross-division)

```yaml
workflow:
  name: "Product Launch Campaign"
  steps:
    - id: research
      type: market_research
      team: marketing
    - id: content
      type: content_writing
      team: marketing
      depends_on: [research]
    - id: seo
      type: seo_optimization
      team: marketing
      depends_on: [content]
    - id: leads
      type: lead_generation
      team: sales
      depends_on: [research]     # parallel with content
    - id: outreach
      type: email_outreach
      team: sales
      depends_on: [leads, content]
```

**2. CEO dashboard view:**

```
┌─────────────────────────────────────────┐
│  MagiC Dashboard — CEO View            │
├─────────────────────────────────────────┤
│  Workers: 7 active / 0 offline          │
│  Tasks today: 23 completed / 2 failed   │
│  Cost today: $4.52 / $20.00 budget      │
│                                         │
│  Division Performance:                  │
│  Marketing  ████████░░ 82% quality      │
│  Sales      ██████████ 95% quality      │
│  Engineering███████░░░ 70% quality      │
│  Finance    █████████░ 91% quality      │
└─────────────────────────────────────────┘
```

---

## 10. Open-Source & Viral Strategy

### GitHub README Structure (optimized for virality)

```markdown
# MagiC

> Don't build another AI. Manage the ones you have.

[hero image/animation showing multiple workers joining]

## What is MagiC?

One paragraph. Clear, no jargon.

## Quick Start (< 5 minutes)

\`\`\`bash
# Install
go install github.com/kienbm/magic-claw/cmd/magic@latest

# Start MagiC
magic serve

# Register your first worker (in another terminal)
pip install magic-claw
python examples/hello-worker/main.py
\`\`\`

## Why MagiC?

| Without MagiC | With MagiC |
| ... | ... |

## Architecture (1 diagram)

## Star History / Community
```

### Python SDK Hello World (must be < 20 lines)

```python
from magic_claw import Worker, capability

worker = Worker(name="HelloBot", endpoint="http://localhost:9000")

@worker.capability("greeting")
def greet(name: str) -> str:
    return f"Hello, {name}! I'm managed by MagiC."

worker.register("http://localhost:8899")  # MagiC server
worker.serve()
```

### Viral Checklist

- [ ] README with hero animation
- [ ] "Quick Start" in < 5 minutes
- [ ] Hello World worker in < 20 lines
- [ ] Architecture diagram (1 image)
- [ ] Comparison table (vs CrewAI, AutoGen, Agno)
- [ ] Docker one-liner: `docker run magic-claw`
- [ ] Example: OpenClaw integration
- [ ] Example: CEO Division use case
- [ ] Contributing guide
- [ ] Discord/community link

---

## 11. Risks & Mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| 9 modules = large scope | HIGH | 3-tier build order. Tier 1 (core) testable in 2 weeks |
| Go learning curve | MEDIUM | Go syntax simple. Start with Gateway (HTTP handler) to learn |
| Protocol doesn't fit real use | HIGH | Build alongside CEO use case. Extract patterns from real code |
| MS/Google enters with similar tool | MEDIUM | They lock-in (Azure/GCP). MagiC = open, any-cloud |
| Protocol fragmentation (forks) | LOW | Conformance test suite. Semantic versioning |
| Adoption too slow | MEDIUM | Viral README. Hello World in 5 min. OpenClaw integration as hook |

---

## Appendix: Key Decisions Log

| Decision | Choice | Rationale |
|---|---|---|
| Name | MagiC (C = Company/Crew/Claw) | Short, memorable, multi-meaning |
| Core language | Go | Performance, concurrency, infra precedent |
| SDK language | Python + Go | AI ecosystem + native option |
| MVP scope | Full 9 modules | Market competitive, needs "wow" |
| License | Apache 2.0 | Standard for infra open-source |
| First use case | CEO division orchestration | Multi-worker, cross-team, real complexity |
| Storage | SQLite (dev) / PostgreSQL (prod) | Simple start, scale later |
| Protocol name | MCP² (MagiC Protocol) | Nod to MCP (Anthropic), differentiated |
