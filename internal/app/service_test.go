package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tavo5691/job-searcher/internal/domain"
)

// TestNewService verifies that the Service constructor wires ports and is non-nil.
func TestNewService(t *testing.T) {
	svc := NewService(&stubStore{}, &stubLLM{}, &stubPDF{})
	if svc == nil {
		t.Fatal("NewService must return a non-nil *Service")
	}
}

// TestServiceCreateHunt verifies that CreateHunt persists a Hunt and returns it.
func TestServiceCreateHunt(t *testing.T) {
	store := &stubStore{}
	svc := NewService(store, &stubLLM{}, &stubPDF{})
	ctx := context.Background()

	h, err := svc.CreateHunt(ctx, "My Hunt")
	if err != nil {
		t.Fatalf("CreateHunt returned error: %v", err)
	}
	if h.ID == "" {
		t.Error("CreateHunt must assign a non-empty ID")
	}
	if h.Title != "My Hunt" {
		t.Errorf("Title = %q, want %q", h.Title, "My Hunt")
	}
	if h.Status != domain.HuntStatusActive {
		t.Errorf("Status = %q, want Active", h.Status)
	}
	if h.CreatedAt.IsZero() {
		t.Error("CreatedAt must be set")
	}
}

// TestServiceCloseHunt verifies that CloseHunt transitions a Hunt to closed.
func TestServiceCloseHunt(t *testing.T) {
	t.Run("active → closed", func(t *testing.T) {
		store := &stubStore{}
		svc := NewService(store, &stubLLM{}, &stubPDF{})
		ctx := context.Background()

		// Create a hunt via the service so it's in the stub store.
		h, err := svc.CreateHunt(ctx, "Closing Hunt")
		if err != nil {
			t.Fatalf("CreateHunt: %v", err)
		}

		closed, err := svc.CloseHunt(ctx, h.ID)
		if err != nil {
			t.Fatalf("CloseHunt returned error: %v", err)
		}
		if closed.Status != domain.HuntStatusClosed {
			t.Errorf("Status = %q, want %q", closed.Status, domain.HuntStatusClosed)
		}
		if closed.ClosedAt == nil {
			t.Error("ClosedAt must be set after CloseHunt")
		}
	})

	t.Run("already closed", func(t *testing.T) {
		store := &stubStore{}
		svc := NewService(store, &stubLLM{}, &stubPDF{})
		ctx := context.Background()

		h, err := svc.CreateHunt(ctx, "Already Closed Hunt")
		if err != nil {
			t.Fatalf("CreateHunt: %v", err)
		}

		// First close — must succeed.
		first, err := svc.CloseHunt(ctx, h.ID)
		if err != nil {
			t.Fatalf("first CloseHunt returned error: %v", err)
		}
		if first.Status != domain.HuntStatusClosed {
			t.Errorf("first CloseHunt status = %q, want %q", first.Status, domain.HuntStatusClosed)
		}

		// Second close — must NOT return an error; hunt stays closed.
		second, err := svc.CloseHunt(ctx, h.ID)
		if err != nil {
			t.Fatalf("second CloseHunt returned error: %v", err)
		}
		if second.Status != domain.HuntStatusClosed {
			t.Errorf("second CloseHunt status = %q, want %q", second.Status, domain.HuntStatusClosed)
		}
	})
}

// TestServiceListHunts verifies that ListHunts delegates to the store.
func TestServiceListHunts(t *testing.T) {
	store := &stubStore{
		hunts: []domain.Hunt{{ID: "1", Title: "h1", Status: domain.HuntStatusActive, CreatedAt: time.Now()}},
	}
	svc := NewService(store, &stubLLM{}, &stubPDF{})
	hunts, err := svc.ListHunts(context.Background())
	if err != nil {
		t.Fatalf("ListHunts error: %v", err)
	}
	if len(hunts) != 1 {
		t.Errorf("ListHunts returned %d items, want 1", len(hunts))
	}
}

// TestServiceGetHunt verifies that GetHunt retrieves by ID and surfaces ErrNotFound.
func TestServiceGetHunt(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		store := &stubStore{}
		svc := NewService(store, &stubLLM{}, &stubPDF{})
		ctx := context.Background()

		h, err := svc.CreateHunt(ctx, "Roundtrip Hunt")
		if err != nil {
			t.Fatalf("CreateHunt: %v", err)
		}

		got, err := svc.GetHunt(ctx, h.ID)
		if err != nil {
			t.Fatalf("GetHunt returned error: %v", err)
		}
		if got.ID != h.ID {
			t.Errorf("ID = %q, want %q", got.ID, h.ID)
		}
		if got.Title != h.Title {
			t.Errorf("Title = %q, want %q", got.Title, h.Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &stubStore{}
		svc := NewService(store, &stubLLM{}, &stubPDF{})

		_, err := svc.GetHunt(context.Background(), "nonexistent-id")
		if err == nil {
			t.Fatal("GetHunt with unknown ID must return an error")
		}
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("error = %v, must wrap domain.ErrNotFound", err)
		}
	})
}

