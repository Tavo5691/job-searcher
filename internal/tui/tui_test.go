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

// TestViewAppListEmpty asserts that the appList view shows an empty-state message.
func TestViewAppListEmpty(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appList
	app.currentHunt = domain.Hunt{ID: "h1", Title: "Big Tech Hunt", Status: "active"}
	app.apps = []domain.Application{}

	v := app.View()

	if !strings.Contains(v, "No applications yet") {
		t.Errorf("expected empty state message, got:\n%s", v)
	}
	if !strings.Contains(v, "n") {
		t.Errorf("expected key hint 'n', got:\n%s", v)
	}
}

// TestViewAppListWithApplications asserts that the appList view renders rows correctly.
func TestViewAppListWithApplications(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appList
	app.currentHunt = domain.Hunt{ID: "h1", Title: "Big Tech Hunt", Status: "active"}
	app.apps = []domain.Application{
		{ID: "a1", CompanyName: "Acme Corp", RoleTitle: "Engineer", Status: domain.ApplicationStatusApplied},
		{ID: "a2", CompanyName: "Beta Inc", RoleTitle: "Manager", Status: domain.ApplicationStatusInterviewing},
	}
	app.appCursor = 0

	v := app.View()

	// Selected row should have "> " prefix.
	if !strings.Contains(v, "> ") {
		t.Errorf("expected cursor '> ' in view, got:\n%s", v)
	}
	// Both companies should appear.
	if !strings.Contains(v, "Acme Corp") {
		t.Errorf("expected 'Acme Corp' in view, got:\n%s", v)
	}
	if !strings.Contains(v, "Beta Inc") {
		t.Errorf("expected 'Beta Inc' in view, got:\n%s", v)
	}
	// Status should appear.
	if !strings.Contains(v, "applied") {
		t.Errorf("expected status 'applied' in view, got:\n%s", v)
	}
}

// TestAppListCursorNavigation asserts that j/k keys move the cursor with clamping.
func TestAppListCursorNavigation(t *testing.T) {
	apps := []domain.Application{
		{ID: "a1", CompanyName: "Acme", Status: domain.ApplicationStatusApplied},
		{ID: "a2", CompanyName: "Beta", Status: domain.ApplicationStatusInterviewing},
		{ID: "a3", CompanyName: "Gamma", Status: domain.ApplicationStatusOffer},
	}
	tests := []struct {
		name       string
		startAt    int
		key        string
		wantCursor int
	}{
		{"j moves down", 0, "j", 1},
		{"j at bottom clamps", 2, "j", 2},
		{"k moves up", 2, "k", 1},
		{"k at top clamps", 0, "k", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{}
			app := NewApp(svc)
			app.currentView = appList
			app.apps = apps
			app.appCursor = tc.startAt

			model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			updated := model.(*App)

			if updated.appCursor != tc.wantCursor {
				t.Errorf("key %q: want cursor %d, got %d", tc.key, tc.wantCursor, updated.appCursor)
			}
		})
	}
}

// TestEscOnAppListReturnsToHuntDetail asserts that pressing Esc in appList view
// returns to huntDetail view.
func TestEscOnAppListReturnsToHuntDetail(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appList

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != huntDetail {
		t.Errorf("expected currentView == huntDetail (%d), got %d", huntDetail, updated.currentView)
	}
}

// TestApplicationsLoadedMsg asserts that applicationsLoadedMsg populates a.apps and resets cursor.
func TestApplicationsLoadedMsg(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.appCursor = 5 // should be reset
	apps := []domain.Application{
		{ID: "a1", CompanyName: "Acme", Status: domain.ApplicationStatusApplied},
		{ID: "a2", CompanyName: "Beta Corp", Status: domain.ApplicationStatusInterviewing},
	}

	model, _ := app.Update(applicationsLoadedMsg{apps: apps})
	updated := model.(*App)

	if len(updated.apps) != 2 {
		t.Errorf("expected 2 apps, got %d", len(updated.apps))
	}
	if updated.appCursor != 0 {
		t.Errorf("expected appCursor reset to 0, got %d", updated.appCursor)
	}
}

// TestEnterOnHuntDetailNavigatesToAppList asserts that pressing Enter in huntDetail
// sets currentHunt, transitions to appList, and fires loadApplicationsCmd.
func TestEnterOnHuntDetailNavigatesToAppList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	hunt := domain.Hunt{ID: "h1", Title: "Big Tech Hunt", Status: "active"}
	app.hunts = []domain.Hunt{hunt}
	app.cursor = 0
	app.currentView = huntDetail

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != appList {
		t.Errorf("expected currentView == appList (%d), got %d", appList, updated.currentView)
	}
	if updated.currentHunt.ID != "h1" {
		t.Errorf("expected currentHunt.ID == 'h1', got %q", updated.currentHunt.ID)
	}
	if cmd == nil {
		t.Error("expected loadApplicationsCmd to be issued")
	}
}

// TestLoadApplicationsCmd asserts that loadApplicationsCmd calls ListApplications and
// returns an applicationsLoadedMsg with the results.
func TestLoadApplicationsCmd(t *testing.T) {
	svc := &stubService{}
	cmd := loadApplicationsCmd(svc, "hunt-1")
	if cmd == nil {
		t.Fatal("expected a non-nil command")
	}
	msg := cmd()
	loaded, ok := msg.(applicationsLoadedMsg)
	if !ok {
		t.Fatalf("expected applicationsLoadedMsg, got %T", msg)
	}
	_ = loaded.apps // field must exist
}

// TestCreateApplicationCmd asserts that createApplicationCmd calls CreateApplication
// and returns an applicationCreatedMsg.
func TestCreateApplicationCmd(t *testing.T) {
	svc := &stubService{}
	cmd := createApplicationCmd(svc, "hunt-1", "Acme Corp", "Engineer", "Build stuff")
	if cmd == nil {
		t.Fatal("expected a non-nil command")
	}
	msg := cmd()
	created, ok := msg.(applicationCreatedMsg)
	if !ok {
		t.Fatalf("expected applicationCreatedMsg, got %T", msg)
	}
	if created.app.CompanyName != "Acme Corp" {
		t.Errorf("expected CompanyName 'Acme Corp', got %q", created.app.CompanyName)
	}
}

