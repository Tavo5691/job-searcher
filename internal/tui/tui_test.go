// Package tui provides the terminal user interface using Bubble Tea.
package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Tavo5691/job-searcher/internal/domain"
)

// compile-time check that stubService satisfies the minimal interface used by TUI
var _ serviceIface = (*stubService)(nil)

func TestNewApp(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	if app == nil {
		t.Error("NewApp must return a non-nil App")
	}
}

// TestAppInitialView asserts that the initial view contains "Hunts" header
// and the j/k/q help text — regression guard.
func TestAppInitialView(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)

	view := app.View()

	if !strings.Contains(view, "Hunts") {
		t.Errorf("expected view to contain 'Hunts', got:\n%s", view)
	}
	wantHelp := []string{"j", "k", "q"}
	for _, key := range wantHelp {
		if !strings.Contains(view, key) {
			t.Errorf("expected view to contain help key %q, got:\n%s", key, view)
		}
	}
}

// TestHuntListRowFormat asserts that each hunt list row follows the spec format:
// "{Title} ({status}) — {N} applications"
func TestHuntListRowFormat(t *testing.T) {
	tests := []struct {
		name       string
		hunt       domain.Hunt
		count      int
		wantSubstr string
	}{
		{
			name:       "active hunt with 3 apps",
			hunt:       domain.Hunt{ID: "h1", Title: "Q1 2026", Status: "active"},
			count:      3,
			wantSubstr: "Q1 2026 (active) \u2014 3 applications",
		},
		{
			name:       "closed hunt with 0 apps",
			hunt:       domain.Hunt{ID: "h2", Title: "Old Hunt", Status: "closed"},
			count:      0,
			wantSubstr: "Old Hunt (closed) \u2014 0 applications",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{}
			app := NewApp(svc)
			app.hunts = []domain.Hunt{tc.hunt}
			app.counts = map[string]int{tc.hunt.ID: tc.count}

			view := app.View()

			if !strings.Contains(view, tc.wantSubstr) {
				t.Errorf("expected view to contain %q, got:\n%s", tc.wantSubstr, view)
			}
		})
	}
}

// TestHuntListCountDisplay asserts that the hunt list view shows application
// counts per hunt row. Two hunts: one with 3 apps, one with 0.
func TestHuntListCountDisplay(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)

	// Inject two hunts and their counts directly (same package access).
	app.hunts = []domain.Hunt{
		{ID: "h1", Title: "Big Tech Hunt", Status: "active"},
		{ID: "h2", Title: "Startup Hunt", Status: "active"},
	}
	app.counts = map[string]int{
		"h1": 3,
		"h2": 0,
	}

	view := app.View()

	if !strings.Contains(view, "3 applications") {
		t.Errorf("expected view to contain '3 applications', got:\n%s", view)
	}
	if !strings.Contains(view, "0 applications") {
		t.Errorf("expected view to contain '0 applications', got:\n%s", view)
	}
}

