# Contributing to MagiC

Thank you for your interest in contributing! MagiC is early-stage — your contributions shape the framework.

## Prerequisites

- Go 1.24+
- Python 3.11+ (for SDK)

## Getting Started

```bash
git clone https://github.com/kienbui1995/magic.git
cd magic

# Build and test Go
cd core && go build ./cmd/magic && go test ./... -v -race

# Python SDK
cd sdk/python && python -m venv .venv
.venv/bin/pip install -e ".[dev]" && .venv/bin/pytest tests/ -v
```

## Good First Issues

New contributor? Look for issues labeled [`good first issue`](https://github.com/kienbui1995/magic/labels/good%20first%20issue).

These are scoped, well-defined tasks with clear acceptance criteria:
- Small bug fixes
- Documentation improvements
- Adding tests to existing code
- Example worker scripts

Comment on the issue to claim it — we'll confirm and unblock you.

## Development Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Make your changes with tests
4. Run all checks:
   ```bash
   cd core && go test ./... -race && go vet ./...
   cd sdk/python && .venv/bin/pytest tests/ -v
   ```
5. Commit with conventional commits: `feat(module): description`
6. Push and open a Pull Request against `main`

## Commit Convention

```
feat(scope): add new feature
fix(scope): fix a bug
docs: update documentation
chore: maintenance tasks
test: add or update tests
refactor(scope): restructure without changing behavior
```

## Code Style

- **Go:** `gofmt`, godoc comments on exported types
- **Python:** PEP 8, type hints required
- All tests must pass including `go test -race`

## What to Work On

Check the [roadmap in README](README.md#roadmap) for planned features. High-impact areas:
- **Go SDK** — native Go workers (currently Python only)
- **Persistent storage** — SQLite/PostgreSQL backend
- **WebSocket transport** — real-time worker connections
- **Dashboard** — web UI for monitoring workers and tasks

## Reporting Issues

Use GitHub Issues with:
- Steps to reproduce
- Expected vs actual behavior
- Go/Python version and OS
- Server logs (if applicable)

## Questions?

Open a [GitHub Discussion](https://github.com/kienbui1995/magic/discussions) — no question is too small.
