// Package port defines shared interfaces consumed across multiple layers.
// Interfaces are centralised here per AGENTS.md convention.
package port

import (
	"context"

	"github.com/Tavo5691/job-searcher/internal/domain"
)

// Store is the persistence interface for all domain entities.
// Implementations live in internal/store/.
type Store interface {
	// Hunt operations
	SaveHunt(ctx context.Context, h domain.Hunt) error
	GetHunt(ctx context.Context, id string) (domain.Hunt, error)
	ListHunts(ctx context.Context) ([]domain.Hunt, error)
	UpdateHunt(ctx context.Context, h domain.Hunt) error

	// Profile operations
	SaveProfile(ctx context.Context, p domain.Profile) error
	GetProfile(ctx context.Context, huntID string) (domain.Profile, error)
	UpdateProfile(ctx context.Context, p domain.Profile) error

	// Application operations
	SaveApplication(ctx context.Context, a domain.Application) error
	GetApplication(ctx context.Context, id string) (domain.Application, error)
	ListApplications(ctx context.Context, huntID string) ([]domain.Application, error)
	UpdateApplication(ctx context.Context, a domain.Application) error
	DeleteApplication(ctx context.Context, id string) error

	// Stage operations
	SaveStage(ctx context.Context, st domain.Stage) error
	GetStage(ctx context.Context, id string) (domain.Stage, error)
	ListStages(ctx context.Context, applicationID string) ([]domain.Stage, error)
	UpdateStage(ctx context.Context, st domain.Stage) error
	DeleteStage(ctx context.Context, id string) error

	// Insight operations
	SaveInsight(ctx context.Context, i domain.Insight) error
	GetInsight(ctx context.Context, applicationID string) (domain.Insight, error)
}
