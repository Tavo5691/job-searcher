# job-searcher

A terminal UI application for developers actively job-hunting. Track job-search periods (Hunts), build a profile from a PDF resume, manage applications with individual hiring stages, and generate LLM-powered insights for each application.

## Tech Stack

| Concern | Choice |
|---------|--------|
| Language | Go (module: `github.com/Tavo5691/job-searcher`) |
| TUI | Bubble Tea + Lip Gloss |
| Storage | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| PDF parsing | pdfcpu |
| LLM providers | Anthropic Claude, Google Gemini |
| Config | `.env` file, read at startup |

## Project Structure

```
job-searcher/
├── main.go                  # wires adapters, validates config, starts TUI
├── .env.example             # environment variable template
├── Makefile                 # build/test/vet targets
└── internal/
    ├── domain/              # pure Go types: Hunt, Profile, Application, Stage, Insight
    ├── port/                # interfaces: Store, LLMProvider, PDFParser
    ├── app/                 # application service — all use cases live here
    ├── store/               # SQLite adapter (modernc.org/sqlite, driver: "sqlite")
    ├── llm/                 # Claude and Gemini adapters
    ├── pdf/                 # pdfcpu adapter for PDF text extraction
    └── tui/                 # Bubble Tea TUI views and state
```

## Setup

```bash
cp .env.example .env
# Edit .env: set LLM_PROVIDER, the matching API key, and DB_PATH
```

`.env.example`:
```
LLM_PROVIDER=claude        # or: gemini
ANTHROPIC_API_KEY=...      # required when LLM_PROVIDER=claude
GEMINI_API_KEY=...         # required when LLM_PROVIDER=gemini
DB_PATH=job-searcher.db
```

## Build & Run

```bash
# Build
go build ./...

# Run
go run .

# Via Makefile
make build
make run     # (same as go run .)
```

## Testing

```bash
go test ./...            # run all tests
go test -race ./...      # with race detector (required before merge)
go vet ./...             # static analysis

make verify              # full gate: build + test-race + vet
```

## Architecture

Ports-and-adapters (hexagonal). The `app/` layer is the only entry point for the TUI — it orchestrates domain logic and port interfaces. Adapters (`store/`, `llm/`, `pdf/`) are injected at startup in `main.go`.

See [DESIGN.md](DESIGN.md) for the full domain model, data schemas, LLM integration design, and roadmap.
See [AGENTS.md](AGENTS.md) for coding conventions and AI assistant guidelines.