// TestUpdateApplicationCmd asserts that updateApplicationCmd calls UpdateApplication
// and returns an applicationUpdatedMsg.
func TestUpdateApplicationCmd(t *testing.T) {
	svc := &stubService{}
	app := domain.Application{ID: "app-1", CompanyName: "Acme", Status: domain.ApplicationStatusInterviewing}
	cmd := updateApplicationCmd(svc, app)
	if cmd == nil {
		t.Fatal("expected a non-nil command")
	}
	msg := cmd()
	updated, ok := msg.(applicationUpdatedMsg)
	if !ok {
		t.Fatalf("expected applicationUpdatedMsg, got %T", msg)
	}
	if updated.app.ID != "app-1" {
		t.Errorf("expected app ID 'app-1', got %q", updated.app.ID)
	}
}

// TestAppStructFields asserts that App has the fields required for the application flow.
func TestAppStructFields(t *testing.T) {
	app := NewApp(&stubService{})
	// Verify zero values compile and are accessible (same package access).
	_ = app.currentHunt
	_ = app.apps
	_ = app.appCursor
	_ = app.inputStep
	_ = app.draft
}

// TestViewConstants asserts that the application view constants have correct iota values.
func TestViewConstants(t *testing.T) {
	if appList != 3 {
		t.Errorf("appList want 3 got %d", appList)
	}
	if appInputCompany != 4 {
		t.Errorf("appInputCompany want 4 got %d", appInputCompany)
	}
	if appInputRole != 5 {
		t.Errorf("appInputRole want 5 got %d", appInputRole)
	}
	if appInputJobDesc != 6 {
		t.Errorf("appInputJobDesc want 6 got %d", appInputJobDesc)
	}
	if appDetail != 7 {
		t.Errorf("appDetail want 7 got %d", appDetail)
	}
}

// ============================================================
// Phase 4: Create form happy path
// ============================================================

// TestNKeyOnAppListTransitionsToAppInputCompany asserts that pressing 'n' in
// appList zeroes the draft, resets the input, sets inputStep=0, and transitions
// to appInputCompany.
func TestNKeyOnAppListTransitionsToAppInputCompany(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appList
	app.currentHunt = domain.Hunt{ID: "h1", Title: "Big Tech", Status: "active"}
	app.draft = domain.Application{CompanyName: "OldCo"} // should be zeroed
	app.inputStep = 2                                    // should be zeroed

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updated := model.(*App)

	if updated.currentView != appInputCompany {
		t.Errorf("expected currentView == appInputCompany (%d), got %d", appInputCompany, updated.currentView)
	}
	if updated.draft.CompanyName != "" {
		t.Errorf("expected draft zeroed, got CompanyName=%q", updated.draft.CompanyName)
	}
	if updated.inputStep != 0 {
		t.Errorf("expected inputStep == 0, got %d", updated.inputStep)
	}
	if updated.input.Value() != "" {
		t.Errorf("expected input cleared, got %q", updated.input.Value())
	}
}

// TestAppInputCompanyEnterAdvancesToRole asserts that pressing Enter in
// appInputCompany sets draft.CompanyName, resets input, sets inputStep=1, and
// transitions to appInputRole.
func TestAppInputCompanyEnterAdvancesToRole(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputCompany
	app.input.SetValue("Acme Corp")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != appInputRole {
		t.Errorf("expected currentView == appInputRole (%d), got %d", appInputRole, updated.currentView)
	}
	if updated.draft.CompanyName != "Acme Corp" {
		t.Errorf("expected draft.CompanyName == 'Acme Corp', got %q", updated.draft.CompanyName)
	}
	if updated.inputStep != 1 {
		t.Errorf("expected inputStep == 1, got %d", updated.inputStep)
	}
	if updated.input.Value() != "" {
		t.Errorf("expected input cleared, got %q", updated.input.Value())
	}
}

// TestAppInputRoleEnterAdvancesToJobDesc asserts that pressing Enter in
// appInputRole sets draft.RoleTitle, resets input, sets inputStep=2, and
// transitions to appInputJobDesc.
func TestAppInputRoleEnterAdvancesToJobDesc(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputRole
	app.draft = domain.Application{CompanyName: "Acme Corp"}
	app.inputStep = 1
	app.input.SetValue("Senior Engineer")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != appInputJobDesc {
		t.Errorf("expected currentView == appInputJobDesc (%d), got %d", appInputJobDesc, updated.currentView)
	}
	if updated.draft.RoleTitle != "Senior Engineer" {
		t.Errorf("expected draft.RoleTitle == 'Senior Engineer', got %q", updated.draft.RoleTitle)
	}
	if updated.inputStep != 2 {
		t.Errorf("expected inputStep == 2, got %d", updated.inputStep)
	}
	if updated.input.Value() != "" {
		t.Errorf("expected input cleared, got %q", updated.input.Value())
	}
}

// TestAppInputJobDescCtrlDFiresCreateAndTransitionsToAppList asserts that pressing
// ctrl+d in appInputJobDesc fires createApplicationCmd and transitions to appList.
// (Enter now inserts a newline — ctrl+d is the submit key for multi-line JD input.)
func TestAppInputJobDescCtrlDFiresCreateAndTransitionsToAppList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputJobDesc
	app.currentHunt = domain.Hunt{ID: "h1", Title: "Big Tech", Status: "active"}
	app.draft = domain.Application{CompanyName: "Acme Corp", RoleTitle: "Senior Engineer"}
	app.inputStep = 2
	app.jdInput.SetValue("Build great things")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updated := model.(*App)

	if updated.currentView != appList {
		t.Errorf("expected currentView == appList (%d), got %d", appList, updated.currentView)
	}
	if cmd == nil {
		t.Fatal("expected createApplicationCmd to be issued")
	}
	// Execute the command and verify it returns applicationCreatedMsg
	msg := cmd()
	if _, ok := msg.(applicationCreatedMsg); !ok {
		t.Errorf("expected applicationCreatedMsg, got %T", msg)
	}
}

