// Package app implements the application service layer.
// It orchestrates domain logic and ports. The TUI calls only this package.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Tavo5691/job-searcher/internal/domain"
	"github.com/Tavo5691/job-searcher/internal/port"
)

// Service is the application service. All use cases are methods on Service.
type Service struct {
	store port.Store
	llm   port.LLMProvider
	pdf   port.PDFParser
}

// NewService creates a new Service with the given ports wired in.
func NewService(store port.Store, llm port.LLMProvider, pdf port.PDFParser) *Service {
	return &Service{store: store, llm: llm, pdf: pdf}
}

// CreateHunt creates a new active Hunt with the given title.
func (s *Service) CreateHunt(ctx context.Context, title string) (domain.Hunt, error) {
	h := domain.Hunt{
		ID:        uuid.New().String(),
		Title:     title,
		Status:    domain.HuntStatusActive,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.SaveHunt(ctx, h); err != nil {
		return domain.Hunt{}, fmt.Errorf("create hunt: %w", err)
	}
	return h, nil
}

// ListHunts returns all Hunts from the store.
func (s *Service) ListHunts(ctx context.Context) ([]domain.Hunt, error) {
	hunts, err := s.store.ListHunts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list hunts: %w", err)
	}
	return hunts, nil
}

// GetHunt retrieves a single Hunt by ID.
func (s *Service) GetHunt(ctx context.Context, id string) (domain.Hunt, error) {
	h, err := s.store.GetHunt(ctx, id)
	if err != nil {
		return domain.Hunt{}, fmt.Errorf("get hunt %s: %w", id, err)
	}
	return h, nil
}

// CloseHunt marks a Hunt as closed.
func (s *Service) CloseHunt(ctx context.Context, id string) (domain.Hunt, error) {
	h, err := s.store.GetHunt(ctx, id)
	if err != nil {
		return domain.Hunt{}, fmt.Errorf("close hunt: %w", err)
	}
	now := time.Now().UTC()
	h.Status = domain.HuntStatusClosed
	h.ClosedAt = &now
	if err := s.store.UpdateHunt(ctx, h); err != nil {
		return domain.Hunt{}, fmt.Errorf("close hunt: %w", err)
	}
	return h, nil
}

// CreateApplication creates a new Application within a Hunt.
func (s *Service) CreateApplication(ctx context.Context, huntID, companyName, roleTitle, jobDescription string) (domain.Application, error) {
	a := domain.Application{
		ID:             uuid.New().String(),
		HuntID:         huntID,
		CompanyName:    companyName,
		RoleTitle:      roleTitle,
		JobDescription: jobDescription,
		Status:         domain.ApplicationStatusApplied,
		AppliedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := s.store.SaveApplication(ctx, a); err != nil {
		return domain.Application{}, fmt.Errorf("create application: %w", err)
	}
	return a, nil
}

// ListApplications returns all Applications within a Hunt.
func (s *Service) ListApplications(ctx context.Context, huntID string) ([]domain.Application, error) {
	apps, err := s.store.ListApplications(ctx, huntID)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	return apps, nil
}

// GetApplication retrieves a single Application by ID.
func (s *Service) GetApplication(ctx context.Context, id string) (domain.Application, error) {
	a, err := s.store.GetApplication(ctx, id)
	if err != nil {
		return domain.Application{}, fmt.Errorf("get application %s: %w", id, err)
	}
	return a, nil
}

// UpdateApplication persists changes to an existing Application.
func (s *Service) UpdateApplication(ctx context.Context, a domain.Application) (domain.Application, error) {
	a.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateApplication(ctx, a); err != nil {
		return domain.Application{}, fmt.Errorf("update application: %w", err)
	}
	return a, nil
}

// DeleteApplication removes an Application and its Stages from the store.
func (s *Service) DeleteApplication(ctx context.Context, id string) error {
	if err := s.store.DeleteApplication(ctx, id); err != nil {
		return fmt.Errorf("delete application %s: %w", id, err)
	}
	return nil
}

// AddStage appends a Stage to an Application.
func (s *Service) AddStage(ctx context.Context, st domain.Stage) (domain.Stage, error) {
	st.ID = uuid.New().String()
	if st.Outcome == "" {
		st.Outcome = domain.StageOutcomePending
	}
	if err := s.store.SaveStage(ctx, st); err != nil {
		return domain.Stage{}, fmt.Errorf("add stage: %w", err)
	}
	return st, nil
}

// ListStages returns all Stages for an Application.
func (s *Service) ListStages(ctx context.Context, applicationID string) ([]domain.Stage, error) {
	stages, err := s.store.ListStages(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	return stages, nil
}

// UpdateStage persists changes to an existing Stage.
func (s *Service) UpdateStage(ctx context.Context, st domain.Stage) (domain.Stage, error) {
	if err := s.store.UpdateStage(ctx, st); err != nil {
		return domain.Stage{}, fmt.Errorf("update stage: %w", err)
	}
	return st, nil
}

// DeleteStage removes a Stage from the store.
func (s *Service) DeleteStage(ctx context.Context, id string) error {
	if err := s.store.DeleteStage(ctx, id); err != nil {
		return fmt.Errorf("delete stage %s: %w", id, err)
	}
	return nil
}

// GetProfile retrieves the Profile for a Hunt.
func (s *Service) GetProfile(ctx context.Context, huntID string) (domain.Profile, error) {
	p, err := s.store.GetProfile(ctx, huntID)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("get profile for hunt %s: %w", huntID, err)
	}
	return p, nil
}

// SaveProfile persists a Profile (insert or update).
func (s *Service) SaveProfile(ctx context.Context, p domain.Profile) (domain.Profile, error) {
	p.UpdatedAt = time.Now().UTC()
	if err := s.store.SaveProfile(ctx, p); err != nil {
		return domain.Profile{}, fmt.Errorf("save profile: %w", err)
	}
	return p, nil
}

// GenerateInsight calls the LLM to produce structured advice for an Application.
func (s *Service) GenerateInsight(ctx context.Context, applicationID string) (domain.Insight, error) {
	app, err := s.store.GetApplication(ctx, applicationID)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("generate insight: get application: %w", err)
	}

	stages, err := s.store.ListStages(ctx, applicationID)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("generate insight: list stages: %w", err)
	}

	prompt := buildInsightPrompt(app, stages)
	content, err := s.llm.Complete(ctx, prompt)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("generate insight: llm complete: %w", err)
	}

	insight := domain.Insight{
		ID:            uuid.New().String(),
		ApplicationID: applicationID,
		Content:       content,
		GeneratedAt:   time.Now().UTC(),
	}
	if err := s.store.SaveInsight(ctx, insight); err != nil {
		return domain.Insight{}, fmt.Errorf("generate insight: save: %w", err)
	}
	return insight, nil
}

