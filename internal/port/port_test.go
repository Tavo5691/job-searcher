// Package port defines shared interfaces consumed by multiple layers.
package port

import (
	"context"
	"testing"

	"github.com/Tavo5691/job-searcher/internal/domain"
)

// Compile-time checks that the interfaces are well-formed.
// These tests do not run logic; they just ensure the types exist and are usable.

func TestStoreInterfaceExists(t *testing.T) {
	// Verifies that Store interface is defined with the expected method set.
	var _ Store = (*stubStore)(nil)
}

func TestLLMProviderInterfaceExists(t *testing.T) {
	var _ LLMProvider = (*stubLLM)(nil)
}

func TestPDFParserInterfaceExists(t *testing.T) {
	var _ PDFParser = (*stubPDF)(nil)
}

// --- stubs to satisfy interface compile checks ---

type stubStore struct{}

func (s *stubStore) SaveHunt(ctx context.Context, h domain.Hunt) error { return nil }
func (s *stubStore) GetHunt(ctx context.Context, id string) (domain.Hunt, error) {
	return domain.Hunt{}, nil
}
func (s *stubStore) ListHunts(ctx context.Context) ([]domain.Hunt, error) { return nil, nil }
func (s *stubStore) UpdateHunt(ctx context.Context, h domain.Hunt) error  { return nil }

func (s *stubStore) SaveProfile(ctx context.Context, p domain.Profile) error { return nil }
func (s *stubStore) GetProfile(ctx context.Context, huntID string) (domain.Profile, error) {
	return domain.Profile{}, nil
}
func (s *stubStore) UpdateProfile(ctx context.Context, p domain.Profile) error { return nil }

func (s *stubStore) SaveApplication(ctx context.Context, a domain.Application) error { return nil }
func (s *stubStore) GetApplication(ctx context.Context, id string) (domain.Application, error) {
	return domain.Application{}, nil
}
func (s *stubStore) ListApplications(ctx context.Context, huntID string) ([]domain.Application, error) {
	return nil, nil
}
func (s *stubStore) UpdateApplication(ctx context.Context, a domain.Application) error { return nil }
func (s *stubStore) DeleteApplication(ctx context.Context, id string) error            { return nil }

func (s *stubStore) SaveStage(ctx context.Context, st domain.Stage) error { return nil }
func (s *stubStore) GetStage(ctx context.Context, id string) (domain.Stage, error) {
	return domain.Stage{}, nil
}
func (s *stubStore) ListStages(ctx context.Context, applicationID string) ([]domain.Stage, error) {
	return nil, nil
}
func (s *stubStore) UpdateStage(ctx context.Context, st domain.Stage) error { return nil }
func (s *stubStore) DeleteStage(ctx context.Context, id string) error       { return nil }

func (s *stubStore) SaveInsight(ctx context.Context, i domain.Insight) error { return nil }
func (s *stubStore) GetInsight(ctx context.Context, applicationID string) (domain.Insight, error) {
	return domain.Insight{}, nil
}

type stubLLM struct{}

func (l *stubLLM) Complete(ctx context.Context, prompt string) (string, error) { return "", nil }

type stubPDF struct{}

func (p *stubPDF) ExtractText(ctx context.Context, path string) (string, error) { return "", nil }