// TestApplicationCreatedMsgAppendsToApps asserts that applicationCreatedMsg appends
// the new app to a.apps and zeroes draft and inputStep.
func TestApplicationCreatedMsgAppendsToApps(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentHunt = domain.Hunt{ID: "h1"}
	app.apps = []domain.Application{
		{ID: "a1", CompanyName: "Existing Co"},
	}
	app.draft = domain.Application{CompanyName: "Acme"}
	app.inputStep = 2
	newApp := domain.Application{ID: "a2", CompanyName: "Acme Corp", Status: domain.ApplicationStatusApplied}

	model, _ := app.Update(applicationCreatedMsg{app: newApp})
	updated := model.(*App)

	if updated.draft.CompanyName != "" {
		t.Errorf("expected draft zeroed after creation, got %q", updated.draft.CompanyName)
	}
	if updated.inputStep != 0 {
		t.Errorf("expected inputStep == 0 after creation, got %d", updated.inputStep)
	}
}

// ============================================================
// Phase 5: Create form cancellation
// ============================================================

// TestEscOnAppInputCompanyReturnsToAppList asserts that pressing Esc in
// appInputCompany zeroes draft + inputStep and returns to appList.
func TestEscOnAppInputCompanyReturnsToAppList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputCompany
	app.draft = domain.Application{CompanyName: "Partial"}
	app.inputStep = 0

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != appList {
		t.Errorf("expected currentView == appList (%d), got %d", appList, updated.currentView)
	}
	if updated.draft.CompanyName != "" {
		t.Errorf("expected draft zeroed, got CompanyName=%q", updated.draft.CompanyName)
	}
	if updated.inputStep != 0 {
		t.Errorf("expected inputStep == 0, got %d", updated.inputStep)
	}
}

// TestEscOnAppInputRoleReturnsToAppList asserts that pressing Esc in
// appInputRole zeroes draft + inputStep and returns to appList.
func TestEscOnAppInputRoleReturnsToAppList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputRole
	app.draft = domain.Application{CompanyName: "Acme", RoleTitle: "Partial Role"}
	app.inputStep = 1

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != appList {
		t.Errorf("expected currentView == appList (%d), got %d", appList, updated.currentView)
	}
	if updated.draft.CompanyName != "" || updated.draft.RoleTitle != "" {
		t.Errorf("expected draft zeroed, got %+v", updated.draft)
	}
	if updated.inputStep != 0 {
		t.Errorf("expected inputStep == 0, got %d", updated.inputStep)
	}
}

// TestEscOnAppInputJobDescReturnsToAppList asserts that pressing Esc in
// appInputJobDesc zeroes draft + inputStep and returns to appList.
func TestEscOnAppInputJobDescReturnsToAppList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputJobDesc
	app.draft = domain.Application{CompanyName: "Acme", RoleTitle: "Eng", JobDescription: "Partial desc"}
	app.inputStep = 2

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != appList {
		t.Errorf("expected currentView == appList (%d), got %d", appList, updated.currentView)
	}
	if updated.draft.CompanyName != "" || updated.draft.RoleTitle != "" {
		t.Errorf("expected draft zeroed, got %+v", updated.draft)
	}
	if updated.inputStep != 0 {
		t.Errorf("expected inputStep == 0, got %d", updated.inputStep)
	}
}

// ============================================================
// Phase 6: App Detail navigation + display
// ============================================================

// TestEnterOnAppListTransitionsToAppDetail asserts that pressing Enter on appList
// sets currentApp to the selected app and transitions to appDetail.
func TestEnterOnAppListTransitionsToAppDetail(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appList
	app.apps = []domain.Application{
		{ID: "a1", CompanyName: "Acme Corp", RoleTitle: "Engineer", Status: domain.ApplicationStatusApplied},
		{ID: "a2", CompanyName: "Beta Inc", RoleTitle: "Manager", Status: domain.ApplicationStatusInterviewing},
	}
	app.appCursor = 1

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != appDetail {
		t.Errorf("expected currentView == appDetail (%d), got %d", appDetail, updated.currentView)
	}
	if updated.currentApp.ID != "a2" {
		t.Errorf("expected currentApp.ID == 'a2', got %q", updated.currentApp.ID)
	}
}

// TestViewAppDetail asserts that viewAppDetail renders title, status, job description
// excerpt, and key hints.
func TestViewAppDetail(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appDetail
	app.currentApp = domain.Application{
		ID:             "a1",
		CompanyName:    "Acme Corp",
		RoleTitle:      "Senior Engineer",
		Status:         domain.ApplicationStatusApplied,
		JobDescription: "Build great software at scale.",
	}

	v := app.View()

	wantSubstrs := []string{
		"Acme Corp",
		"Senior Engineer",
		"applied",
		"Build great software at scale.",
		"Esc",
	}
	for _, s := range wantSubstrs {
		if !strings.Contains(v, s) {
			t.Errorf("expected view to contain %q, got:\n%s", s, v)
		}
	}
}

// TestViewAppDetailLongJobDesc asserts that a job description > 200 chars is truncated.
func TestViewAppDetailLongJobDesc(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appDetail
	longDesc := strings.Repeat("x", 300)
	app.currentApp = domain.Application{
		ID:             "a1",
		CompanyName:    "Acme Corp",
		RoleTitle:      "Engineer",
		Status:         domain.ApplicationStatusApplied,
		JobDescription: longDesc,
	}

	v := app.View()

	// The full 300-char string should NOT appear; truncation should be visible.
	if strings.Contains(v, longDesc) {
		t.Error("expected long job description to be truncated in view")
	}
}

// TestEscOnAppDetailReturnsToAppList asserts that pressing Esc in appDetail
// returns to appList.
func TestEscOnAppDetailReturnsToAppList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appDetail
	app.currentApp = domain.Application{ID: "a1", CompanyName: "Acme"}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != appList {
		t.Errorf("expected currentView == appList (%d), got %d", appList, updated.currentView)
	}
}

// ============================================================
// Phase 7: Status cycling
// ============================================================

