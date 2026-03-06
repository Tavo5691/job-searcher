# Job Searcher — Design Document

## Overview

A terminal UI application for developers actively looking for a new job. It helps track job-hunting periods, build a profile from a resume, manage applications with individual hiring stages, and leverage an LLM to generate actionable insights as new information comes in.

---

## Domain Language

| Term | Definition |
|---|---|
| **Hunt** | A job-searching period. Container for a Profile and all Applications. One active Hunt at a time (MVP). |
| **Profile** | Experience, education, and skills data scoped to a Hunt. Built from a PDF resume (LLM-interpreted) and/or manual input. |
| **Application** | One company/role being pursued within a Hunt. |
| **Stage** | A single step in an Application's hiring process (e.g. recruiter screen, technical interview). |
| **Insight** | LLM-generated structured advice for an Application, updated as new information arrives. |

---

## MVP Scope

- Create, view, and close a Hunt
- Build a Profile by uploading a PDF resume (parsed locally, interpreted by LLM) or via manual input
- Create and manage Applications within a Hunt
- Add ordered Stages to each Application with type, notes, feedback, and outcome
- View Insights for an Application, scoped per Application (Hunt-level Insights deferred to v1.3)
  - Insight context includes both completed Stages (feedback, outcome) and upcoming Stages (pending, future date)
  - Regenerated automatically when Application or Stage data changes; manual refresh also available in TUI
  - Scheduled/time-based regeneration deferred (no background daemon in MVP)
- All interactions via a terminal UI (TUI)
- LLM provider configured via `.env` file (never stored by the app)
- App validates LLM provider config at startup; fails fast with a clear error if provider or API key is missing/invalid before any LLM operation is attempted

---

## Tech Stack

| Concern | Choice |
|---|---|
| Language | Go |
| TUI | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Storage | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| PDF parsing | `pdfcpu` — local extraction to plain text |
| LLM providers | Anthropic Claude, Google Gemini (abstracted behind a common interface) |
| Config / secrets | `.env` file, read at startup, never written by the app |

---

## Data Models

### Hunt
```
id          string (uuid)
title       string
status      enum: active | closed
created_at  timestamp
closed_at   timestamp (nullable)
```

### Profile
```
id               string (uuid)
hunt_id          string (fk)
name             string
summary          string
skills           []string
experience       []Experience
education        []Education
raw_resume_text  string  -- retained for re-prompting if needed
updated_at       timestamp
```

**Experience entry:**
```
company    string
role       string
start      string (month/year)
end        string (month/year, nullable)
notes      string
```

**Education entry:**
```
institution  string
degree       string
field        string
year         string
```

### Application
```
id               string (uuid)
hunt_id          string (fk)
company_name     string
role_title       string
job_description  string
status           enum: applied | interviewing | offer | accepted | rejected | withdrawn
applied_at       timestamp
updated_at       timestamp
notes            string
```

### Stage
```
id              string (uuid)
application_id  string (fk)
type            enum: recruiter_screen | technical_screen | take_home | technical_interview | behavioral | system_design | offer | other
label           string  -- used when type is "other"
date            date (nullable)
notes           string
feedback        string
outcome         enum: pending | passed | failed
order           int
```

### Insight
```
id              string (uuid)
application_id  string (fk)
content         string  -- structured markdown advice
generated_at    timestamp
```

> Insight is regenerated whenever an Application or any of its Stages is updated. The previous Insight is overwritten (one Insight per Application for MVP).

---

## LLM Integration

### Provider Interface
```go
type LLMProvider interface {
    Complete(ctx context.Context, prompt string) (string, error)
}
```
Implementations: `ClaudeProvider`, `GeminiProvider`. Selected via `.env`:
```
LLM_PROVIDER=claude   # or: gemini
ANTHROPIC_API_KEY=...
GEMINI_API_KEY=...
```

> **Security note (future):** When the app matures for distribution, migrate to OS keychain integration (e.g. `99designs/keyring`) instead of `.env` files.

