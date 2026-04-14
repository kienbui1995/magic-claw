# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.0] - 2026-04-14

### Added — AI-Native Features
- **LLM Gateway** — multi-provider routing (OpenAI, Anthropic, Ollama) with strategies: `cheapest`, `fastest`, `best`. Automatic fallback, token counting, cost calculation per request. API: `POST /api/v1/llm/chat`, `GET /api/v1/llm/models`
- **Prompt Registry** — versioned prompt templates with `{{variable}}` interpolation and A/B testing (weighted variant selection). API: `POST /api/v1/prompts`, `GET /api/v1/prompts`, `POST /api/v1/prompts/render`
- **Agent Memory** — short-term conversation history (sliding window per session) + long-term vector recall via pgvector. API: `POST /api/v1/memory/turns`, `GET /api/v1/memory/turns`, `POST /api/v1/memory/entries`
- **LLM cost → costctrl integration** — LLM spend flows into budget system via `OnCost` callback
- **AI event bus integration** — `llm.chat`, `prompt.created`, `memory.turn_added` events published for webhooks/audit

### Added — Framework Features
- **TypeScript SDK** — full SDK with Worker + Client classes, tests, npm publish in CI
- **CLI commands** — `magic workers`, `magic tasks`, `magic submit <type>`, `magic status <id>`, `magic version`
- **W3C Trace Context** — distributed tracing with `traceparent` header propagation, instrumented dispatcher
- **Task DLQ** — dead letter queue for permanently failed tasks (persisted to DB). API: `GET /api/v1/dlq`
- **Worker auto-discovery** — UDP broadcast for local dev (Listener + Announcer)
- **Cluster mode** — leader election via PostgreSQL advisory locks for horizontal scaling
- **YAML config** — `magic serve --config magic.yaml` with env var override. Example: `magic.yaml.example`
- **Docker Hub image** — multi-platform (amd64/arm64) push in release workflow

### Fixed — Security
- **Webhook SSRF** — URL validation blocks private IPs + DNS rebinding (resolves hostname before IP check)
- **Cost controller race condition** — atomic read-modify-write under lock
- **X-Forwarded-For spoofing** — only trusted when `MAGIC_TRUSTED_PROXY=true`
- **LLM Gateway lock race** — fixed RUnlock→Lock→Unlock→RLock pattern

### Fixed — Reliability
- **TotalCostToday daily reset** — midnight UTC reset + auto-unpause budget-paused workers
- **Graceful shutdown** — `ShutdownCtx` + `WaitGroup` for gateway + orchestrator dispatches
- **Event bus drop metric** — `magic_events_dropped_total` Prometheus counter
- **Rate limiter goroutine leak** — added stop channel
- **Health check** — skip marking offline if worker has in-flight tasks (`CurrentLoad > 0`)
- **DLQ persistence** — SQLite + PostgreSQL tables (migration 003)
- **Prompts + Memory persistence** — SQLite + PostgreSQL tables (migration 004)
- **Pagination** — all list endpoints now paginate (tokens, webhooks, deliveries, roles, policies, DLQ)
- **Unbounded data caps** — cost records (50K), DLQ memory (10K), LLM history (10K)
- **Dockerfile** — runs as non-root user

### Changed
- README repositioned as AI-native framework with comparison tables
- Config module replaces direct env var reads in main.go

## [0.7.0] - 2026-04-13

### Added
- **Plugin system** — extensible interfaces for Router (`Strategy`), Evaluator (`EvalPlugin`), Cost Controller (`CostPolicy`), and Monitor (`LogSink`). Built-in implementations registered by default; custom plugins via `Register*()` methods
- **RBAC** — role-based access control with 3 roles: `owner` (full), `admin` (read/write/delete), `viewer` (read-only). Dev mode (no bindings) allows all. API: `POST/GET/DELETE /api/v1/orgs/{orgID}/roles`
- **Policy Engine** — hard/soft guardrails enforced before task routing. 4 built-in rules: `allowed_capabilities`, `blocked_capabilities`, `max_cost_per_task`, `max_timeout_ms`. Hard violations return 403; soft violations audit + warn. API: full CRUD at `/api/v1/orgs/{orgID}/policies`
- **Worker manifest** — `tags` (map[string]string) and `session_mode` (stateless/sessionful) fields on Worker
- **TraceID** — auto-generated `trace_id` on every task and workflow, propagated to workers via `X-Trace-ID` header
- **Per-org rate limiting** — task submission rate limited per org via `X-Org-ID` header (in addition to existing per-IP limits)
- **TypeScript SDK** — `sdk/typescript/` with `Worker` and `MagiCClient` classes, zero runtime dependencies
- **VitePress docs** — Governance section with RBAC and Policy guides, updated API reference