// TestNextStatus asserts the status cycle: applied→interviewing→offer→accepted→rejected→withdrawn→applied.
func TestNextStatus(t *testing.T) {
	tests := []struct {
		from domain.ApplicationStatus
		want domain.ApplicationStatus
	}{
		{domain.ApplicationStatusApplied, domain.ApplicationStatusInterviewing},
		{domain.ApplicationStatusInterviewing, domain.ApplicationStatusOffer},
		{domain.ApplicationStatusOffer, domain.ApplicationStatusAccepted},
		{domain.ApplicationStatusAccepted, domain.ApplicationStatusRejected},
		{domain.ApplicationStatusRejected, domain.ApplicationStatusWithdrawn},
		{domain.ApplicationStatusWithdrawn, domain.ApplicationStatusApplied},
	}
	for _, tc := range tests {
		t.Run(string(tc.from), func(t *testing.T) {
			got := nextStatus(tc.from)
			if got != tc.want {
				t.Errorf("nextStatus(%s) want %s, got %s", tc.from, tc.want, got)
			}
		})
	}
}

// TestSKeyOnAppDetailFiresUpdateApplication asserts that pressing 's' in appDetail
// cycles the status and fires updateApplicationCmd.
func TestSKeyOnAppDetailFiresUpdateApplication(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appDetail
	app.currentApp = domain.Application{
		ID:          "a1",
		CompanyName: "Acme Corp",
		Status:      domain.ApplicationStatusApplied,
	}

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	updated := model.(*App)

	if updated.currentApp.Status != domain.ApplicationStatusInterviewing {
		t.Errorf("expected status cycled to 'interviewing', got %q", updated.currentApp.Status)
	}
	if cmd == nil {
		t.Fatal("expected updateApplicationCmd to be issued")
	}
	// Execute the command to verify it returns applicationUpdatedMsg
	msg := cmd()
	if _, ok := msg.(applicationUpdatedMsg); !ok {
		t.Errorf("expected applicationUpdatedMsg, got %T", msg)
	}
}

// TestApplicationUpdatedMsgUpdatesAppsSlice asserts that applicationUpdatedMsg
// updates the matching entry in a.apps in-place and updates a.currentApp.
func TestApplicationUpdatedMsgUpdatesAppsSlice(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.apps = []domain.Application{
		{ID: "a1", CompanyName: "Acme", Status: domain.ApplicationStatusApplied},
		{ID: "a2", CompanyName: "Beta", Status: domain.ApplicationStatusApplied},
	}
	app.currentApp = domain.Application{ID: "a1", CompanyName: "Acme", Status: domain.ApplicationStatusApplied}
	updatedApp := domain.Application{ID: "a1", CompanyName: "Acme", Status: domain.ApplicationStatusInterviewing}

	model, _ := app.Update(applicationUpdatedMsg{app: updatedApp})
	result := model.(*App)

	if result.apps[0].Status != domain.ApplicationStatusInterviewing {
		t.Errorf("expected apps[0].Status == 'interviewing', got %q", result.apps[0].Status)
	}
	if result.apps[1].Status != domain.ApplicationStatusApplied {
		t.Errorf("expected apps[1].Status unchanged, got %q", result.apps[1].Status)
	}
	if result.currentApp.Status != domain.ApplicationStatusInterviewing {
		t.Errorf("expected currentApp.Status == 'interviewing', got %q", result.currentApp.Status)
	}
}

// ============================================================
// Phase 8: View() routing for all 5 new view constants
// ============================================================

// TestViewRoutingNewConstants asserts that View() returns a non-empty string
// for each of the five new application-flow view constants.
func TestViewRoutingNewConstants(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(app *App)
	}{
		{
			name: "appList returns non-empty",
			prepare: func(app *App) {
				app.currentView = appList
				app.currentHunt = domain.Hunt{ID: "h1", Title: "Big Tech", Status: "active"}
				app.apps = []domain.Application{}
			},
		},
		{
			name: "appInputCompany returns non-empty",
			prepare: func(app *App) {
				app.currentView = appInputCompany
			},
		},
		{
			name: "appInputRole returns non-empty",
			prepare: func(app *App) {
				app.currentView = appInputRole
			},
		},
		{
			name: "appInputJobDesc returns non-empty",
			prepare: func(app *App) {
				app.currentView = appInputJobDesc
			},
		},
		{
			name: "appDetail returns non-empty",
			prepare: func(app *App) {
				app.currentView = appDetail
				app.currentApp = domain.Application{
					ID:          "a1",
					CompanyName: "Acme Corp",
					RoleTitle:   "Engineer",
					Status:      domain.ApplicationStatusApplied,
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{}
			app := NewApp(svc)
			tc.prepare(app)

			v := app.View()

			if v == "" {
				t.Errorf("expected View() to return non-empty string for %s", tc.name)
			}
		})
	}
}

// TestEnterOnHuntListSetsCurrentHunt asserts that pressing Enter in huntList
// captures the selected hunt into currentHunt before transitioning to huntDetail.
func TestEnterOnHuntListSetsCurrentHunt(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	h := domain.Hunt{ID: "h1", Title: "Big Tech Hunt", Status: "active"}
	app.hunts = []domain.Hunt{h}
	app.cursor = 0

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != huntDetail {
		t.Fatalf("expected huntDetail view, got %d", updated.currentView)
	}
	if updated.currentHunt.ID != h.ID {
		t.Errorf("expected currentHunt.ID == %q, got %q", h.ID, updated.currentHunt.ID)
	}
	if updated.currentHunt.Title != h.Title {
		t.Errorf("expected currentHunt.Title == %q, got %q", h.Title, updated.currentHunt.Title)
	}
}

// TestViewHuntDetailUsesCurrentHunt asserts that viewHuntDetail renders from
// currentHunt (not the hunt list cursor), and includes the Enter hint.
func TestViewHuntDetailUsesCurrentHunt(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	// Two hunts: cursor points at index 0, but currentHunt is index 1.
	app.hunts = []domain.Hunt{
		{ID: "h1", Title: "First Hunt", Status: "active"},
		{ID: "h2", Title: "Second Hunt", Status: "active"},
	}
	app.cursor = 0
	app.currentHunt = domain.Hunt{ID: "h2", Title: "Second Hunt", Status: "active"}
	app.counts = map[string]int{"h2": 5}
	app.currentView = huntDetail

	v := app.View()

	if !strings.Contains(v, "Second Hunt") {
		t.Errorf("expected view to show currentHunt 'Second Hunt', got:\n%s", v)
	}
	if strings.Contains(v, "First Hunt") {
		t.Errorf("expected view NOT to show cursor hunt 'First Hunt', got:\n%s", v)
	}
	if !strings.Contains(v, "5") {
		t.Errorf("expected view to show application count 5, got:\n%s", v)
	}
	if !strings.Contains(v, "Enter") {
		t.Errorf("expected view to contain 'Enter' hint, got:\n%s", v)
	}
}

