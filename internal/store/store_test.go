package store

import (
	"context"
	"errors"
	"testing"

	"github.com/Tavo5691/job-searcher/internal/domain"
	"github.com/Tavo5691/job-searcher/internal/port"
)

// compile-time check that SQLiteStore implements port.Store
var _ port.Store = (*SQLiteStore)(nil)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSaveAndGetHunt(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	h := domain.Hunt{ID: "h1", Title: "My Hunt", Status: domain.HuntStatusActive}
	if err := s.SaveHunt(ctx, h); err != nil {
		t.Fatalf("SaveHunt: %v", err)
	}

	got, err := s.GetHunt(ctx, "h1")
	if err != nil {
		t.Fatalf("GetHunt: %v", err)
	}
	if got.ID != h.ID || got.Title != h.Title {
		t.Errorf("GetHunt = %+v, want %+v", got, h)
	}
}

func TestGetHuntNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetHunt(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListHunts(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for _, id := range []string{"h1", "h2"} {
		if err := s.SaveHunt(ctx, domain.Hunt{ID: id, Title: id, Status: domain.HuntStatusActive}); err != nil {
			t.Fatalf("SaveHunt %s: %v", id, err)
		}
	}

	hunts, err := s.ListHunts(ctx)
	if err != nil {
		t.Fatalf("ListHunts: %v", err)
	}
	if len(hunts) != 2 {
		t.Errorf("ListHunts returned %d items, want 2", len(hunts))
	}
}

func TestSaveAndGetProfile(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p := domain.Profile{
		ID:     "p1",
		HuntID: "h1",
		Name:   "Alice",
		Skills: []string{"Go", "Postgres"},
	}
	if err := s.SaveProfile(ctx, p); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	got, err := s.GetProfile(ctx, "h1")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.Name != p.Name {
		t.Errorf("Name = %q, want %q", got.Name, p.Name)
	}
	if len(got.Skills) != 2 {
		t.Errorf("Skills length = %d, want 2", len(got.Skills))
	}
}

func TestSaveAndGetApplication(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a := domain.Application{
		ID:          "a1",
		HuntID:      "h1",
		CompanyName: "Acme",
		RoleTitle:   "Engineer",
		Status:      domain.ApplicationStatusApplied,
	}
	if err := s.SaveApplication(ctx, a); err != nil {
		t.Fatalf("SaveApplication: %v", err)
	}

	got, err := s.GetApplication(ctx, "a1")
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if got.CompanyName != "Acme" {
		t.Errorf("CompanyName = %q, want Acme", got.CompanyName)
	}
}

func TestSaveAndGetStage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	st := domain.Stage{
		ID:            "s1",
		ApplicationID: "a1",
		Type:          domain.StageTypeRecruiterScreen,
		Outcome:       domain.StageOutcomePending,
		Order:         1,
	}
	if err := s.SaveStage(ctx, st); err != nil {
		t.Fatalf("SaveStage: %v", err)
	}

	got, err := s.GetStage(ctx, "s1")
	if err != nil {
		t.Fatalf("GetStage: %v", err)
	}
	if got.Type != domain.StageTypeRecruiterScreen {
		t.Errorf("Type = %q, want recruiter_screen", got.Type)
	}
}

func TestSaveAndGetInsight(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	i := domain.Insight{
		ID:            "i1",
		ApplicationID: "a1",
		Content:       "some insight",
	}
	if err := s.SaveInsight(ctx, i); err != nil {
		t.Fatalf("SaveInsight: %v", err)
	}

	got, err := s.GetInsight(ctx, "a1")
	if err != nil {
		t.Fatalf("GetInsight: %v", err)
	}
	if got.Content != "some insight" {
		t.Errorf("Content = %q, want 'some insight'", got.Content)
	}
}