// TestServiceListApplications verifies that ListApplications delegates to the store.
func TestServiceListApplications(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		svc := NewService(&stubStore{}, &stubLLM{}, &stubPDF{})
		apps, err := svc.ListApplications(context.Background(), "hunt-1")
		if err != nil {
			t.Fatalf("ListApplications error: %v", err)
		}
		if len(apps) != 0 {
			t.Errorf("expected 0 apps, got %d", len(apps))
		}
	})

	t.Run("returns stored applications", func(t *testing.T) {
		store := &stubStore{
			applications: []domain.Application{
				{ID: "a1", HuntID: "hunt-1", CompanyName: "Acme", RoleTitle: "Engineer"},
				{ID: "a2", HuntID: "hunt-1", CompanyName: "Beta", RoleTitle: "Analyst"},
			},
		}
		svc := NewService(store, &stubLLM{}, &stubPDF{})
		apps, err := svc.ListApplications(context.Background(), "hunt-1")
		if err != nil {
			t.Fatalf("ListApplications error: %v", err)
		}
		if len(apps) != 2 {
			t.Errorf("expected 2 apps, got %d", len(apps))
		}
	})
}

// --- stubs ---

type stubStore struct {
	hunts        []domain.Hunt
	applications []domain.Application
}

func (s *stubStore) SaveHunt(ctx context.Context, h domain.Hunt) error {
	s.hunts = append(s.hunts, h)
	return nil
}
func (s *stubStore) GetHunt(ctx context.Context, id string) (domain.Hunt, error) {
	for _, h := range s.hunts {
		if h.ID == id {
			return h, nil
		}
	}
	return domain.Hunt{}, domain.ErrNotFound
}
func (s *stubStore) ListHunts(ctx context.Context) ([]domain.Hunt, error) {
	return s.hunts, nil
}
func (s *stubStore) UpdateHunt(ctx context.Context, h domain.Hunt) error {
	for i, existing := range s.hunts {
		if existing.ID == h.ID {
			s.hunts[i] = h
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *stubStore) SaveProfile(ctx context.Context, p domain.Profile) error { return nil }
func (s *stubStore) GetProfile(ctx context.Context, huntID string) (domain.Profile, error) {
	return domain.Profile{}, domain.ErrNotFound
}
func (s *stubStore) UpdateProfile(ctx context.Context, p domain.Profile) error { return nil }

func (s *stubStore) SaveApplication(ctx context.Context, a domain.Application) error { return nil }
func (s *stubStore) GetApplication(ctx context.Context, id string) (domain.Application, error) {
	return domain.Application{}, domain.ErrNotFound
}
func (s *stubStore) ListApplications(ctx context.Context, huntID string) ([]domain.Application, error) {
	return s.applications, nil
}
func (s *stubStore) UpdateApplication(ctx context.Context, a domain.Application) error { return nil }
func (s *stubStore) DeleteApplication(ctx context.Context, id string) error            { return nil }

func (s *stubStore) SaveStage(ctx context.Context, st domain.Stage) error { return nil }
func (s *stubStore) GetStage(ctx context.Context, id string) (domain.Stage, error) {
	return domain.Stage{}, domain.ErrNotFound
}
func (s *stubStore) ListStages(ctx context.Context, applicationID string) ([]domain.Stage, error) {
	return nil, nil
}
func (s *stubStore) UpdateStage(ctx context.Context, st domain.Stage) error { return nil }
func (s *stubStore) DeleteStage(ctx context.Context, id string) error       { return nil }

func (s *stubStore) SaveInsight(ctx context.Context, i domain.Insight) error { return nil }
func (s *stubStore) GetInsight(ctx context.Context, applicationID string) (domain.Insight, error) {
	return domain.Insight{}, domain.ErrNotFound
}

type stubLLM struct{}

func (l *stubLLM) Complete(ctx context.Context, prompt string) (string, error) { return "ok", nil }

type stubPDF struct{}

func (p *stubPDF) ExtractText(ctx context.Context, path string) (string, error) { return "text", nil }