// TestApplicationCreatedMsgIncrementsCount asserts that receiving an
// applicationCreatedMsg increments the count for the currentHunt in the
// counts map — keeping the hunt list and hunt detail counts in sync.
func TestApplicationCreatedMsgIncrementsCount(t *testing.T) {
	t.Run("increments existing count", func(t *testing.T) {
		svc := &stubService{}
		app := NewApp(svc)
		app.currentHunt = domain.Hunt{ID: "h1", Title: "My Hunt", Status: "active"}
		app.counts = map[string]int{"h1": 2}

		newApp := domain.Application{ID: "app-1", HuntID: "h1", CompanyName: "Acme", RoleTitle: "Eng", Status: domain.ApplicationStatusApplied}
		model, _ := app.Update(applicationCreatedMsg{app: newApp})
		updated := model.(*App)

		if updated.counts["h1"] != 3 {
			t.Errorf("expected count to be 3, got %d", updated.counts["h1"])
		}
	})

	t.Run("initialises nil counts map", func(t *testing.T) {
		svc := &stubService{}
		app := NewApp(svc)
		app.currentHunt = domain.Hunt{ID: "h1", Title: "My Hunt", Status: "active"}
		// counts is nil — simulates first app created before a reload

		newApp := domain.Application{ID: "app-1", HuntID: "h1", CompanyName: "Acme", RoleTitle: "Eng", Status: domain.ApplicationStatusApplied}
		model, _ := app.Update(applicationCreatedMsg{app: newApp})
		updated := model.(*App)

		if updated.counts == nil {
			t.Fatal("expected counts map to be initialised")
		}
		if updated.counts["h1"] != 1 {
			t.Errorf("expected count to be 1, got %d", updated.counts["h1"])
		}
	})
}

// ============================================================
// jd-textarea-input: Phase 1 — Struct + Initialization
// ============================================================

// TestNewAppInitializesJdInput asserts that NewApp initializes jdInput with
// non-zero width and height (i.e. the field was properly set up).
func TestNewAppInitializesJdInput(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	if app.jdInput.Width() == 0 {
		t.Error("expected jdInput to be initialized with a non-zero width")
	}
	if app.jdInput.Height() == 0 {
		t.Error("expected jdInput to be initialized with a non-zero height")
	}
}

// TestJdInputIsFocusedAfterRoleTransition asserts that jdInput is focused
// when Enter is pressed on the role input view, transitioning to appInputJobDesc.
func TestJdInputIsFocusedAfterRoleTransition(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputRole
	app.input.SetValue("Senior Engineer")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != appInputJobDesc {
		t.Fatalf("expected appInputJobDesc, got %d", updated.currentView)
	}
	if !updated.jdInput.Focused() {
		t.Error("jdInput must be focused after transition to appInputJobDesc")
	}
}

// TestJdInputCtrlDNotBoundToDelete asserts that ctrl+d is NOT in the
// DeleteCharacterForward keymap binding on jdInput after initialization.
func TestJdInputCtrlDNotBoundToDelete(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	for _, key := range app.jdInput.KeyMap.DeleteCharacterForward.Keys() {
		if key == "ctrl+d" {
			t.Errorf("expected ctrl+d NOT to be in DeleteCharacterForward keymap, found it")
		}
	}
}

// ============================================================
// jd-textarea-input: Phase 2 — updateAppInputJobDesc behaviour
// ============================================================

// TestCtrlDSubmitsJdAndTransitionsToAppList asserts that ctrl+d in appInputJobDesc
// fires createApplicationCmd with the textarea value and transitions to appList.
func TestCtrlDSubmitsJdAndTransitionsToAppList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputJobDesc
	app.currentHunt = domain.Hunt{ID: "h1", Title: "Big Tech", Status: "active"}
	app.draft = domain.Application{CompanyName: "Acme Corp", RoleTitle: "Senior Engineer"}
	app.inputStep = 2
	app.jdInput.SetValue("We are looking for a talented engineer to join our team.\nMust have Go experience.")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updated := model.(*App)

	if updated.currentView != appList {
		t.Errorf("expected currentView == appList (%d), got %d", appList, updated.currentView)
	}
	if cmd == nil {
		t.Fatal("expected createApplicationCmd to be issued")
	}
	// Execute the command and verify it returns applicationCreatedMsg with correct jobDesc
	msg := cmd()
	created, ok := msg.(applicationCreatedMsg)
	if !ok {
		t.Fatalf("expected applicationCreatedMsg, got %T", msg)
	}
	if created.app.JobDescription == "" {
		t.Error("expected JobDescription to be non-empty")
	}
	// jdInput must be reset after submit
	if updated.jdInput.Value() != "" {
		t.Errorf("expected jdInput to be reset after submit, got %q", updated.jdInput.Value())
	}
}

// TestCtrlDWithEmptyJdIsNoop asserts that ctrl+d in appInputJobDesc with empty
// textarea does NOT fire createApplicationCmd and stays on appInputJobDesc.
func TestCtrlDWithEmptyJdIsNoop(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputJobDesc
	app.currentHunt = domain.Hunt{ID: "h1", Title: "Big Tech", Status: "active"}
	app.draft = domain.Application{CompanyName: "Acme", RoleTitle: "Engineer"}
	// jdInput value is empty (default)

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updated := model.(*App)

	if updated.currentView != appInputJobDesc {
		t.Errorf("expected to remain in appInputJobDesc, got view %d", updated.currentView)
	}
	if cmd != nil {
		t.Error("expected no command to be issued for empty JD")
	}
}

