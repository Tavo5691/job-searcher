// Package store implements the port.Store interface using SQLite (modernc.org/sqlite).
// Driver name: "sqlite" (pure Go, no CGO).
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // register the "sqlite" driver

	"github.com/Tavo5691/job-searcher/internal/domain"
)

// SQLiteStore implements port.Store using a local SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// New opens a SQLite database at dsn and runs schema migrations.
// Use ":memory:" for an in-process test database.
func New(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite WAL is single-writer; keep connection pool minimal
	s := &SQLiteStore{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// migrate creates all tables if they don't already exist.
func (s *SQLiteStore) migrate(_ context.Context) error {
	const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS hunts (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    status     TEXT NOT NULL,
    created_at TEXT NOT NULL,
    closed_at  TEXT
);

CREATE TABLE IF NOT EXISTS profiles (
    id               TEXT PRIMARY KEY,
    hunt_id          TEXT NOT NULL,
    name             TEXT NOT NULL DEFAULT '',
    summary          TEXT NOT NULL DEFAULT '',
    skills           TEXT NOT NULL DEFAULT '[]',
    experience       TEXT NOT NULL DEFAULT '[]',
    education        TEXT NOT NULL DEFAULT '[]',
    raw_resume_text  TEXT NOT NULL DEFAULT '',
    updated_at       TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS applications (
    id               TEXT PRIMARY KEY,
    hunt_id          TEXT NOT NULL,
    company_name     TEXT NOT NULL,
    role_title       TEXT NOT NULL,
    job_description  TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL,
    applied_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    notes            TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS stages (
    id             TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    type           TEXT NOT NULL,
    label          TEXT NOT NULL DEFAULT '',
    date           TEXT,
    notes          TEXT NOT NULL DEFAULT '',
    feedback       TEXT NOT NULL DEFAULT '',
    outcome        TEXT NOT NULL,
    sort_order     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS insights (
    id             TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    content        TEXT NOT NULL DEFAULT '',
    generated_at   TEXT NOT NULL
);
`
	_, err := s.db.Exec(schema)
	return err
}

// ── Hunt ─────────────────────────────────────────────────────────────────────

func (s *SQLiteStore) SaveHunt(ctx context.Context, h domain.Hunt) error {
	var closedAt *string
	if h.ClosedAt != nil {
		v := h.ClosedAt.UTC().Format(time.RFC3339)
		closedAt = &v
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO hunts (id, title, status, created_at, closed_at) VALUES (?,?,?,?,?)`,
		h.ID, h.Title, string(h.Status), h.CreatedAt.UTC().Format(time.RFC3339), closedAt,
	)
	if err != nil {
		return fmt.Errorf("save hunt: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetHunt(ctx context.Context, id string) (domain.Hunt, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, title, status, created_at, closed_at FROM hunts WHERE id=?`, id)
	return scanHunt(row)
}

func (s *SQLiteStore) ListHunts(ctx context.Context) ([]domain.Hunt, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, status, created_at, closed_at FROM hunts ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list hunts: %w", err)
	}
	defer rows.Close()

	var hunts []domain.Hunt
	for rows.Next() {
		h, err := scanHunt(rows)
		if err != nil {
			return nil, err
		}
		hunts = append(hunts, h)
	}
	return hunts, rows.Err()
}

func (s *SQLiteStore) UpdateHunt(ctx context.Context, h domain.Hunt) error {
	var closedAt *string
	if h.ClosedAt != nil {
		v := h.ClosedAt.UTC().Format(time.RFC3339)
		closedAt = &v
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE hunts SET title=?, status=?, closed_at=? WHERE id=?`,
		h.Title, string(h.Status), closedAt, h.ID,
	)
	if err != nil {
		return fmt.Errorf("update hunt: %w", err)
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanHunt(s scanner) (domain.Hunt, error) {
	var h domain.Hunt
	var createdAt string
	var closedAt *string
	if err := s.Scan(&h.ID, &h.Title, (*string)(&h.Status), &createdAt, &closedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Hunt{}, domain.ErrNotFound
		}
		return domain.Hunt{}, fmt.Errorf("scan hunt: %w", err)
	}
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.Hunt{}, fmt.Errorf("parse hunt created_at: %w", err)
	}
	h.CreatedAt = t
	if closedAt != nil {
		tc, err := time.Parse(time.RFC3339, *closedAt)
		if err != nil {
			return domain.Hunt{}, fmt.Errorf("parse hunt closed_at: %w", err)
		}
		h.ClosedAt = &tc
	}
	return h, nil
}

// ── Profile ───────────────────────────────────────────────────────────────────

func (s *SQLiteStore) SaveProfile(ctx context.Context, p domain.Profile) error {
	skills, err := json.Marshal(p.Skills)
	if err != nil {
		return fmt.Errorf("save profile: marshal skills: %w", err)
	}
	exp, err := json.Marshal(p.Experience)
	if err != nil {
		return fmt.Errorf("save profile: marshal experience: %w", err)
	}
	edu, err := json.Marshal(p.Education)
	if err != nil {
		return fmt.Errorf("save profile: marshal education: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO profiles
         (id, hunt_id, name, summary, skills, experience, education, raw_resume_text, updated_at)
         VALUES (?,?,?,?,?,?,?,?,?)`,
		p.ID, p.HuntID, p.Name, p.Summary,
		string(skills), string(exp), string(edu),
		p.RawResumeText, p.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("save profile: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetProfile(ctx context.Context, huntID string) (domain.Profile, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, hunt_id, name, summary, skills, experience, education, raw_resume_text, updated_at
         FROM profiles WHERE hunt_id=?`, huntID)
	return scanProfile(row)
}

func (s *SQLiteStore) UpdateProfile(ctx context.Context, p domain.Profile) error {
	return s.SaveProfile(ctx, p) // SaveProfile uses INSERT OR REPLACE
}

func scanProfile(s scanner) (domain.Profile, error) {
	var p domain.Profile
	var skillsJSON, expJSON, eduJSON, updatedAt string
	if err := s.Scan(&p.ID, &p.HuntID, &p.Name, &p.Summary,
		&skillsJSON, &expJSON, &eduJSON, &p.RawResumeText, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Profile{}, domain.ErrNotFound
		}
		return domain.Profile{}, fmt.Errorf("scan profile: %w", err)
	}
	if err := json.Unmarshal([]byte(skillsJSON), &p.Skills); err != nil {
		return domain.Profile{}, fmt.Errorf("unmarshal skills: %w", err)
	}
	if err := json.Unmarshal([]byte(expJSON), &p.Experience); err != nil {
		return domain.Profile{}, fmt.Errorf("unmarshal experience: %w", err)
	}
	if err := json.Unmarshal([]byte(eduJSON), &p.Education); err != nil {
		return domain.Profile{}, fmt.Errorf("unmarshal education: %w", err)
	}
	t, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("parse profile updated_at: %w", err)
	}
	p.UpdatedAt = t
	return p, nil
}

// ── Application ───────────────────────────────────────────────────────────────

func (s *SQLiteStore) SaveApplication(ctx context.Context, a domain.Application) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO applications
         (id, hunt_id, company_name, role_title, job_description, status, applied_at, updated_at, notes)
         VALUES (?,?,?,?,?,?,?,?,?)`,
		a.ID, a.HuntID, a.CompanyName, a.RoleTitle, a.JobDescription,
		string(a.Status),
		a.AppliedAt.UTC().Format(time.RFC3339),
		a.UpdatedAt.UTC().Format(time.RFC3339),
		a.Notes,
	)
	if err != nil {
		return fmt.Errorf("save application: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetApplication(ctx context.Context, id string) (domain.Application, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, hunt_id, company_name, role_title, job_description, status, applied_at, updated_at, notes
         FROM applications WHERE id=?`, id)
	return scanApplication(row)
}

func (s *SQLiteStore) ListApplications(ctx context.Context, huntID string) ([]domain.Application, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, hunt_id, company_name, role_title, job_description, status, applied_at, updated_at, notes
         FROM applications WHERE hunt_id=? ORDER BY applied_at DESC`, huntID)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []domain.Application
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (s *SQLiteStore) UpdateApplication(ctx context.Context, a domain.Application) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE applications
         SET company_name=?, role_title=?, job_description=?, status=?, updated_at=?, notes=?
         WHERE id=?`,
		a.CompanyName, a.RoleTitle, a.JobDescription, string(a.Status),
		a.UpdatedAt.UTC().Format(time.RFC3339), a.Notes, a.ID,
	)
	if err != nil {
		return fmt.Errorf("update application: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteApplication(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM stages WHERE application_id=?`, id); err != nil {
		return fmt.Errorf("delete stages for application %s: %w", id, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM insights WHERE application_id=?`, id); err != nil {
		return fmt.Errorf("delete insights for application %s: %w", id, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM applications WHERE id=?`, id); err != nil {
		return fmt.Errorf("delete application %s: %w", id, err)
	}
	return nil
}

func scanApplication(s scanner) (domain.Application, error) {
	var a domain.Application
	var appliedAt, updatedAt string
	if err := s.Scan(&a.ID, &a.HuntID, &a.CompanyName, &a.RoleTitle, &a.JobDescription,
		(*string)(&a.Status), &appliedAt, &updatedAt, &a.Notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Application{}, domain.ErrNotFound
		}
		return domain.Application{}, fmt.Errorf("scan application: %w", err)
	}
	t1, err := time.Parse(time.RFC3339, appliedAt)
	if err != nil {
		return domain.Application{}, fmt.Errorf("parse application applied_at: %w", err)
	}
	a.AppliedAt = t1
	t2, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return domain.Application{}, fmt.Errorf("parse application updated_at: %w", err)
	}
	a.UpdatedAt = t2
	return a, nil
}

// ── Stage ─────────────────────────────────────────────────────────────────────

func (s *SQLiteStore) SaveStage(ctx context.Context, st domain.Stage) error {
	var date *string
	if st.Date != nil {
		v := st.Date.UTC().Format(time.RFC3339)
		date = &v
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO stages (id, application_id, type, label, date, notes, feedback, outcome, sort_order)
         VALUES (?,?,?,?,?,?,?,?,?)`,
		st.ID, st.ApplicationID, string(st.Type), st.Label, date,
		st.Notes, st.Feedback, string(st.Outcome), st.Order,
	)
	if err != nil {
		return fmt.Errorf("save stage: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetStage(ctx context.Context, id string) (domain.Stage, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, application_id, type, label, date, notes, feedback, outcome, sort_order
         FROM stages WHERE id=?`, id)
	return scanStage(row)
}

func (s *SQLiteStore) ListStages(ctx context.Context, applicationID string) ([]domain.Stage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, application_id, type, label, date, notes, feedback, outcome, sort_order
         FROM stages WHERE application_id=? ORDER BY sort_order ASC`, applicationID)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	defer rows.Close()

	var stages []domain.Stage
	for rows.Next() {
		st, err := scanStage(rows)
		if err != nil {
			return nil, err
		}
		stages = append(stages, st)
	}
	return stages, rows.Err()
}

func (s *SQLiteStore) UpdateStage(ctx context.Context, st domain.Stage) error {
	var date *string
	if st.Date != nil {
		v := st.Date.UTC().Format(time.RFC3339)
		date = &v
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE stages SET type=?, label=?, date=?, notes=?, feedback=?, outcome=?, sort_order=?
         WHERE id=?`,
		string(st.Type), st.Label, date, st.Notes, st.Feedback, string(st.Outcome), st.Order, st.ID,
	)
	if err != nil {
		return fmt.Errorf("update stage: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteStage(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM stages WHERE id=?`, id); err != nil {
		return fmt.Errorf("delete stage %s: %w", id, err)
	}
	return nil
}

func scanStage(s scanner) (domain.Stage, error) {
	var st domain.Stage
	var date *string
	if err := s.Scan(&st.ID, &st.ApplicationID, (*string)(&st.Type), &st.Label, &date,
		&st.Notes, &st.Feedback, (*string)(&st.Outcome), &st.Order); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Stage{}, domain.ErrNotFound
		}
		return domain.Stage{}, fmt.Errorf("scan stage: %w", err)
	}
	if date != nil {
		t, err := time.Parse(time.RFC3339, *date)
		if err != nil {
			return domain.Stage{}, fmt.Errorf("parse stage date: %w", err)
		}
		st.Date = &t
	}
	return st, nil
}

// ── Insight ───────────────────────────────────────────────────────────────────

func (s *SQLiteStore) SaveInsight(ctx context.Context, i domain.Insight) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO insights (id, application_id, content, generated_at)
         VALUES (?,?,?,?)`,
		i.ID, i.ApplicationID, i.Content, i.GeneratedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("save insight: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetInsight(ctx context.Context, applicationID string) (domain.Insight, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, application_id, content, generated_at FROM insights WHERE application_id=?`,
		applicationID)
	return scanInsight(row)
}

func scanInsight(s scanner) (domain.Insight, error) {
	var i domain.Insight
	var generatedAt string
	if err := s.Scan(&i.ID, &i.ApplicationID, &i.Content, &generatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Insight{}, domain.ErrNotFound
		}
		return domain.Insight{}, fmt.Errorf("scan insight: %w", err)
	}
	t, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("parse insight generated_at: %w", err)
	}
	i.GeneratedAt = t
	return i, nil
}