### Resume Parsing Prompt
Sends extracted PDF text to the LLM and asks it to return structured JSON matching the Profile schema. Result is parsed and stored; raw text is retained.

### Insight Generation Prompt
Context sent to LLM:
- Profile summary, skills, experience
- Application: company, role, job description, notes
- Completed Stages: type, notes, feedback, outcome
- Upcoming Stages: type, scheduled date, notes (outcome: pending, date in the future)

Output: structured markdown with sections such as:
- Current status summary
- Preparation recommendations (prioritized around any upcoming stages and their types)
- Areas to address (based on feedback received from completed stages)
- Overall assessment

---

## TUI Structure

```
Hunt List View
  └── Hunt Detail View
        ├── Profile View / Edit
        └── Application List View
              └── Application Detail View
                    ├── Stage List / Edit
                    └── Insight View
```

Navigation: arrow keys / vim keys (`j`/`k`), `Enter` to drill in, `Esc` to go back, `?` for help, `q` to quit.

---

## Project Structure

Follows a pragmatic **ports-and-adapters (hexagonal)** architecture. The domain is pure Go with no external dependencies. Ports are interfaces defined where they are consumed (idiomatic Go). Adapters implement those interfaces and are injected at startup.

```
job-searcher/
├── main.go             -- wires adapters, validates config, starts TUI
├── .env.example
├── Makefile            -- run / build / test-race / vet / verify targets
└── internal/
    ├── domain/         -- pure structs and business rules (Hunt, Profile, Application, Stage, Insight)
    │                      no database, no LLM, no I/O dependencies
    ├── port/           -- interfaces (Store, LLMProvider, PDFParser)
    │                      defined here, consumed by app/ layer
    ├── app/            -- use cases / application service layer
    │                      orchestrates domain + ports (e.g. CreateHunt, CreateApplication)
    │                      this is the only layer the TUI calls
    ├── store/          -- SQLite adapter implementing port.Store
    │                      SQLite setup and migrations live here; driver name: "sqlite"
    ├── llm/            -- Claude and Gemini adapters implementing port.LLMProvider
    ├── pdf/            -- pdfcpu adapter implementing port.PDFParser
    └── tui/            -- Bubble Tea views and state; calls app/ layer only
```

Key principle: **accept interfaces, return structs** (idiomatic Go). Each adapter is independently testable by swapping the real dependency for an in-memory or stub implementation.

---

## Iteration Roadmap

### v1.1 — Polish & Intelligence
- Confirmation dialogs for destructive actions (close Hunt, delete Application)
- Input validation and error messages in TUI
- `.env` not found: guided setup flow on first run
- Export Insights as markdown file
- **LLM-assisted job description analysis** (explicit trigger from Application view): compares JD against Profile, surfaces fit assessment, skill gaps, and suggested talking points

### v1.2 — Profile Enhancements
- Manual skill/experience/education CRUD in TUI
- Re-parse resume with updated prompt without losing manual edits
- `source` field on Profile entries: `resume | manual` (prepares for GitHub)

### v1.3 — Hunt-level Intelligence
- **Hunt Insight**: LLM-generated strategic analysis across all Applications in a Hunt (pattern detection: repeated rejections at same stage type, common feedback themes)
- Track which companies appear across multiple Hunts (read-only cross-hunt view)
- Stage templates for common hiring flows (FAANG, startup, agency)
- LLM suggests likely next Stage based on Application history

### v2.0 — Multiple Hunts & GitHub
- Multiple simultaneous active Hunts
- GitHub integration: connect account or individual repos, extract languages/frameworks/technologies, surface as Profile skills with `source: github`
- OS keychain support for API keys (replaces `.env` for production use)

### v2.x — Extended Reach
- CLI flag shortcuts (`--new`, `--list`) alongside TUI for scriptability
- Reminder/follow-up tracking per Application
- Resume tailoring suggestions per Application (LLM rewrites profile summary to target a specific JD)
