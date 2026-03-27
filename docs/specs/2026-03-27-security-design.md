# Zero-Trust Worker Security - Design Specification

**Version:** 1.0.0
**Date:** 2026-03-27
**Author:** Kien (kienbm) + Claude
**Status:** Draft
**Requirements:** `docs/specs/2026-03-27-security-requirements.md`
**Competitive Analysis:** `docs/competitive/2026-03-27-speed-security-analysis.md`

---

## Table of Contents

1. [Summary](#1-summary)
2. [Architecture Direction](#2-architecture-direction)
3. [Data Model](#3-data-model)
4. [Token Format and Generation](#4-token-format-and-generation)
5. [Authentication Flow](#5-authentication-flow)
6. [Org Isolation](#6-org-isolation)
7. [Audit Log](#7-audit-log)
8. [API Contracts](#8-api-contracts)
9. [Edge Cases and Error Handling](#9-edge-cases-and-error-handling)
10. [Testing Strategy](#10-testing-strategy)
11. [Migration Path from Current Auth](#11-migration-path-from-current-auth)
12. [Open Questions](#12-open-questions)

---

## 1. Summary

MagiC currently uses a single shared `MAGIC_API_KEY` environment variable for all server authentication. This means every worker and every API client shares the same credential. There is no way to distinguish which worker made a request, no per-worker revocation, no org-level isolation, and no audit trail tied to identity.

This design introduces per-worker tokens, org-scoped isolation, and an immutable audit log. The goal is to become the first AI agent framework with zero-trust worker identity -- a competitive moat that no competitor (AutoGen, CrewAI, Agno, LangGraph) currently offers.

**Scope:** This design covers the "Must Have" requirements from the requirements doc. RBAC, OAuth/SSO, rate limiting per worker, and token encryption at rest are explicitly out of scope.

---

## 2. Architecture Direction

**Approach: Token-per-worker, validated at Gateway middleware.**

The current `authMiddleware` in `core/internal/gateway/middleware.go` checks a single `MAGIC_API_KEY` from env. This design replaces that model for worker-facing endpoints with per-worker token validation, while keeping the shared API key for admin/user-facing endpoints (task submission, team management, etc.).

Two authentication realms:

| Realm | Endpoints | Auth Mechanism | Who Uses It |
|-------|-----------|---------------|-------------|
| **Admin** | `POST /api/v1/tasks`, `POST /api/v1/teams`, `GET /api/v1/workers`, `GET /api/v1/metrics`, etc. | `MAGIC_API_KEY` (existing behavior) | Org admins, API clients, dashboards |
| **Worker** | `POST /api/v1/workers/register`, `POST /api/v1/workers/heartbeat`, `DELETE /api/v1/workers/{id}`, task callbacks | Per-worker `worker_token` in request body or header | AI workers (Python SDK, Go SDK) |

Why two realms:
- Worker registration itself needs a bootstrap credential (the org admin creates a token, gives it to the worker).
- Admin endpoints should not require worker-level tokens -- an admin listing workers should not need to authenticate as a specific worker.
- This matches the Kubernetes model: kubeconfig for admins, ServiceAccount tokens for pods.

---

## 3. Data Model

### 3.1 WorkerToken Entity

A new entity representing an authentication credential for a specific worker within a specific org.

```go
// core/internal/protocol/types.go

// WorkerToken represents an authentication credential issued to a worker.
type WorkerToken struct {
    ID        string     `json:"id"`         // token_<random>, used as revocation handle
    OrgID     string     `json:"org_id"`     // which org this token belongs to
    WorkerID  string     `json:"worker_id"`  // empty until worker registers with this token
    TokenHash string     `json:"-"`          // SHA-256 hash of the actual token, never serialized
    Name      string     `json:"name"`       // human-readable label, e.g. "content-bot-prod"
    ExpiresAt *time.Time `json:"expires_at,omitempty"` // nil = no expiry
    RevokedAt *time.Time `json:"revoked_at,omitempty"` // non-nil = revoked
    CreatedAt time.Time  `json:"created_at"`
}

// IsValid checks if the token is not expired and not revoked.
func (t *WorkerToken) IsValid() bool {
    if t.RevokedAt != nil {
        return false
    }
    if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
        return false
    }
    return true
}
```

Design decisions:
- **Token ID vs Token Value:** The API returns the raw token value only once at creation time. The store keeps only the SHA-256 hash. This means a leaked database does not expose usable tokens.
- **WorkerID initially empty:** A token is created before a worker registers. When a worker calls `POST /api/v1/workers/register` with a valid token, the token gets bound to that worker's ID. This is a one-time binding -- once bound, the token cannot be used by a different worker.
- **OrgID on the token:** This is the key to org isolation. A token created under `org_acme` can only register workers into `org_acme`, and those workers can only see tasks and other workers within `org_acme`.

### 3.2 Worker Entity Changes

The `Worker` struct in `core/internal/protocol/types.go` gains an `OrgID` field:

```go
type Worker struct {
    ID             string            `json:"id"`
    Name           string            `json:"name"`
    OrgID          string            `json:"org_id"`             // NEW: which org this worker belongs to
    TeamID         string            `json:"team_id,omitempty"`
    // ... rest unchanged
}
```

Currently, `Worker` has no `OrgID`. The `TaskContext` has an `OrgID` field but it is optional and not enforced anywhere. This design makes `OrgID` a required, system-set field on every worker, derived from the token used to register.

### 3.3 AuditEntry Entity

A new entity for the immutable audit log:

```go
// core/internal/protocol/types.go

// AuditEntry records a security-relevant action.
type AuditEntry struct {
    ID        string         `json:"id"`         // audit_<random>
    Timestamp time.Time      `json:"timestamp"`
    OrgID     string         `json:"org_id"`
    WorkerID  string         `json:"worker_id,omitempty"`  // empty for admin actions
    Action    string         `json:"action"`               // e.g. "worker.register", "task.route", "token.revoke"
    Resource  string         `json:"resource"`             // e.g. "worker:worker_abc123"
    Detail    map[string]any `json:"detail,omitempty"`     // action-specific metadata
    RequestID string         `json:"request_id,omitempty"` // correlation with HTTP request
    Outcome   string         `json:"outcome"`              // "success" | "denied" | "error"
}
```

Actions to audit (comprehensive list):

| Action | Trigger | Outcome Values |
|--------|---------|---------------|
| `token.create` | Admin creates a worker token | success |
| `token.revoke` | Admin revokes a worker token | success |
| `worker.register` | Worker registers with a token | success, denied |
| `worker.heartbeat` | Worker sends heartbeat | success, denied |
| `worker.deregister` | Worker deregisters | success, denied |
| `task.submit` | Admin submits a task | success |
| `task.route` | Router assigns task to worker | success, error |
| `task.complete` | Worker completes a task | success |
| `task.fail` | Worker reports task failure | success |
| `auth.rejected` | Any request with invalid token | denied |

### 3.4 Store Interface Changes

```go
// core/internal/store/store.go -- additions

type Store interface {
    // ... existing methods ...

    // Worker tokens
    AddWorkerToken(t *protocol.WorkerToken) error
    GetWorkerToken(id string) (*protocol.WorkerToken, error)
    GetWorkerTokenByHash(hash string) (*protocol.WorkerToken, error)
    UpdateWorkerToken(t *protocol.WorkerToken) error
    ListWorkerTokensByOrg(orgID string) []*protocol.WorkerToken
    ListWorkerTokensByWorker(workerID string) []*protocol.WorkerToken

    HasAnyWorkerTokens() bool  // fast check for migration mode

    // Audit log
    AppendAudit(e *protocol.AuditEntry) error
    QueryAudit(filter AuditFilter) []*protocol.AuditEntry

    // Org-scoped queries (new overloads)
    ListWorkersByOrg(orgID string) []*protocol.Worker
    ListTasksByOrg(orgID string) []*protocol.Task
    FindWorkersByCapabilityAndOrg(capability, orgID string) []*protocol.Worker
}

// AuditFilter defines query parameters for audit log.
type AuditFilter struct {
    OrgID     string     // required
    WorkerID  string     // optional, filter by specific worker
    Action    string     // optional, filter by action type
    StartTime *time.Time // optional, events after this time
    EndTime   *time.Time // optional, events before this time
    Limit     int        // max results, default 100
    Offset    int        // pagination offset
}
```

### 3.5 In-Memory Store Additions

The `MemoryStore` gets three new maps:

```go
type MemoryStore struct {
    mu          sync.RWMutex
    workers     map[string]*protocol.Worker
    tasks       map[string]*protocol.Task
    workflows   map[string]*protocol.Workflow
    teams       map[string]*protocol.Team
    knowledge   map[string]*protocol.KnowledgeEntry
    tokens      map[string]*protocol.WorkerToken  // NEW: token ID -> token
    tokenIndex  map[string]string                  // NEW: token hash -> token ID (lookup index)
    auditLog    []*protocol.AuditEntry             // NEW: append-only slice
}
```

The `tokenIndex` map enables O(1) lookup by token hash, which is the hot path (every worker request requires a hash lookup). The `auditLog` is an append-only slice -- entries are never deleted or modified.

---

## 4. Token Format and Generation

### 4.1 Token Format

```
mct_<32 bytes hex>
```

- Prefix `mct_` (MagiC Token) makes tokens visually identifiable and grep-able in logs/configs.
- 32 bytes of cryptographic randomness = 256 bits of entropy (same as GitHub personal access tokens).
- Total length: 4 + 64 = 68 characters.

Example: `mct_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2`

### 4.2 Token Generation

```go
// core/internal/protocol/token.go

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

const tokenPrefix = "mct_"
const tokenBytes = 32

// GenerateToken creates a new token value and its SHA-256 hash.
// Returns (rawToken, tokenHash).
func GenerateToken() (string, string) {
    b := make([]byte, tokenBytes)
    if _, err := rand.Read(b); err != nil {
        panic("crypto/rand failed: " + err.Error())
    }
    raw := tokenPrefix + hex.EncodeToString(b)
    return raw, HashToken(raw)
}

// HashToken computes the SHA-256 hash of a raw token string.
func HashToken(raw string) string {
    h := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(h[:])
}
```

Why SHA-256 and not bcrypt:
- Tokens are high-entropy random strings (256 bits), not human-chosen passwords. Brute-force is infeasible regardless of hash speed.
- SHA-256 is fast enough for per-request validation without adding latency.
- This is the same approach used by GitHub, Stripe, and AWS for API tokens.

### 4.3 Token Lifecycle

```
1. Admin calls POST /api/v1/orgs/{orgID}/tokens
   - Server generates token: mct_a1b2c3...
   - Server stores WorkerToken{ID: "token_xyz", OrgID: orgID, TokenHash: SHA256(token), WorkerID: ""}
   - Response includes raw token (ONCE ONLY, never stored or returned again)
   - Audit: token.create

2. Admin gives raw token to worker operator (env var, config file, secret manager)

3. Worker calls POST /api/v1/workers/register with body { "worker_token": "mct_a1b2c3...", ... }
   - Server computes SHA256("mct_a1b2c3...")
   - Server looks up token by hash -> finds WorkerToken
   - Checks: IsValid()? WorkerID empty (unbound)?
   - Binds: sets WorkerToken.WorkerID = new_worker.ID
   - Sets Worker.OrgID = token.OrgID
   - Audit: worker.register (success)

4. Worker calls POST /api/v1/workers/heartbeat with body { "worker_token": "mct_a1b2c3...", "worker_id": "worker_123", ... }
   - Server validates token hash -> finds WorkerToken
   - Checks: IsValid()? WorkerID == "worker_123" (token bound to this worker)?
   - Audit: worker.heartbeat (success)

5. Admin calls DELETE /api/v1/orgs/{orgID}/tokens/{tokenID}
   - Sets WorkerToken.RevokedAt = now
   - Next heartbeat from worker with this token -> 401
   - Audit: token.revoke
```

---

## 5. Authentication Flow

### 5.1 Registration Auth

The `POST /api/v1/workers/register` endpoint currently accepts any request (if `MAGIC_API_KEY` is set, it checks that, but all workers share the same key). The new flow:

```
HTTP Request:
POST /api/v1/workers/register
Content-Type: application/json

{
    "worker_token": "mct_a1b2c3...",       // NEW: required field
    "name": "ContentBot",
    "capabilities": [...],
    "endpoint": {...},
    "limits": {...}
}
```

**RegisterPayload change:**

```go
// core/internal/protocol/messages.go

type RegisterPayload struct {
    WorkerToken  string            `json:"worker_token"`         // NEW: required
    Name         string            `json:"name"`
    Capabilities []Capability      `json:"capabilities"`
    Endpoint     Endpoint          `json:"endpoint"`
    Limits       WorkerLimits      `json:"limits"`
    Metadata     map[string]any    `json:"metadata,omitempty"`
}
```

**Registry.Register changes:**

```go
// core/internal/registry/registry.go

func (r *Registry) Register(p protocol.RegisterPayload) (*protocol.Worker, error) {
    // 1. Validate token
    tokenHash := protocol.HashToken(p.WorkerToken)
    token, err := r.store.GetWorkerTokenByHash(tokenHash)
    if err != nil {
        return nil, fmt.Errorf("invalid worker token")
    }
    if !token.IsValid() {
        return nil, fmt.Errorf("token expired or revoked")
    }
    if token.WorkerID != "" {
        return nil, fmt.Errorf("token already bound to worker %s", token.WorkerID)
    }

    // 2. Create worker with org from token
    w := &protocol.Worker{
        ID:            protocol.GenerateID("worker"),
        Name:          p.Name,
        OrgID:         token.OrgID,    // Set from token
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

    // 3. Bind token to worker
    token.WorkerID = w.ID
    if err := r.store.UpdateWorkerToken(token); err != nil {
        // Rollback worker creation
        r.store.RemoveWorker(w.ID)
        return nil, fmt.Errorf("failed to bind token")
    }

    // 4. Publish event
    r.bus.Publish(events.Event{
        Type:   "worker.registered",
        Source: "registry",
        Payload: map[string]any{
            "worker_id":   w.ID,
            "worker_name": w.Name,
            "org_id":      w.OrgID,
        },
    })

    return w, nil
}
```

### 5.2 Heartbeat Auth

The `POST /api/v1/workers/heartbeat` endpoint must verify that the token belongs to the worker sending the heartbeat (prevents cross-worker impersonation).

**HeartbeatPayload change:**

```go
// core/internal/protocol/messages.go

type HeartbeatPayload struct {
    WorkerToken string `json:"worker_token"`  // NEW: required
    WorkerID    string `json:"worker_id"`
    CurrentLoad int    `json:"current_load"`
    Status      string `json:"status"`
}
```

**Registry.Heartbeat changes:**

```go
func (r *Registry) Heartbeat(p protocol.HeartbeatPayload) error {
    // 1. Validate token
    tokenHash := protocol.HashToken(p.WorkerToken)
    token, err := r.store.GetWorkerTokenByHash(tokenHash)
    if err != nil {
        return fmt.Errorf("invalid worker token")
    }
    if !token.IsValid() {
        return fmt.Errorf("token expired or revoked")
    }

    // 2. Verify token is bound to this worker (prevents impersonation)
    if token.WorkerID != p.WorkerID {
        return fmt.Errorf("token not bound to worker %s", p.WorkerID)
    }

    // 3. Update heartbeat (existing logic)
    w, err := r.store.GetWorker(p.WorkerID)
    if err != nil {
        return err
    }
    w.LastHeartbeat = time.Now()
    w.CurrentLoad = p.CurrentLoad
    if p.Status != "" && w.Status != protocol.StatusPaused {
        w.Status = p.Status
    }
    return r.store.UpdateWorker(w)
}
```

### 5.3 Deregister Auth

The `DELETE /api/v1/workers/{id}` endpoint uses the same token validation pattern. The worker token is passed in the `Authorization` header:

```
DELETE /api/v1/workers/worker_abc123
Authorization: Bearer mct_a1b2c3...
```

The middleware validates: token exists, is valid, and is bound to `worker_abc123`.

### 5.4 Auth Middleware Refactor

The current `authMiddleware` in `middleware.go` becomes a two-tier check:

```go
// core/internal/gateway/middleware.go

// workerAuthMiddleware validates worker tokens for worker-facing endpoints.
// It extracts the token from the Authorization header (Bearer mct_...) and
// validates it against the token store. On success, it injects the validated
// WorkerToken into the request context.
func workerAuthMiddleware(tokenStore TokenLookup, auditFn AuditFunc) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            raw := extractBearerToken(r)
            if raw == "" {
                auditFn(r, "auth.rejected", "", "denied", map[string]any{"reason": "missing token"})
                writeError(w, http.StatusUnauthorized, "worker token required")
                return
            }

            hash := protocol.HashToken(raw)
            token, err := tokenStore.GetWorkerTokenByHash(hash)
            if err != nil || !token.IsValid() {
                auditFn(r, "auth.rejected", "", "denied", map[string]any{"reason": "invalid or revoked token"})
                writeError(w, http.StatusUnauthorized, "invalid or revoked token")
                return
            }

            // Inject token into context for handlers to use
            ctx := context.WithValue(r.Context(), ctxKeyWorkerToken, token)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// extractBearerToken extracts the token from Authorization header.
func extractBearerToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimPrefix(auth, "Bearer ")
    }
    return ""
}

type contextKey string
const ctxKeyWorkerToken contextKey = "worker_token"

// TokenFromContext retrieves the validated WorkerToken from the request context.
func TokenFromContext(ctx context.Context) *protocol.WorkerToken {
    t, _ := ctx.Value(ctxKeyWorkerToken).(*protocol.WorkerToken)
    return t
}
```

**Route grouping in gateway.go:**

```go
func (g *Gateway) Handler() http.Handler {
    mux := http.NewServeMux()

    // Public (no auth)
    mux.HandleFunc("GET /health", g.handleHealth)

    // Admin routes (MAGIC_API_KEY auth -- existing behavior)
    adminMux := http.NewServeMux()
    adminMux.HandleFunc("POST /api/v1/tasks", g.handleSubmitTask)
    adminMux.HandleFunc("GET /api/v1/tasks", g.handleListTasks)
    adminMux.HandleFunc("GET /api/v1/tasks/{id}", g.handleGetTask)
    adminMux.HandleFunc("GET /api/v1/workers", g.handleListWorkers)
    adminMux.HandleFunc("GET /api/v1/workers/{id}", g.handleGetWorker)
    adminMux.HandleFunc("POST /api/v1/teams", g.handleCreateTeam)
    adminMux.HandleFunc("GET /api/v1/teams", g.handleListTeams)
    adminMux.HandleFunc("GET /api/v1/metrics", g.handleGetStats)
    adminMux.HandleFunc("GET /api/v1/costs", g.handleCostReport)
    // ... other admin routes
    // Token management (admin only)
    adminMux.HandleFunc("POST /api/v1/orgs/{orgID}/tokens", g.handleCreateToken)
    adminMux.HandleFunc("GET /api/v1/orgs/{orgID}/tokens", g.handleListTokens)
    adminMux.HandleFunc("DELETE /api/v1/orgs/{orgID}/tokens/{tokenID}", g.handleRevokeToken)
    // Audit log (admin only)
    adminMux.HandleFunc("GET /api/v1/orgs/{orgID}/audit", g.handleQueryAudit)

    // Worker routes (worker token auth)
    workerMux := http.NewServeMux()
    workerMux.HandleFunc("POST /api/v1/workers/register", g.handleRegisterWorker)
    workerMux.HandleFunc("POST /api/v1/workers/heartbeat", g.handleHeartbeat)
    workerMux.HandleFunc("DELETE /api/v1/workers/{id}", g.handleDeregisterWorker)

    // Compose with middleware
    // ...
}
```

### 5.5 Backward Compatibility

The transition from shared `MAGIC_API_KEY` to per-worker tokens must not break existing deployments.

**Dev mode (no `MAGIC_API_KEY` set, no tokens created):** All requests pass through as today. The system works without any tokens. This is important for local development and quick-start experience.

**Migration mode (`MAGIC_API_KEY` set, tokens may or may not exist):** The existing `MAGIC_API_KEY` continues to work for admin endpoints. Worker endpoints accept both the old `MAGIC_API_KEY` and new `worker_token`. If no worker tokens exist in the store, worker endpoints fall back to `MAGIC_API_KEY` validation. Once the first worker token is created, worker endpoints require a worker token (the old key stops working for worker endpoints). A startup log message warns about this transition.

**Full security mode (tokens exist):** Worker endpoints require valid worker tokens. Admin endpoints require `MAGIC_API_KEY`. This is the target state.

```go
func (r *Registry) requiresWorkerToken() bool {
    // If any worker tokens exist, worker endpoints require them.
    // This allows gradual migration: create tokens first, then update workers.
    // Implementation: the Store tracks a simple boolean flag that flips to true
    // on the first AddWorkerToken call (avoids scanning the full token table).
    return r.store.HasAnyWorkerTokens()
}
```

The `HasAnyWorkerTokens() bool` method is added to the Store interface. The in-memory implementation uses a simple boolean flag set to `true` on the first `AddWorkerToken` call. This avoids the cost of listing all tokens on every request.

---

## 6. Org Isolation

### 6.1 Principle

Every data query that returns workers or tasks MUST be scoped to the requesting org. A worker in `org_acme` must never see workers or tasks from `org_beta`.

### 6.2 Enforcement Points

| Operation | Current Behavior | New Behavior |
|-----------|-----------------|--------------|
| `Registry.Register()` | No org assignment | Worker.OrgID set from token.OrgID |
| `Registry.ListWorkers()` | Returns all workers | Returns workers filtered by org |
| `Registry.FindByCapability()` | Returns all matching workers | Returns matching workers in same org |
| `Router.RouteTask()` | Routes to any capable worker | Routes only to workers in same org as task |
| `Store.ListTasks()` | Returns all tasks | Scoped by org when called from org context |

### 6.3 Router Org Isolation

The router is the most critical enforcement point. A task from `org_acme` must never be routed to a worker from `org_beta`.

```go
// core/internal/router/router.go

func (r *Router) RouteTask(task *protocol.Task) (*protocol.Worker, error) {
    orgID := task.Context.OrgID
    if orgID == "" {
        // Tasks without org context use the global pool (dev mode)
        return r.routeGlobal(task)
    }

    // Only consider workers in the same org
    allWorkers := r.store.ListWorkersByOrg(orgID)
    capable := filterByCapability(allWorkers, task.Routing.RequiredCapabilities)

    if len(capable) == 0 {
        return nil, ErrNoWorkerAvailable
    }

    // ... rest of routing logic unchanged
}
```

### 6.4 Task Org Context

When a task is submitted via `POST /api/v1/tasks`, the admin's `MAGIC_API_KEY` does not carry org identity. The submitter must specify `context.org_id` in the task payload. If omitted, the task is org-less (dev mode).

When a task is created as part of a workflow or delegation, the org context is inherited from the originating worker or workflow.

### 6.5 Org ID Handling for Single-Org Deployments

Most MagiC deployments will be single-org. To avoid requiring org configuration for simple use cases:

- If `MAGIC_DEFAULT_ORG` env var is set, all tokens are created under that org unless explicitly specified.
- If no org exists, the system creates a default org `org_default` on first token creation.
- The default org ID is used for task routing when `context.org_id` is empty but tokens are in use.

---

## 7. Audit Log

### 7.1 Design

The audit log is an append-only store of `AuditEntry` records. Entries are never modified or deleted. In the in-memory store, it is a slice that only grows. In SQLite, it is a table with no UPDATE or DELETE operations.

### 7.2 Audit Service

```go
// core/internal/audit/audit.go

package audit

import (
    "time"

    "github.com/kienbui1995/magic/core/internal/events"
    "github.com/kienbui1995/magic/core/internal/protocol"
    "github.com/kienbui1995/magic/core/internal/store"
)

// Logger records security-relevant actions to the audit store.
type Logger struct {
    store store.Store
    bus   *events.Bus
}

func New(s store.Store, bus *events.Bus) *Logger {
    return &Logger{store: s, bus: bus}
}

// Record writes an audit entry and publishes an event.
func (l *Logger) Record(orgID, workerID, action, resource, requestID, outcome string, detail map[string]any) {
    entry := &protocol.AuditEntry{
        ID:        protocol.GenerateID("audit"),
        Timestamp: time.Now(),
        OrgID:     orgID,
        WorkerID:  workerID,
        Action:    action,
        Resource:  resource,
        Detail:    detail,
        RequestID: requestID,
        Outcome:   outcome,
    }

    l.store.AppendAudit(entry)

    l.bus.Publish(events.Event{
        Type:     "audit." + action,
        Source:   "audit",
        Severity: severityForOutcome(outcome),
        Payload: map[string]any{
            "audit_id":  entry.ID,
            "org_id":    orgID,
            "worker_id": workerID,
            "action":    action,
            "outcome":   outcome,
        },
    })
}

func severityForOutcome(outcome string) string {
    switch outcome {
    case "denied":
        return "warn"
    case "error":
        return "error"
    default:
        return "info"
    }
}

// Query returns audit entries matching the filter.
func (l *Logger) Query(filter store.AuditFilter) []*protocol.AuditEntry {
    return l.store.QueryAudit(filter)
}
```

### 7.3 Event Bus Integration

The audit logger subscribes to relevant events from the bus and also records actions directly from the middleware. Two recording paths:

1. **Direct recording** from middleware and handlers -- for auth rejections, token operations.
2. **Event-driven recording** from the bus -- for task lifecycle events that are already published.

The event-driven path subscribes to existing events:

```go
func (l *Logger) SubscribeToEvents() {
    l.bus.Subscribe("worker.registered", func(e events.Event) {
        l.Record(
            strVal(e.Payload, "org_id"),
            strVal(e.Payload, "worker_id"),
            "worker.register",
            "worker:"+strVal(e.Payload, "worker_id"),
            "", "success", e.Payload,
        )
    })

    l.bus.Subscribe("task.routed", func(e events.Event) {
        l.Record(
            "", // org from task context
            strVal(e.Payload, "worker_id"),
            "task.route",
            "task:"+strVal(e.Payload, "task_id"),
            "", "success", e.Payload,
        )
    })

    // ... similar for task.completed, task.failed, worker.deregistered
}
```

### 7.4 Audit Query API

```
GET /api/v1/orgs/{orgID}/audit?worker_id=...&action=...&start=...&end=...&limit=100&offset=0
```

Response:

```json
{
    "entries": [
        {
            "id": "audit_abc123",
            "timestamp": "2026-03-27T10:00:00Z",
            "org_id": "org_acme",
            "worker_id": "worker_001",
            "action": "worker.register",
            "resource": "worker:worker_001",
            "detail": {"worker_name": "ContentBot"},
            "request_id": "req_xyz",
            "outcome": "success"
        }
    ],
    "total": 42,
    "limit": 100,
    "offset": 0
}
```

### 7.5 Audit Export

```
GET /api/v1/orgs/{orgID}/audit/export?format=json&start=...&end=...
```

Returns the full audit log as a JSON array (newline-delimited JSON for large exports). This satisfies the "Should Have" requirement for exportable audit logs.

---

## 8. API Contracts

### 8.1 Token Management APIs

**Create Token:**

```
POST /api/v1/orgs/{orgID}/tokens
Authorization: Bearer <MAGIC_API_KEY>
Content-Type: application/json

{
    "name": "content-bot-prod",
    "expires_in_hours": 0           // 0 = no expiry (default)
}
```

Response (201 Created):

```json
{
    "id": "token_abc123",
    "org_id": "org_acme",
    "name": "content-bot-prod",
    "token": "mct_a1b2c3d4e5f6...",    // Raw token, returned ONCE ONLY
    "expires_at": null,
    "created_at": "2026-03-27T10:00:00Z"
}
```

**List Tokens:**

```
GET /api/v1/orgs/{orgID}/tokens
Authorization: Bearer <MAGIC_API_KEY>
```

Response (200 OK):

```json
[
    {
        "id": "token_abc123",
        "org_id": "org_acme",
        "worker_id": "worker_001",
        "name": "content-bot-prod",
        "expires_at": null,
        "revoked_at": null,
        "created_at": "2026-03-27T10:00:00Z"
    }
]
```

Note: The raw token value is never returned in list/get responses.

**Revoke Token:**

```
DELETE /api/v1/orgs/{orgID}/tokens/{tokenID}
Authorization: Bearer <MAGIC_API_KEY>
```

Response (200 OK):

```json
{
    "status": "revoked",
    "token_id": "token_abc123",
    "revoked_at": "2026-03-27T11:00:00Z"
}
```

Revocation is immediate. The next request using this token will be rejected. There is no caching layer between the store and the middleware -- every request looks up the token from the store directly. The in-memory store lookup is O(1) by hash, so this adds negligible latency.

### 8.2 Token Rotation

Token rotation is a "Should Have" requirement. The flow:

1. Admin creates a new token for the same worker label.
2. Admin configures the worker to use the new token.
3. Worker starts sending heartbeats with the new token.
4. Admin revokes the old token.

**Grace period:** Not implemented in MVP. The old token stops working immediately on revocation. The admin must coordinate the switch. This is acceptable because:
- Workers are not human users who need seamless rotation.
- The operator controls both the token issuance and the worker configuration.
- A 60-second grace period adds complexity (timer goroutines, dual-valid state) that is not justified for MVP.

### 8.3 Updated Registration API

```
POST /api/v1/workers/register
Content-Type: application/json

{
    "worker_token": "mct_a1b2c3...",
    "name": "ContentBot",
    "capabilities": [
        {
            "name": "content_writing",
            "description": "Write blog posts and articles",
            "est_cost_per_call": 0.05
        }
    ],
    "endpoint": {
        "type": "http",
        "url": "http://contentbot:9000/mcp2"
    },
    "limits": {
        "max_concurrent_tasks": 5
    }
}
```

Response (201 Created):

```json
{
    "id": "worker_abc123",
    "name": "ContentBot",
    "org_id": "org_acme",
    "capabilities": [...],
    "endpoint": {...},
    "status": "active",
    "registered_at": "2026-03-27T10:00:00Z"
}
```

Error responses:

| Status | Condition | Body |
|--------|-----------|------|
| 400 | Missing `worker_token` field | `{"error": "worker_token is required"}` |
| 401 | Token not found or invalid hash | `{"error": "invalid worker token"}` |
| 401 | Token expired | `{"error": "token expired"}` |
| 401 | Token revoked | `{"error": "token revoked"}` |
| 409 | Token already bound to another worker | `{"error": "token already in use"}` |

### 8.4 Updated Heartbeat API

```
POST /api/v1/workers/heartbeat
Content-Type: application/json

{
    "worker_token": "mct_a1b2c3...",
    "worker_id": "worker_abc123",
    "current_load": 2,
    "status": "active"
}
```

Error responses:

| Status | Condition | Body |
|--------|-----------|------|
| 401 | Token invalid or revoked | `{"error": "invalid or revoked token"}` |
| 403 | Token bound to different worker | `{"error": "token not authorized for this worker"}` |
| 404 | Worker ID not found | `{"error": "worker not found"}` |

---

## 9. Edge Cases and Error Handling

### 9.1 Token Reuse After Worker Deregister

**Scenario:** Worker registers with token T1, then deregisters. Can another worker register with T1?

**Decision:** No. Once a token is bound to a worker ID, it stays bound even after deregistration. The admin must create a new token for a new worker. This prevents token recycling attacks where a deregistered malicious worker's token is reused.

To register a replacement worker, the admin creates a new token.

### 9.2 Concurrent Registration with Same Token

**Scenario:** Two workers attempt to register with the same unbound token simultaneously.

**Handling:** The store's `UpdateWorkerToken` must use compare-and-swap semantics. Only one registration succeeds; the other gets a 409 Conflict response.

```go
// In MemoryStore:
func (s *MemoryStore) UpdateWorkerToken(t *protocol.WorkerToken) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    existing, ok := s.tokens[t.ID]
    if !ok {
        return ErrNotFound
    }
    // CAS: if token was unbound when we read it but is now bound, reject
    if existing.WorkerID != "" && t.WorkerID != existing.WorkerID {
        return ErrTokenAlreadyBound
    }
    s.tokens[t.ID] = deepCopyWorkerToken(t)
    return nil
}
```

### 9.3 Revoked Token During Active Task

**Scenario:** Worker is executing a task, admin revokes its token.

**Behavior:**
- The task continues to completion (the worker already has the task).
- The worker's next heartbeat will fail with 401.
- The heartbeat failure causes the health checker to mark the worker offline after timeout.
- No active tasks are forcefully interrupted.

This is a deliberate design choice: revoking a token is an authorization action, not a kill signal. The task was authorized when assigned. If immediate task cancellation is needed, that is a separate `task.cancel` operation (future scope).

### 9.4 Token in Request Body vs Header

For `POST /api/v1/workers/register`, the token is in the JSON body (`worker_token` field) because registration is the first interaction and the worker does not yet have an ID.

For `POST /api/v1/workers/heartbeat`, the token is also in the JSON body because the heartbeat payload is already a JSON object and adding it there keeps the API consistent.

For `DELETE /api/v1/workers/{id}`, the token is in the `Authorization: Bearer mct_...` header because DELETE requests conventionally do not have bodies.

Alternative considered: Always use the `Authorization` header. This was rejected because the registration flow is more natural with the token in the body (the worker sends all its data in one JSON object), and it avoids the complexity of parsing the same credential from two different locations.

### 9.5 Dev Mode (No Tokens, No API Key)

When neither `MAGIC_API_KEY` is set nor any worker tokens exist, the system operates in full open mode. This is the default for `magic serve` with no configuration. All auth middleware passes through.

This is critical for the developer experience. The hello-world example must work without any authentication setup:

```bash
magic serve
python examples/hello-worker/main.py  # just works
```

### 9.6 Multiple Tokens per Worker

**Scenario:** Can a worker have multiple valid tokens?

**Decision:** Yes. A worker can be registered with one token, and additional tokens can be created and bound to the same worker. This enables token rotation without downtime.

However, for MVP, a token is bound during registration and there is no API to bind additional tokens to an existing worker. Token rotation is achieved by creating a new token, re-registering the worker with it (which creates a new worker ID), and updating the configuration. Full multi-token support for a single worker is future scope.

### 9.7 Org Mismatch in Delegation

**Scenario:** Worker A (org_acme) delegates a task. The router must only consider workers from org_acme for the delegation, even if org_beta has a worker with the needed capability.

**Handling:** The delegation flow inherits `context.org_id` from the delegating worker. The router enforces org isolation as described in section 6.3. No cross-org delegation is possible.

### 9.8 Clock Skew and Token Expiry

Token expiry is checked on the server side using the server's clock. There is no tolerance window for clock skew because:
- Both the check and the expiry timestamp are set by the same server.
- Workers do not set or influence the expiry time.
- If the server's clock is wrong, all timestamps are wrong -- this is a deployment problem, not a protocol problem.

---

## 10. Testing Strategy

### 10.1 Test Categories

| Category | What | How | Files |
|----------|------|-----|-------|
| Unit: Token | Generation, hashing, validation | Pure functions, no I/O | `core/internal/protocol/token_test.go` |
| Unit: Store | Token CRUD, audit append/query, org-filtered queries | MemoryStore, isolated | `core/internal/store/memory_test.go` (extend) |
| Unit: Audit | Record, query, filtering | Audit logger + MemoryStore | `core/internal/audit/audit_test.go` |
| Integration: Auth | Full request flow through middleware | httptest.Server + Gateway | `core/internal/gateway/gateway_test.go` (extend) |
| Integration: Org isolation | Cross-org queries return empty | Full stack with 2 orgs | `core/internal/gateway/security_test.go` |
| Integration: Registration | Token binding, duplicate rejection | Registry + Store | `core/internal/registry/registry_test.go` (extend) |

### 10.2 Key Test Cases

**Token validation tests (`protocol/token_test.go`):**

```go
func TestGenerateToken_Format(t *testing.T)              // starts with "mct_", length 68
func TestGenerateToken_Unique(t *testing.T)               // two calls produce different tokens
func TestHashToken_Deterministic(t *testing.T)            // same input -> same hash
func TestHashToken_DifferentFromInput(t *testing.T)       // hash != raw token
func TestWorkerToken_IsValid_Active(t *testing.T)         // not expired, not revoked -> true
func TestWorkerToken_IsValid_Expired(t *testing.T)        // expired -> false
func TestWorkerToken_IsValid_Revoked(t *testing.T)        // revoked -> false
```

**Registration auth tests (`registry/registry_test.go`):**

```go
func TestRegister_ValidToken(t *testing.T)                // success, worker gets org from token
func TestRegister_InvalidToken(t *testing.T)              // 401
func TestRegister_ExpiredToken(t *testing.T)               // 401
func TestRegister_RevokedToken(t *testing.T)               // 401
func TestRegister_AlreadyBoundToken(t *testing.T)          // 409, token used by another worker
func TestRegister_SetsOrgID(t *testing.T)                  // worker.OrgID == token.OrgID
```

**Heartbeat auth tests (`registry/registry_test.go`):**

```go
func TestHeartbeat_ValidToken(t *testing.T)               // success
func TestHeartbeat_WrongWorkerID(t *testing.T)             // 403, token bound to different worker
func TestHeartbeat_RevokedToken(t *testing.T)              // 401
```

**Org isolation tests (`gateway/security_test.go`):**

```go
func TestOrgIsolation_WorkerListFiltered(t *testing.T)    // org A workers not visible to org B
func TestOrgIsolation_TaskRouting(t *testing.T)            // task from org A never routed to org B worker
func TestOrgIsolation_TaskListFiltered(t *testing.T)       // org A tasks not visible to org B
```

**Audit log tests (`audit/audit_test.go`):**

```go
func TestAudit_RecordsRegistration(t *testing.T)          // worker.register creates audit entry
func TestAudit_RecordsAuthRejection(t *testing.T)         // auth failure creates audit entry
func TestAudit_RecordsTokenRevocation(t *testing.T)       // token.revoke creates audit entry
func TestAudit_QueryByWorker(t *testing.T)                // filter by worker_id
func TestAudit_QueryByTimeRange(t *testing.T)             // filter by start/end time
func TestAudit_QueryByAction(t *testing.T)                // filter by action type
func TestAudit_AppendOnly(t *testing.T)                   // no update/delete operations possible
```

**Integration tests (`gateway/gateway_test.go`):**

```go
func TestFullFlow_CreateToken_Register_Heartbeat(t *testing.T)  // happy path end-to-end
func TestFullFlow_RevokeToken_HeartbeatFails(t *testing.T)      // revocation immediately effective
func TestFullFlow_DevMode_NoAuth(t *testing.T)                   // no tokens, no API key -> all passes
func TestFullFlow_BackwardCompat_APIKeyOnly(t *testing.T)        // MAGIC_API_KEY still works for admin
```

### 10.3 Test Fixtures

```go
// test helper: create an org + token + registered worker
func setupTestOrgWithWorker(t *testing.T, s store.Store) (orgID, workerID, rawToken string) {
    orgID = "org_test"
    raw, hash := protocol.GenerateToken()
    token := &protocol.WorkerToken{
        ID:        protocol.GenerateID("token"),
        OrgID:     orgID,
        TokenHash: hash,
        Name:      "test-token",
        CreatedAt: time.Now(),
    }
    s.AddWorkerToken(token)

    // Registration would normally bind the token; simulate it:
    workerID = protocol.GenerateID("worker")
    worker := &protocol.Worker{
        ID:    workerID,
        Name:  "TestBot",
        OrgID: orgID,
        // ...
    }
    s.AddWorker(worker)

    token.WorkerID = workerID
    s.UpdateWorkerToken(token)

    return orgID, workerID, raw
}
```

---

## 11. Migration Path from Current Auth

### 11.1 Current State

- `MAGIC_API_KEY` env var checked in `authMiddleware` (middleware.go:12-39)
- Single shared key for all endpoints and all workers
- No org concept enforced
- Workers registered without any identity binding

### 11.2 Migration Steps

**Step 1: Add token infrastructure (non-breaking).**
- Add `WorkerToken`, `AuditEntry` types to protocol.
- Add token/audit methods to Store interface and MemoryStore.
- Add `audit` package.
- Add token management API endpoints (POST/GET/DELETE).
- All existing functionality continues to work unchanged.

**Step 2: Add `OrgID` to Worker (non-breaking).**
- Add `OrgID` field to `Worker` struct (optional, `omitempty`).
- Existing workers get empty `OrgID` (no org = global pool).
- Router treats empty org as "match any" (backward compatible).

**Step 3: Update Registry to support token auth (non-breaking).**
- `Register()` checks for `worker_token` in payload.
- If present: validate token, bind worker, set OrgID.
- If absent AND no tokens exist in store: allow (dev mode).
- If absent AND tokens exist: reject (security mode).

**Step 4: Update heartbeat to support token auth (non-breaking in dev mode).**
- Same pattern as Step 3.

**Step 5: Refactor auth middleware (non-breaking).**
- Split into admin auth (existing `MAGIC_API_KEY`) and worker auth (new token validation).
- Dev mode (no key, no tokens) continues to work.

**Step 6: Add org-scoped store queries and router isolation.**
- Add `ListWorkersByOrg`, `FindWorkersByCapabilityAndOrg` to Store.
- Router uses org-scoped queries when `context.org_id` is present.

### 11.3 Python SDK Changes

The Python SDK `MagiCClient` and `Worker` classes must be updated to pass the worker token:

```python
class Worker:
    def __init__(self, name: str, endpoint: str, worker_token: str = ""):
        self._worker_token = worker_token
        # ...

    def register(self, magic_url: str):
        payload = {
            "worker_token": self._worker_token,  # NEW
            "name": self.name,
            # ...
        }
```

If `worker_token` is empty string, the SDK sends the payload without it (dev mode / backward compatibility).

---

## 12. Open Questions

### Resolved During Design

| Question | Resolution |
|----------|-----------|
| Token in header vs body? | Body for register/heartbeat (JSON consistency), header for DELETE |
| Grace period for token rotation? | Not in MVP. Immediate revocation. Admin coordinates switch. |
| Can tokens be recycled after deregister? | No. One-time bind. Create new token for new worker. |
| How does dev mode work? | No tokens + no API key = full open. First token creation activates security. |

### Still Open (Decide During Implementation)

1. **SQLite store for tokens:** The `SQLiteStore` in `core/internal/store/sqlite.go` needs the token and audit tables. Schema:

```sql
CREATE TABLE IF NOT EXISTS worker_tokens (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL,
    worker_id TEXT DEFAULT '',
    token_hash TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    expires_at TEXT,
    revoked_at TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_tokens_hash ON worker_tokens(token_hash);
CREATE INDEX idx_tokens_org ON worker_tokens(org_id);

CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    timestamp TEXT NOT NULL,
    org_id TEXT NOT NULL,
    worker_id TEXT DEFAULT '',
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    detail TEXT,          -- JSON
    request_id TEXT,
    outcome TEXT NOT NULL
);
CREATE INDEX idx_audit_org ON audit_log(org_id);
CREATE INDEX idx_audit_worker ON audit_log(worker_id);
CREATE INDEX idx_audit_action ON audit_log(action);
CREATE INDEX idx_audit_time ON audit_log(timestamp);
```

2. **Audit log retention:** The in-memory audit log grows unbounded. For MVP this is acceptable (same as in-memory worker/task storage). When SQLite is used, should there be a retention policy? Suggestion: no automatic deletion for MVP. Add `MAGIC_AUDIT_RETENTION_DAYS` env var as future enhancement.

3. **Go SDK token support:** The Go SDK (`sdk/go/`) needs the same `worker_token` parameter. The changes mirror the Python SDK.

4. **Dashboard auth:** The dashboard endpoint (`GET /dashboard`) currently skips auth. With token-based security, the dashboard should require `MAGIC_API_KEY` via query parameter (as it does now with `?key=`). No change needed.

5. **Performance at scale:** Token lookup is O(1) hash map lookup per request. Audit log append is O(1). Neither adds meaningful latency. If audit log queries become slow at millions of entries, add pagination limits and time-based partitioning. Not a concern for MVP.

---

## Appendix: File Change Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `core/internal/protocol/types.go` | Modify | Add `OrgID` to Worker, add WorkerToken and AuditEntry types |
| `core/internal/protocol/token.go` | Create | GenerateToken, HashToken functions |
| `core/internal/protocol/token_test.go` | Create | Token generation and validation tests |
| `core/internal/protocol/messages.go` | Modify | Add `WorkerToken` field to RegisterPayload, HeartbeatPayload |
| `core/internal/store/store.go` | Modify | Add token, audit, org-scoped query methods to Store interface |
| `core/internal/store/memory.go` | Modify | Implement new Store methods (tokens, audit, org-scoped queries) |
| `core/internal/store/memory_test.go` | Modify | Tests for new store methods |
| `core/internal/store/sqlite.go` | Modify | Add token and audit tables, implement new Store methods |
| `core/internal/audit/audit.go` | Create | Audit logger service |
| `core/internal/audit/audit_test.go` | Create | Audit logger tests |
| `core/internal/registry/registry.go` | Modify | Token validation in Register and Heartbeat |
| `core/internal/registry/registry_test.go` | Modify | Auth tests for registration and heartbeat |
| `core/internal/router/router.go` | Modify | Org-scoped worker filtering |
| `core/internal/router/router_test.go` | Modify | Org isolation routing tests |
| `core/internal/gateway/middleware.go` | Modify | Split into admin auth + worker auth middleware |
| `core/internal/gateway/handlers.go` | Modify | Add token management and audit query handlers |
| `core/internal/gateway/gateway.go` | Modify | Route grouping with per-group middleware |
| `core/internal/gateway/gateway_test.go` | Modify | Integration tests for new auth flow |
| `core/internal/gateway/security_test.go` | Create | Org isolation integration tests |
| `core/cmd/magic/main.go` | Modify | Wire up audit logger |
| `sdk/python/magic_ai_sdk/worker.py` | Modify | Add worker_token parameter |
| `sdk/python/magic_ai_sdk/client.py` | Modify | Pass worker_token in register/heartbeat |
| `sdk/python/tests/test_client.py` | Modify | Token auth tests |