### Changed
- Router refactored from switch/case to strategy registry pattern
- Evaluator refactored from monolithic to plugin chain
- Cost Controller refactored from hardcoded thresholds to policy chain
- Monitor refactored from single writer to multi-sink fan-out
- Policy enforcement added to both `handleSubmitTask` and `handleStreamTask`
- Go version in release workflow updated to 1.25

## [0.6.0] - 2026-04-09

### Added
- **PostgreSQL backend** — `MAGIC_POSTGRES_URL` auto-selects PostgreSQL; auto-runs golang-migrate migrations on startup
- **SQLite persistent storage** — `MAGIC_STORE=path.db` for single-instance persistence (was always there, now documented)
- **pgvector semantic search** — `POST /knowledge/{id}/embedding` stores embeddings; `POST /knowledge/search/semantic` for cosine similarity search
- **SSE streaming** — `POST /api/v1/tasks/stream` submits and streams task output; `GET /api/v1/tasks/{id}/stream` for reconnection
- **Webhooks (at-least-once)** — `POST /orgs/{orgID}/webhooks` registers endpoints; events delivered with HMAC-SHA256 signature, exponential backoff retry (30s→5m→30m→2h→8h)
- **Prometheus metrics** — `GET /metrics` (unauthenticated) exports 14 metrics covering tasks, workers, cost, workflows, knowledge, webhooks, and SSE streams
- **Go SDK** — `sdk/go/` with Worker struct, auto-discovery, `Worker.Run()`, `SubmitAndWait()`
- **Worker token authentication** — per-org tokens for worker auth (`POST /orgs/{orgID}/tokens`)
- **Audit log** — all API actions logged; queryable via `GET /orgs/{orgID}/audit`
- **Rate limiting** — per-endpoint token bucket rate limits with Prometheus instrumentation

### Changed
- Go version updated to 1.25+
- Python SDK package name: `magic-ai-sdk` (import as `from magic_ai_sdk import Worker`)

## [0.2.0] - 2026-03-17

### Added
- SQLite persistent storage backend (`MAGIC_STORE=path.db`)
- Human-in-the-loop approval gates for workflow steps
- Step output flows to dependent steps via `_deps` field
- Template workers: Summarizer, Translator, Classifier, Extractor, Generator
- CrewAI integration guide
- Landing page (`site/index.html`)
- Async Python client (`AsyncMagiCClient`) with full API coverage
- Full sync Python client with all endpoints (tasks, workflows, teams, costs, metrics, knowledge)
- API key authentication support in Python SDK
- Release CI workflow (binary builds for linux/darwin + Docker image to GHCR)
- `SECURITY.md`, GitHub issue/PR templates
- README badges (CI, Go, Python, License)

### Changed
- Renamed Python SDK from `magic-claw` to `magic-ai-sdk` (`pip install magic-ai-sdk`, `from magic_ai_sdk import Worker`)
- Go version requirement updated to 1.24+ (Dockerfile, CI, docs)

### Fixed
- Router race condition: worker load now persisted via `store.UpdateWorker()` instead of direct pointer mutation
- Orchestrator workflow state race: added mutex to protect concurrent step completions
- Event bus now logs panics instead of silently swallowing them
- Dispatcher retry with linear backoff on worker failure
- Router enforces `MaxConcurrentTasks` and priority-aware scoring
- `DELETE /api/v1/workers/{id}` endpoint added

## [0.1.1] - 2026-03-16

### Fixed
- Deep copy store to prevent data races
- Stable pagination ordering
- Health check graceful shutdown
- All data races resolved (`go test -race` clean)

## [0.1.0] - 2026-03-16

### Added
- Core server (Go): Gateway, Registry, Router, Dispatcher, Monitor
- MCP² protocol: JSON over HTTP message format
- Worker registration with capability-based discovery
- Task routing: best_match, cheapest, specific strategies
- DAG workflow orchestrator with parallel execution
- Cost controller with budget alerts (80%) and auto-pause (100%)
- Output evaluator with JSON schema validation
- Organization/team management
- Knowledge hub (shared context)
- Python SDK (`Worker` class, `@capability` decorator)
- In-memory store with thread-safe deep copies
- API key authentication, CORS, body size limits, SSRF protection
- Hello worker and multi-worker examples
- CI with Go race detection and Python tests
- Multi-stage Dockerfile