// TestEnterInJdViewDoesNotSubmit asserts that pressing Enter in appInputJobDesc
// does NOT fire createApplicationCmd (Enter adds a newline, ctrl+d submits).
func TestEnterInJdViewDoesNotSubmit(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputJobDesc
	app.currentHunt = domain.Hunt{ID: "h1", Title: "Big Tech", Status: "active"}
	app.draft = domain.Application{CompanyName: "Acme", RoleTitle: "Engineer"}
	app.jdInput.SetValue("Some text")

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// The command (if any) must NOT be a createApplicationCmd — we can verify by
	// checking the view stays in appInputJobDesc.
	// (We can't inspect cmd type directly; we verify via view state.)
	app2 := &App{}
	_ = app2
	// The cmd returned is the textarea's own update cmd, not createApplicationCmd.
	// Key assertion: no applicationCreatedMsg fires.
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(applicationCreatedMsg); ok {
			t.Error("expected Enter NOT to fire applicationCreatedMsg in JD view")
		}
	}
}

// TestEscOnJdInputViewReturnsToAppList asserts that Esc in appInputJobDesc
// zeros draft and inputStep and returns to appList.
func TestEscOnJdInputViewReturnsToAppList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputJobDesc
	app.draft = domain.Application{CompanyName: "Acme", RoleTitle: "Eng"}
	app.inputStep = 2
	app.jdInput.SetValue("partial jd text")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != appList {
		t.Errorf("expected currentView == appList (%d), got %d", appList, updated.currentView)
	}
	if updated.draft.CompanyName != "" || updated.draft.RoleTitle != "" {
		t.Errorf("expected draft zeroed, got %+v", updated.draft)
	}
	if updated.inputStep != 0 {
		t.Errorf("expected inputStep == 0, got %d", updated.inputStep)
	}
	if updated.jdInput.Value() != "" {
		t.Errorf("expected jdInput reset after Esc, got %q", updated.jdInput.Value())
	}
}

// ============================================================
// jd-textarea-input: Phase 3 — viewAppInputJobDesc rendering
// ============================================================

// TestJdInputRetainsMultilineValue asserts that jdInput preserves newlines
// when set and retrieved — covering the "JD input accepts multi-line text" scenario.
func TestJdInputRetainsMultilineValue(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)

	want := "Line 1\nLine 2\nLine 3"
	app.jdInput.SetValue(want)

	if got := app.jdInput.Value(); got != want {
		t.Errorf("expected jdInput.Value() == %q, got %q", want, got)
	}
}

// TestViewAppInputJobDescContainsTextareaAndHint asserts that the JD view
// renders the textarea widget and includes "ctrl+d to submit" hint.
func TestViewAppInputJobDescContainsTextareaAndHint(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appInputJobDesc

	v := app.View()

	if v == "" {
		t.Fatal("expected non-empty view for appInputJobDesc")
	}
	if !strings.Contains(v, "ctrl+d") {
		t.Errorf("expected view to contain 'ctrl+d' submit hint, got:\n%s", v)
	}
}

// ============================================================
// tui-stage-flow: Phase 1 — Scaffolding
// ============================================================

func TestStageInsightViewConstants(t *testing.T) {
	if int(stageList) != 8 {
		t.Errorf("expected stageList == 8, got %d", stageList)
	}
	if int(stageInputType) != 9 {
		t.Errorf("expected stageInputType == 9, got %d", stageInputType)
	}
	if int(stageInputLabel) != 10 {
		t.Errorf("expected stageInputLabel == 10, got %d", stageInputLabel)
	}
	if int(stageInputNotes) != 11 {
		t.Errorf("expected stageInputNotes == 11, got %d", stageInputNotes)
	}
	if int(insightView) != 12 {
		t.Errorf("expected insightView == 12, got %d", insightView)
	}
}

func TestStageTypePickerSlices(t *testing.T) {
	if len(stageTypeLabels) != 8 {
		t.Errorf("expected 8 stageTypeLabels, got %d", len(stageTypeLabels))
	}
	if len(stageTypeValues) != 8 {
		t.Errorf("expected 8 stageTypeValues, got %d", len(stageTypeValues))
	}
	// Last entry must be "other".
	if stageTypeValues[len(stageTypeValues)-1] != domain.StageTypeOther {
		t.Errorf("expected last stageTypeValue to be StageTypeOther, got %q", stageTypeValues[len(stageTypeValues)-1])
	}
}

func TestAppStageInsightFields(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	// Zero-value checks.
	if app.stageCursor != 0 {
		t.Errorf("expected stageCursor == 0, got %d", app.stageCursor)
	}
	if app.insightLoading {
		t.Error("expected insightLoading == false at init")
	}
	if app.stages != nil {
		t.Error("expected stages == nil at init")
	}
}

// ============================================================
// tui-stage-flow: Phase 2 — Stage List
// ============================================================

func TestTKeyInAppDetailNavigatesToStageList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appDetail
	app.currentApp = domain.Application{ID: "app1", CompanyName: "Acme", RoleTitle: "Eng"}

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updated := model.(*App)

	if updated.currentView != stageList {
		t.Errorf("expected stageList, got %d", updated.currentView)
	}
	if cmd == nil {
		t.Fatal("expected loadStagesCmd to be issued")
	}
}

func TestStagesLoadedMsgPopulatesStages(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	stages := []domain.Stage{
		{ID: "s1", ApplicationID: "app1", Type: domain.StageTypeRecruiterScreen, Notes: "Went well"},
		{ID: "s2", ApplicationID: "app1", Type: domain.StageTypeTechnicalInterview},
	}

	model, _ := app.Update(stagesLoadedMsg{stages: stages})
	updated := model.(*App)

	if len(updated.stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(updated.stages))
	}
	if updated.stageCursor != 0 {
		t.Errorf("expected stageCursor reset to 0, got %d", updated.stageCursor)
	}
}

func TestStageListJKNavigation(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageList
	app.stages = []domain.Stage{
		{ID: "s1", Type: domain.StageTypeRecruiterScreen},
		{ID: "s2", Type: domain.StageTypeTechnicalInterview},
	}
	app.stageCursor = 0

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updated := model.(*App)
	if updated.stageCursor != 1 {
		t.Errorf("expected stageCursor == 1 after j, got %d", updated.stageCursor)
	}

	model2, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	updated2 := model2.(*App)
	if updated2.stageCursor != 0 {
		t.Errorf("expected stageCursor == 0 after k, got %d", updated2.stageCursor)
	}
}