// GetInsight retrieves the current Insight for an Application.
func (s *Service) GetInsight(ctx context.Context, applicationID string) (domain.Insight, error) {
	i, err := s.store.GetInsight(ctx, applicationID)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("get insight for application %s: %w", applicationID, err)
	}
	return i, nil
}

// ParseResume extracts text from a PDF and stores it in the Profile.
func (s *Service) ParseResume(ctx context.Context, huntID, pdfPath string) (domain.Profile, error) {
	text, err := s.pdf.ExtractText(ctx, pdfPath)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("parse resume: extract text: %w", err)
	}

	prompt := buildResumePrompt(text)
	_, err = s.llm.Complete(ctx, prompt)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("parse resume: llm interpret: %w", err)
	}

	// For the scaffold, store the raw text; LLM parsing is wired but JSON extraction
	// belongs to a later change.
	p := domain.Profile{
		ID:            uuid.New().String(),
		HuntID:        huntID,
		RawResumeText: text,
		UpdatedAt:     time.Now().UTC(),
	}
	if err := s.store.SaveProfile(ctx, p); err != nil {
		return domain.Profile{}, fmt.Errorf("parse resume: save profile: %w", err)
	}
	return p, nil
}

// buildInsightPrompt constructs the LLM prompt for Insight generation.
func buildInsightPrompt(app domain.Application, stages []domain.Stage) string {
	prompt := fmt.Sprintf(
		"Generate structured markdown advice for a job application.\n\nCompany: %s\nRole: %s\nJob Description: %s\nNotes: %s\n\nStages (%d total):\n",
		app.CompanyName, app.RoleTitle, app.JobDescription, app.Notes, len(stages),
	)
	for _, st := range stages {
		prompt += fmt.Sprintf("- %s | outcome: %s | feedback: %s\n", st.Type, st.Outcome, st.Feedback)
	}
	prompt += "\nProvide: current status summary, preparation recommendations, areas to address, overall assessment."
	return prompt
}

// buildResumePrompt constructs the LLM prompt for resume parsing.
func buildResumePrompt(rawText string) string {
	return fmt.Sprintf(
		"Parse the following resume text and return structured JSON matching this schema:\n"+
			"{name, summary, skills: [], experience: [{company, role, start, end, notes}], education: [{institution, degree, field, year}]}\n\n"+
			"Resume text:\n%s",
		rawText,
	)
}