// TestHuntCreationFlow asserts that pressing Enter in huntInput view
// calls CreateHunt with the input value and issues loadHuntsAndCountsCmd.
func TestHuntCreationFlow(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = huntInput

	// Set input value by simulating rune key presses
	for _, r := range "My Hunt" {
		app.input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	// Force set the input value via the textinput model
	app.input.SetValue("My Hunt")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	// The createHuntCmd should be issued — execute it to trigger the service call
	if cmd == nil {
		t.Fatal("expected a command to be returned")
	}
	msg := cmd()

	// Now send the huntCreatedMsg to transition back to huntList
	model2, cmd2 := updated.Update(msg)
	updated2 := model2.(*App)

	if !svc.createHuntCalled {
		t.Error("expected CreateHunt to be called")
	}
	if svc.createHuntName != "My Hunt" {
		t.Errorf("expected CreateHunt called with 'My Hunt', got %q", svc.createHuntName)
	}
	// After huntCreatedMsg success, should transition back to huntList and issue loadHuntsAndCountsCmd
	if updated2.currentView != huntList {
		t.Errorf("expected currentView == huntList (%d), got %d", huntList, updated2.currentView)
	}
	if cmd2 == nil {
		t.Error("expected loadHuntsAndCountsCmd to be issued after successful creation")
	}
}

// TestCloseHuntFlow asserts that pressing 'c' in huntList view on an active hunt
// calls CloseHunt and reloads hunts. Also tests that 'c' on a closed hunt sets statusMsg.
func TestCloseHuntFlow(t *testing.T) {
	t.Run("active hunt closes", func(t *testing.T) {
		svc := &stubService{}
		app := NewApp(svc)
		app.hunts = []domain.Hunt{
			{ID: "h1", Title: "Big Tech Hunt", Status: "active"},
		}
		app.cursor = 0

		model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
		updated := model.(*App)

		if cmd == nil {
			t.Fatal("expected a command to be returned for active hunt close")
		}

		// Execute the closeHuntCmd to trigger service call
		msg := cmd()

		if !svc.closeHuntCalled {
			t.Error("expected CloseHunt to be called")
		}
		if svc.closeHuntID != "h1" {
			t.Errorf("expected CloseHunt called with 'h1', got %q", svc.closeHuntID)
		}

		// Send the huntClosedMsg back — should issue loadHuntsAndCountsCmd
		model2, cmd2 := updated.Update(msg)
		_ = model2

		if cmd2 == nil {
			t.Error("expected loadHuntsAndCountsCmd to be issued after successful close")
		}
	})

	t.Run("closed hunt sets statusMsg", func(t *testing.T) {
		svc := &stubService{}
		app := NewApp(svc)
		app.hunts = []domain.Hunt{
			{ID: "h1", Title: "Old Hunt", Status: "closed"},
		}
		app.cursor = 0

		model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
		updated := model.(*App)

		if svc.closeHuntCalled {
			t.Error("expected CloseHunt NOT to be called for already-closed hunt")
		}
		if cmd != nil {
			t.Error("expected no command to be issued for already-closed hunt")
		}
		if updated.statusMsg == "" {
			t.Error("expected statusMsg to be set for already-closed hunt")
		}
		if !strings.Contains(updated.statusMsg, "closed") {
			t.Errorf("expected statusMsg to mention 'closed', got %q", updated.statusMsg)
		}
	})
}

// TestEscReturnsFromDetail asserts that pressing Esc in huntDetail view
// returns to huntList view.
func TestEscReturnsFromDetail(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = huntDetail
	app.hunts = []domain.Hunt{
		{ID: "h1", Title: "Big Tech Hunt", Status: "active"},
	}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != huntList {
		t.Errorf("expected currentView == huntList (%d), got %d", huntList, updated.currentView)
	}
}

// TestEnterNavigatesToDetail asserts that pressing Enter in huntList view
// with a hunt selected navigates to huntDetail and shows the hunt name.
func TestEnterNavigatesToDetail(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.hunts = []domain.Hunt{
		{ID: "h1", Title: "Big Tech Hunt", Status: "active"},
	}
	app.cursor = 0

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != huntDetail {
		t.Errorf("expected currentView == huntDetail (%d), got %d", huntDetail, updated.currentView)
	}
	v := updated.View()
	if !strings.Contains(v, "Big Tech Hunt") {
		t.Errorf("expected View() to contain hunt name 'Big Tech Hunt', got:\n%s", v)
	}
}

// TestEscCancelsInput asserts that pressing Esc in huntInput view
// returns to huntList view and clears the input.
func TestEscCancelsInput(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = huntInput
	app.input.SetValue("partial name")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != huntList {
		t.Errorf("expected currentView == huntList (%d), got %d", huntList, updated.currentView)
	}
	if updated.input.Value() != "" {
		t.Errorf("expected input cleared, got %q", updated.input.Value())
	}
}

// TestEmptyHuntNameIgnored asserts that pressing Enter with empty/whitespace input
// does NOT call CreateHunt and stays in huntInput view.
func TestEmptyHuntNameIgnored(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{}
			app := NewApp(svc)
			app.currentView = huntInput
			app.input.SetValue(tc.value)

			model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
			updated := model.(*App)

			if svc.createHuntCalled {
				t.Error("expected CreateHunt NOT to be called on empty input")
			}
			if updated.currentView != huntInput {
				t.Errorf("expected to remain in huntInput, got view %d", updated.currentView)
			}
			if cmd != nil {
				t.Error("expected no command to be issued on empty input")
			}
		})
	}
}

// TestNKeyActivatesInputMode asserts that pressing 'n' in huntList view
// switches to huntInput view and clears any status message.
func TestNKeyActivatesInputMode(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.statusMsg = "some previous status"

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updated := model.(*App)

	if updated.currentView != huntInput {
		t.Errorf("expected currentView == huntInput (%d), got %d", huntInput, updated.currentView)
	}
	if updated.statusMsg != "" {
		t.Errorf("expected statusMsg cleared, got %q", updated.statusMsg)
	}
}

// stubService satisfies serviceIface for testing.
type stubService struct {
	createHuntCalled bool
	createHuntName   string
	closeHuntCalled  bool
	closeHuntID      string
	hunts            []domain.Hunt
	createHuntErr    error
	closeHuntErr     error
	closedHunt       domain.Hunt
}

func (s *stubService) ListHunts(ctx context.Context) ([]domain.Hunt, error) {
	return s.hunts, nil
}
func (s *stubService) CreateHunt(ctx context.Context, title string) (domain.Hunt, error) {
	s.createHuntCalled = true
	s.createHuntName = title
	return domain.Hunt{ID: "new-id", Title: title, Status: "active"}, s.createHuntErr
}
func (s *stubService) GetHunt(ctx context.Context, id string) (domain.Hunt, error) {
	return domain.Hunt{}, nil
}
func (s *stubService) CloseHunt(ctx context.Context, id string) (domain.Hunt, error) {
	s.closeHuntCalled = true
	s.closeHuntID = id
	return s.closedHunt, s.closeHuntErr
}
func (s *stubService) ListApplications(ctx context.Context, huntID string) ([]domain.Application, error) {
	return nil, nil
}