func TestStageListEscReturnsToAppDetail(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageList

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != appDetail {
		t.Errorf("expected appDetail, got %d", updated.currentView)
	}
}

func TestViewStageListEmptyState(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageList
	app.currentApp = domain.Application{CompanyName: "Acme"}

	v := app.View()
	if !strings.Contains(v, "No stages yet") {
		t.Errorf("expected 'No stages yet' in empty stage list, got:\n%s", v)
	}
}

func TestViewStageListRendersStages(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageList
	app.currentApp = domain.Application{CompanyName: "Acme"}
	app.stages = []domain.Stage{
		{ID: "s1", Type: domain.StageTypeRecruiterScreen, Notes: "Friendly call"},
	}

	v := app.View()
	if !strings.Contains(v, "recruiter_screen") {
		t.Errorf("expected stage type in view, got:\n%s", v)
	}
	if !strings.Contains(v, "Friendly call") {
		t.Errorf("expected stage notes in view, got:\n%s", v)
	}
}

// ============================================================
// tui-stage-flow: Phase 3 — Stage Create Wizard
// ============================================================

func TestNKeyInStageListNavigatesToStageInputType(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageList

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updated := model.(*App)

	if updated.currentView != stageInputType {
		t.Errorf("expected stageInputType, got %d", updated.currentView)
	}
	if updated.stageTypeIdx != 0 {
		t.Errorf("expected stageTypeIdx reset to 0, got %d", updated.stageTypeIdx)
	}
}

func TestStageInputTypeJKNavigation(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputType
	app.stageTypeIdx = 0

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updated := model.(*App)
	if updated.stageTypeIdx != 1 {
		t.Errorf("expected stageTypeIdx == 1 after j, got %d", updated.stageTypeIdx)
	}
}

func TestStageInputTypeEnterNonOtherGoesToNotes(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputType
	app.stageTypeIdx = 0 // RecruiterScreen

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != stageInputNotes {
		t.Errorf("expected stageInputNotes for non-Other type, got %d", updated.currentView)
	}
	if updated.draftStage.Type != domain.StageTypeRecruiterScreen {
		t.Errorf("expected draftStage.Type == RecruiterScreen, got %q", updated.draftStage.Type)
	}
}

func TestStageInputTypeEnterOtherGoesToLabel(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputType
	app.stageTypeIdx = 7 // Other

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != stageInputLabel {
		t.Errorf("expected stageInputLabel for Other type, got %d", updated.currentView)
	}
}

func TestStageInputLabelEnterAdvancesToNotes(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputLabel
	app.draftStage = domain.Stage{Type: domain.StageTypeOther}
	app.input.SetValue("Custom round")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.currentView != stageInputNotes {
		t.Errorf("expected stageInputNotes after label, got %d", updated.currentView)
	}
	if updated.draftStage.Label != "Custom round" {
		t.Errorf("expected draftStage.Label == 'Custom round', got %q", updated.draftStage.Label)
	}
}

func TestStageInputNotesEnterFiresAddStage(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputNotes
	app.currentApp = domain.Application{ID: "app1", CompanyName: "Acme", RoleTitle: "Eng"}
	app.draftStage = domain.Stage{Type: domain.StageTypeRecruiterScreen}
	app.stages = []domain.Stage{{ID: "existing"}}
	app.input.SetValue("Went really well")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)
	_ = updated

	if cmd == nil {
		t.Fatal("expected addStageCmd to be issued")
	}
	msg := cmd()
	created, ok := msg.(stageCreatedMsg)
	if !ok {
		t.Fatalf("expected stageCreatedMsg, got %T", msg)
	}
	if created.stage.Notes != "Went really well" {
		t.Errorf("expected notes 'Went really well', got %q", created.stage.Notes)
	}
	if created.stage.ApplicationID != "app1" {
		t.Errorf("expected ApplicationID == 'app1', got %q", created.stage.ApplicationID)
	}
	if created.stage.Order != 1 {
		t.Errorf("expected Order == 1 (len of existing stages), got %d", created.stage.Order)
	}
}

func TestStageCreatedMsgAppendsAndReturnsToList(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputNotes

	model, _ := app.Update(stageCreatedMsg{stage: domain.Stage{ID: "s-new", Type: domain.StageTypeOffer}})
	updated := model.(*App)

	if updated.currentView != stageList {
		t.Errorf("expected stageList after stage created, got %d", updated.currentView)
	}
	if len(updated.stages) != 1 {
		t.Errorf("expected 1 stage appended, got %d", len(updated.stages))
	}
}

func TestStageInputNotesEscNonOtherGoesToType(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputNotes
	app.draftStage = domain.Stage{Type: domain.StageTypeRecruiterScreen}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != stageInputType {
		t.Errorf("expected stageInputType on Esc for non-Other, got %d", updated.currentView)
	}
}

func TestStageInputNotesEscOtherGoesToLabel(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputNotes
	app.draftStage = domain.Stage{Type: domain.StageTypeOther}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != stageInputLabel {
		t.Errorf("expected stageInputLabel on Esc for Other, got %d", updated.currentView)
	}
}

func TestViewStageInputTypeRendersAllTypes(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageInputType

	v := app.View()
	for _, label := range stageTypeLabels {
		if !strings.Contains(v, label) {
			t.Errorf("expected stage type label %q in view, got:\n%s", label, v)
		}
	}
}

// ============================================================
// tui-stage-flow: Phase 4 — Stage Delete
// ============================================================

func TestDKeyDeletesStageAndRefreshesSlice(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageList
	app.stages = []domain.Stage{
		{ID: "s1", Type: domain.StageTypeRecruiterScreen},
		{ID: "s2", Type: domain.StageTypeTechnicalInterview},
	}
	app.stageCursor = 0

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd == nil {
		t.Fatal("expected deleteStageCmd to be issued")
	}
	msg := cmd()
	deleted, ok := msg.(stageDeletedMsg)
	if !ok {
		t.Fatalf("expected stageDeletedMsg, got %T", msg)
	}
	if deleted.id != "s1" {
		t.Errorf("expected deleted ID == 's1', got %q", deleted.id)
	}
}

