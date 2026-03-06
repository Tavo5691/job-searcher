# AGENTS.md — Job Searcher

This file provides conventions and constraints for AI coding assistants working on this project.
Always read `DESIGN.md` first for full context on domain, architecture, and scope.

---

## Commands

```bash
make run                 # Load .env and run the app (go run .)
go build ./...           # Build all packages
go test ./...            # Run all tests
go test -race ./...      # Run with race detector (required before any merge)
go vet ./...             # Static analysis
gofmt -w .               # Format all files
go mod tidy              # Clean up go.mod / go.sum
```

---

## Architecture

This project follows a **ports-and-adapters (hexagonal)** architecture. Respect layer boundaries strictly.

```
main.go
└── internal/
    ├── domain/    pure structs and business rules — NO external imports, NO I/O
    ├── port/      interfaces defined here, consumed by app/ (Store, LLMProvider, PDFParser)
    ├── app/       use cases / application service — orchestrates domain + ports
    │              the ONLY layer the TUI is allowed to call
    ├── store/     SQLite adapter implementing port.Store
    ├── llm/       Claude and Gemini adapters implementing port.LLMProvider
    ├── pdf/       pdfcpu adapter implementing port.PDFParser
    └── tui/       Bubble Tea views and state; calls app/ only
```

### Layer import rules

| Layer | May import | Must NOT import |
|---|---|---|
| `domain/` | stdlib only | anything in `internal/` |
| `port/` | `domain/`, stdlib | adapters (`store/`, `llm/`, `pdf/`) |
| `app/` | `domain/`, `port/` | adapters directly |
| `store/`, `llm/`, `pdf/` | `domain/`, `port/`, stdlib, their own deps | each other, `app/`, `tui/` |
| `tui/` | `app/`, `domain/` | `store/`, `llm/`, `pdf/`, `port/` directly |
| `main.go` | all of the above | — |

**Never bypass the app/ layer from the TUI.** All business logic lives in `app/`.

---

## Domain Language

| Term | Definition |
|---|---|
| **Hunt** | A job-searching period. Container for a Profile and all Applications. |
| **Profile** | Experience, education, and skills data scoped to a Hunt. |
| **Application** | One company/role being pursued within a Hunt. |
| **Stage** | A single step in an Application's hiring process. |
| **Insight** | LLM-generated structured advice for an Application. |

Use these terms exactly in code: type names, variable names, function names, SQL table names.

---

## Go Conventions

### Error handling
- Always wrap errors with context: `fmt.Errorf("create application: %w", err)`
- Return errors as the last return value; never ignore them with `_`
- Use sentinel errors (`var ErrNotFound = errors.New(...)`) when callers need to check identity
- No `panic` except for unrecoverable programmer errors (never on user input or external data)

### Interfaces
- Define interfaces at the **consumer** (idiomatic Go), not next to implementations
- Exception: `port/` package houses shared interfaces used across layers
- Keep interfaces small; prefer one-method interfaces where possible
- **Accept interfaces, return concrete types**

### Context
- `context.Context` is always the **first argument** of any function that does I/O or calls an LLM
- Never store `Context` in a struct field

### Naming
- Package names: short, lowercase, no underscores (e.g. `store`, `llm`, `domain`)
- Avoid stutter: `domain.Hunt`, not `domain.HuntDomain`
- Test files: `_test.go` suffix, table-driven tests with `t.Run` subtests

### No CGO
- Use `modernc.org/sqlite` (pure Go). Do not use `mattn/go-sqlite3` or any CGO-based SQLite driver.
- SQLite driver name is `"sqlite"` (not `"sqlite3"`): `sql.Open("sqlite", dsn)`.
- `store.New(dsn string)` takes only a DSN — no `context.Context` in the constructor.
- `db.SetMaxOpenConns(1)` is set on the pool (SQLite is single-writer).
- Compound fields (`skills`, `experience`, `education`) are stored as JSON text columns.

---

## Configuration & Secrets

- Config is read from `.env` at startup via environment variables
- The app **never writes** to `.env`
- API keys are never logged, stored in the database, or embedded in prompts
- Validate LLM provider config at startup; fail fast with a clear error before any LLM call

---

## LLM Provider Interface

```go
type LLMProvider interface {
    Complete(ctx context.Context, prompt string) (string, error)
}
```

Both `ClaudeProvider` and `GeminiProvider` implement this. Selection is driven by `LLM_PROVIDER` env var.
Never call a provider directly from outside `app/` — always go through the interface.

---

## What NOT to Do

- Do not add direct SQLite calls in `app/`, `domain/`, or `tui/`
- Do not call LLM providers directly from `tui/`
- Do not store secrets in any struct, log, or database field
- Do not use `init()` for side effects — prefer explicit initialization in `main.go`
- Do not add background goroutines or daemons (scheduled Insight regeneration is deferred to post-MVP)
- Do not create a global state or singleton pattern — wire dependencies explicitly in `main.go`

---

## SDD Rules

Artifact store: `engram` (no openspec). When invoking any SDD sub-agent, pass these settings explicitly — do not rely on auto-detection.

| Setting | Value |
|---|---|
| `tdd` | `true` |
| `test_command` | `go test ./...` |
| `build_command` | `go build ./...` |
| `verify_command` | `go test -race ./... && go vet ./...` |

### TDD Workflow (mandatory)

Every task follows RED → GREEN → REFACTOR:

1. **RED** — write a failing test first; run it and confirm it fails
2. **GREEN** — write the minimum code to make it pass; run and confirm
3. **REFACTOR** — clean up without changing behavior; run again to confirm

Tests use table-driven style with `t.Run` subtests (see Go Conventions above).
Run only the relevant package during a task cycle (`go test ./internal/store/...`), not the full suite, for speed.

### Verification Gate

Before any change is considered done:

```bash
go build ./...          # must exit 0
go test -race ./...     # must exit 0
go vet ./...            # must exit 0
```

`sdd-verify` maps every spec scenario to a test result. A scenario is only COMPLIANT when a passing test proves the behavior at runtime — code existing in the codebase is not sufficient evidence.