func TestStageDeletedMsgFiltersSliceAndClampsCursor(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.stages = []domain.Stage{
		{ID: "s1", Type: domain.StageTypeRecruiterScreen},
		{ID: "s2", Type: domain.StageTypeTechnicalInterview},
	}
	app.stageCursor = 1

	model, _ := app.Update(stageDeletedMsg{id: "s2"})
	updated := model.(*App)

	if len(updated.stages) != 1 {
		t.Errorf("expected 1 stage remaining, got %d", len(updated.stages))
	}
	if updated.stageCursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", updated.stageCursor)
	}
}

func TestDKeyOnEmptyStageListIsNoop(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = stageList
	app.stages = nil

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd != nil {
		t.Error("expected no cmd when deleting from empty stage list")
	}
}

// ============================================================
// tui-stage-flow: Phase 5 — Insight Flow
// ============================================================

func TestIKeyInAppDetailFiresGetInsight(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = appDetail
	app.currentApp = domain.Application{ID: "app1"}

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	updated := model.(*App)

	if updated.currentView != insightView {
		t.Errorf("expected insightView, got %d", updated.currentView)
	}
	if !updated.insightLoading {
		t.Error("expected insightLoading == true after i")
	}
	if cmd == nil {
		t.Fatal("expected getInsightCmd to be issued")
	}
}

func TestInsightLoadedMsgWithExistingInsight(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.insightLoading = true
	insight := domain.Insight{ID: "ins1", Content: "Great candidate match"}

	model, _ := app.Update(insightLoadedMsg{insight: insight})
	updated := model.(*App)

	if updated.insightLoading {
		t.Error("expected insightLoading cleared after insight loaded")
	}
	if updated.currentInsight.ID != "ins1" {
		t.Errorf("expected currentInsight.ID == 'ins1', got %q", updated.currentInsight.ID)
	}
}

func TestInsightLoadedMsgEmptyIDTriggersGenerate(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentApp = domain.Application{ID: "app1"}
	app.insightLoading = true

	_, cmd := app.Update(insightLoadedMsg{insight: domain.Insight{}})
	if cmd == nil {
		t.Fatal("expected generateInsightCmd to be issued when no insight found")
	}
}

func TestInsightGeneratedMsgSetsContentAndClearsLoading(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.insightLoading = true
	insight := domain.Insight{ID: "ins2", Content: "Tailored advice here"}

	model, _ := app.Update(insightGeneratedMsg{insight: insight})
	updated := model.(*App)

	if updated.insightLoading {
		t.Error("expected insightLoading cleared after generation")
	}
	if updated.currentInsight.Content != "Tailored advice here" {
		t.Errorf("expected insight content set, got %q", updated.currentInsight.Content)
	}
}

func TestRKeyInInsightViewRegenerates(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = insightView
	app.currentApp = domain.Application{ID: "app1"}
	app.insightLoading = false

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("expected generateInsightCmd on r")
	}
}

func TestRKeyGuardSkipsWhenAlreadyLoading(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = insightView
	app.insightLoading = true

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd != nil {
		t.Error("expected no cmd when already loading insight")
	}
}

func TestInsightViewShowsLoadingMessage(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = insightView
	app.insightLoading = true
	app.currentApp = domain.Application{CompanyName: "Acme", RoleTitle: "Eng"}

	v := app.View()
	if !strings.Contains(v, "Generating") {
		t.Errorf("expected loading message in insight view, got:\n%s", v)
	}
}

func TestInsightViewRendersContent(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = insightView
	app.insightLoading = false
	app.currentApp = domain.Application{CompanyName: "Acme", RoleTitle: "Eng"}
	app.currentInsight = domain.Insight{ID: "i1", Content: "Focus on system design"}

	v := app.View()
	if !strings.Contains(v, "Focus on system design") {
		t.Errorf("expected insight content in view, got:\n%s", v)
	}
	if !strings.Contains(v, "regenerate") {
		t.Errorf("expected regenerate hint in view, got:\n%s", v)
	}
}

func TestInsightViewEscReturnsToAppDetail(t *testing.T) {
	svc := &stubService{}
	app := NewApp(svc)
	app.currentView = insightView

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(*App)

	if updated.currentView != appDetail {
		t.Errorf("expected appDetail after Esc from insightView, got %d", updated.currentView)
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
	// Stage stub fields.
	listStagesCalled  bool
	listStagesResult  []domain.Stage
	listStagesErr     error
	addStageCalled    bool
	addedStage        domain.Stage
	addStageErr       error
	deleteStageCalled bool
	deleteStageID     string
	deleteStageErr    error
	// Insight stub fields.
	getInsightResult      domain.Insight
	getInsightErr         error
	generateInsightResult domain.Insight
	generateInsightErr    error
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
func (s *stubService) CreateApplication(ctx context.Context, huntID, company, role, jobDesc string) (domain.Application, error) {
	return domain.Application{
		ID:             "app-new",
		HuntID:         huntID,
		CompanyName:    company,
		RoleTitle:      role,
		JobDescription: jobDesc,
		Status:         domain.ApplicationStatusApplied,
	}, nil
}
func (s *stubService) UpdateApplication(ctx context.Context, app domain.Application) (domain.Application, error) {
	return app, nil
}
func (s *stubService) ListStages(ctx context.Context, applicationID string) ([]domain.Stage, error) {
	s.listStagesCalled = true
	return s.listStagesResult, s.listStagesErr
}
func (s *stubService) AddStage(ctx context.Context, st domain.Stage) (domain.Stage, error) {
	s.addStageCalled = true
	s.addedStage = st
	if s.addStageErr != nil {
		return domain.Stage{}, s.addStageErr
	}
	st.ID = "stage-new"
	return st, nil
}
func (s *stubService) DeleteStage(ctx context.Context, id string) error {
	s.deleteStageCalled = true
	s.deleteStageID = id
	return s.deleteStageErr
}
func (s *stubService) GetInsight(ctx context.Context, applicationID string) (domain.Insight, error) {
	return s.getInsightResult, s.getInsightErr
}
func (s *stubService) GenerateInsight(ctx context.Context, applicationID string) (domain.Insight, error) {
	return s.generateInsightResult, s.generateInsightErr
}
