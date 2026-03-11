// Package tui provides the terminal user interface built with Bubble Tea.
// The TUI calls only the app/ layer — never adapters directly.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Tavo5691/job-searcher/internal/domain"
)

// view is the discriminant for which screen is currently active.
type view int

const (
	huntList        view = iota // 0 — list of hunts
	huntInput                   // 1 — text input to create a new hunt
	huntDetail                  // 2 — detail view of a single hunt
	appList                     // 3 — application list for the current hunt
	appInputCompany             // 4 — text input: company name
	appInputRole                // 5 — text input: role/title
	appInputJobDesc             // 6 — text input: job description
	appDetail                   // 7 — detail view for a single application
	stageList                   // 8 — list of stages for the current application
	stageInputType              // 9 — stage type picker (j/k to select)
	stageInputLabel             // 10 — label input (only when type=other)
	stageInputNotes             // 11 — notes text input
	insightView                 // 12 — LLM-generated insight view
)

// stageTypeLabels and stageTypeValues are parallel slices for the stage type picker.
// Index into one gives the display label; same index into the other gives the domain value.
var stageTypeLabels = []string{
	"Recruiter Screen",
	"Technical Screen",
	"Take-Home Assignment",
	"Technical Interview",
	"Behavioral Interview",
	"System Design",
	"Offer",
	"Other",
}

var stageTypeValues = []domain.StageType{
	domain.StageTypeRecruiterScreen,
	domain.StageTypeTechnicalScreen,
	domain.StageTypeTakeHome,
	domain.StageTypeTechnicalInterview,
	domain.StageTypeBehavioral,
	domain.StageTypeSystemDesign,
	domain.StageTypeOffer,
	domain.StageTypeOther,
}

// serviceIface is the minimal subset of app.Service that the TUI requires.
// Defined here (at the consumer) per idiomatic Go.
type serviceIface interface {
	ListHunts(ctx context.Context) ([]domain.Hunt, error)
	CreateHunt(ctx context.Context, title string) (domain.Hunt, error)
	GetHunt(ctx context.Context, id string) (domain.Hunt, error)
	CloseHunt(ctx context.Context, id string) (domain.Hunt, error)
	ListApplications(ctx context.Context, huntID string) ([]domain.Application, error)
	CreateApplication(ctx context.Context, huntID, company, role, jobDesc string) (domain.Application, error)
	UpdateApplication(ctx context.Context, app domain.Application) (domain.Application, error)
	ListStages(ctx context.Context, applicationID string) ([]domain.Stage, error)
	AddStage(ctx context.Context, st domain.Stage) (domain.Stage, error)
	DeleteStage(ctx context.Context, id string) error
	GetInsight(ctx context.Context, applicationID string) (domain.Insight, error)
	GenerateInsight(ctx context.Context, applicationID string) (domain.Insight, error)
}

// App is the root Bubble Tea model for the job-searcher TUI.
type App struct {
	svc         serviceIface
	hunts       []domain.Hunt
	cursor      int
	err         error
	currentView view
	input       textinput.Model
	jdInput     textarea.Model
	counts      map[string]int
	statusMsg   string
	// Application flow fields.
	currentHunt domain.Hunt
	apps        []domain.Application
	appCursor   int
	inputStep   int
	draft       domain.Application
	currentApp  domain.Application
	// Stage flow fields.
	stages       []domain.Stage
	stageCursor  int
	draftStage   domain.Stage
	stageTypeIdx int
	// Insight flow fields.
	currentInsight domain.Insight
	insightLoading bool
}

// NewApp creates a new App with the given service.
func NewApp(svc serviceIface) *App {
	ti := textinput.New()
	ti.Placeholder = "Hunt name…"
	ti.CharLimit = 100

	ta := textarea.New()
	ta.Placeholder = "Paste or type the job description…"
	ta.SetWidth(60)
	ta.SetHeight(8)
	// Unbind DeleteCharacterForward (ctrl+d) so we can intercept it as the
	// submit key. Without this, ctrl+d would be consumed by the textarea itself.
	ta.KeyMap.DeleteCharacterForward.Unbind()

	return &App{
		svc:     svc,
		input:   ti,
		jdInput: ta,
	}
}

// Run starts the Bubble Tea program.
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("tui run: %w", err)
	}
	return nil
}

// Init is the Bubble Tea initialisation command.
func (a *App) Init() tea.Cmd {
	return loadHuntsAndCountsCmd(a.svc)
}

// Update handles messages and returns updated model + next command.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch a.currentView {
		case huntList:
			return a.updateHuntList(m)
		case huntInput:
			return a.updateHuntInput(m)
		case huntDetail:
			return a.updateHuntDetail(m)
		case appList:
			return a.updateAppList(m)
		case appInputCompany:
			return a.updateAppInputCompany(m)
		case appInputRole:
			return a.updateAppInputRole(m)
		case appInputJobDesc:
			return a.updateAppInputJobDesc(m)
		case appDetail:
			return a.updateAppDetail(m)
		case stageList:
			return a.updateStageList(m)
		case stageInputType:
			return a.updateStageInputType(m)
		case stageInputLabel:
			return a.updateStageInputLabel(m)
		case stageInputNotes:
			return a.updateStageInputNotes(m)
		case insightView:
			return a.updateInsightView(m)
		}
	case stagesLoadedMsg:
		if m.err != nil {
			a.statusMsg = m.err.Error()
		} else {
			a.stages = m.stages
			a.stageCursor = 0
		}
	case stageCreatedMsg:
		if m.err != nil {
			a.statusMsg = m.err.Error()
		} else {
			a.stages = append(a.stages, m.stage)
			a.draftStage = domain.Stage{}
			a.currentView = stageList
		}
	case stageDeletedMsg:
		if m.err != nil {
			a.statusMsg = m.err.Error()
		} else {
			filtered := make([]domain.Stage, 0, len(a.stages))
			for _, s := range a.stages {
				if s.ID != m.id {
					filtered = append(filtered, s)
				}
			}
			a.stages = filtered
			if a.stageCursor >= len(a.stages) && a.stageCursor > 0 {
				a.stageCursor = len(a.stages) - 1
			}
		}
	case insightLoadedMsg:
		if m.err != nil {
			a.statusMsg = m.err.Error()
			a.insightLoading = false
		} else if m.insight.ID == "" {
			// No insight yet — auto-trigger generation.
			return a, generateInsightCmd(a.svc, a.currentApp.ID)
		} else {
			a.currentInsight = m.insight
			a.insightLoading = false
		}
	case insightGeneratedMsg:
		a.insightLoading = false
		if m.err != nil {
			a.statusMsg = m.err.Error()
		} else {
			a.currentInsight = m.insight
		}
	case huntsLoadedMsg:
		a.hunts = m.hunts
		a.counts = m.counts
		a.err = m.err
	case huntCreatedMsg:
		if m.err != nil {
			a.statusMsg = m.err.Error()
		} else {
			a.currentView = huntList
			return a, loadHuntsAndCountsCmd(a.svc)
		}
	case huntClosedMsg:
		if m.err != nil {
			a.statusMsg = m.err.Error()
		} else {
			return a, loadHuntsAndCountsCmd(a.svc)
		}
	case applicationsLoadedMsg:
		a.apps = m.apps
		a.appCursor = 0
	case applicationCreatedMsg:
		a.apps = append(a.apps, m.app)
		a.draft = domain.Application{}
		a.inputStep = 0
		if a.counts == nil {
			a.counts = make(map[string]int)
		}
		a.counts[a.currentHunt.ID]++
	case applicationUpdatedMsg:
		// Refresh the apps list with the updated application.
		for i, ap := range a.apps {
			if ap.ID == m.app.ID {
				a.apps[i] = m.app
				break
			}
		}
		// Update currentApp if it matches the updated application.
		if a.currentApp.ID == m.app.ID {
			a.currentApp = m.app
		}
	case statusMsg:
		a.statusMsg = string(m)
	}
	return a, nil
}

// updateHuntList handles key messages in the hunt list view.
func (a *App) updateHuntList(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.String() {
	case "q", "ctrl+c":
		return a, tea.Quit
	case "j", "down":
		if a.cursor < len(a.hunts)-1 {
			a.cursor++
		}
	case "k", "up":
		if a.cursor > 0 {
			a.cursor--
		}
	case "n":
		a.currentView = huntInput
		a.statusMsg = ""
		a.input.Reset()
		a.input.Focus()
		return a, textinput.Blink
	case "enter":
		if len(a.hunts) > 0 {
			a.currentHunt = a.hunts[a.cursor]
			a.currentView = huntDetail
		}
	case "c":
		if len(a.hunts) > 0 {
			h := a.hunts[a.cursor]
			if h.Status == "closed" {
				a.statusMsg = "Hunt already closed"
			} else {
				return a, closeHuntCmd(a.svc, h.ID)
			}
		}
	}
	return a, nil
}

// updateHuntInput handles key messages in the hunt input view.
func (a *App) updateHuntInput(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(a.input.Value())
		if name == "" {
			return a, nil
		}
		return a, createHuntCmd(a.svc, name)
	case tea.KeyEsc:
		a.currentView = huntList
		a.input.Reset()
		return a, nil
	default:
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(m)
		return a, cmd
	}
}

// updateAppList handles key messages in the application list view.
func (a *App) updateAppList(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.currentView = huntDetail
		return a, nil
	case tea.KeyEnter:
		if len(a.apps) > 0 && a.appCursor < len(a.apps) {
			a.currentApp = a.apps[a.appCursor]
			a.currentView = appDetail
		}
		return a, nil
	}
	switch m.String() {
	case "n":
		a.draft = domain.Application{}
		a.inputStep = 0
		a.input.Reset()
		a.input.Focus()
		a.currentView = appInputCompany
		return a, textinput.Blink
	case "j", "down":
		if a.appCursor < len(a.apps)-1 {
			a.appCursor++
		}
	case "k", "up":
		if a.appCursor > 0 {
			a.appCursor--
		}
	}
	return a, nil
}

// updateAppInputCompany handles key messages in the company name input view.
func (a *App) updateAppInputCompany(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEnter:
		a.draft.CompanyName = a.input.Value()
		a.input.Reset()
		a.inputStep = 1
		a.currentView = appInputRole
		return a, nil
	case tea.KeyEsc:
		a.draft = domain.Application{}
		a.inputStep = 0
		a.input.Reset()
		a.currentView = appList
		return a, nil
	default:
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(m)
		return a, cmd
	}
}

// updateAppInputRole handles key messages in the role title input view.
func (a *App) updateAppInputRole(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEnter:
		a.draft.RoleTitle = a.input.Value()
		a.input.Reset()
		a.inputStep = 2
		a.currentView = appInputJobDesc
		a.jdInput.Reset()
		cmd := a.jdInput.Focus()
		return a, cmd
	case tea.KeyEsc:
		a.draft = domain.Application{}
		a.inputStep = 0
		a.input.Reset()
		a.currentView = appList
		return a, nil
	default:
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(m)
		return a, cmd
	}
}

// updateAppInputJobDesc handles key messages in the job description input view.
// ctrl+d is the submit key; Enter inserts a newline (multi-line textarea).
func (a *App) updateAppInputJobDesc(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyCtrlD:
		jobDesc := strings.TrimSpace(a.jdInput.Value())
		if jobDesc == "" {
			return a, nil
		}
		company := a.draft.CompanyName
		role := a.draft.RoleTitle
		a.jdInput.Reset()
		a.draft = domain.Application{}
		a.inputStep = 0
		a.currentView = appList
		return a, createApplicationCmd(a.svc, a.currentHunt.ID, company, role, jobDesc)
	case tea.KeyEsc:
		a.draft = domain.Application{}
		a.inputStep = 0
		a.jdInput.Reset()
		a.currentView = appList
		return a, nil
	default:
		var cmd tea.Cmd
		a.jdInput, cmd = a.jdInput.Update(m)
		return a, cmd
	}
}

// updateAppDetail handles key messages in the application detail view.
func (a *App) updateAppDetail(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.currentView = appList
		return a, nil
	}
	switch m.String() {
	case "s":
		a.currentApp.Status = nextStatus(a.currentApp.Status)
		return a, updateApplicationCmd(a.svc, a.currentApp)
	case "t":
		a.stageCursor = 0
		a.currentView = stageList
		return a, loadStagesCmd(a.svc, a.currentApp.ID)
	case "i":
		if !a.insightLoading {
			a.insightLoading = true
			a.currentView = insightView
			return a, getInsightCmd(a.svc, a.currentApp.ID)
		}
	}
	return a, nil
}

// updateStageList handles key messages in the stage list view.
func (a *App) updateStageList(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.currentView = appDetail
		return a, nil
	}
	switch m.String() {
	case "j", "down":
		if a.stageCursor < len(a.stages)-1 {
			a.stageCursor++
		}
	case "k", "up":
		if a.stageCursor > 0 {
			a.stageCursor--
		}
	case "n":
		a.draftStage = domain.Stage{}
		a.stageTypeIdx = 0
		a.currentView = stageInputType
	case "d":
		if len(a.stages) > 0 && a.stageCursor < len(a.stages) {
			id := a.stages[a.stageCursor].ID
			return a, deleteStageCmd(a.svc, id)
		}
	}
	return a, nil
}

// updateStageInputType handles key messages in the stage type picker view.
func (a *App) updateStageInputType(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.currentView = stageList
		return a, nil
	case tea.KeyEnter:
		a.draftStage.Type = stageTypeValues[a.stageTypeIdx]
		if a.draftStage.Type == domain.StageTypeOther {
			a.input.Reset()
			a.input.Focus()
			a.currentView = stageInputLabel
			return a, textinput.Blink
		}
		a.input.Reset()
		a.input.Focus()
		a.currentView = stageInputNotes
		return a, textinput.Blink
	}
	switch m.String() {
	case "j", "down":
		a.stageTypeIdx = (a.stageTypeIdx + 1) % len(stageTypeLabels)
	case "k", "up":
		a.stageTypeIdx = (a.stageTypeIdx + len(stageTypeLabels) - 1) % len(stageTypeLabels)
	}
	return a, nil
}

// updateStageInputLabel handles key messages in the stage label input view (type=other).
func (a *App) updateStageInputLabel(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.input.Reset()
		a.currentView = stageInputType
		return a, nil
	case tea.KeyEnter:
		a.draftStage.Label = a.input.Value()
		a.input.Reset()
		a.input.Focus()
		a.currentView = stageInputNotes
		return a, textinput.Blink
	default:
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(m)
		return a, cmd
	}
}

// updateStageInputNotes handles key messages in the stage notes input view.
func (a *App) updateStageInputNotes(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.input.Reset()
		if a.draftStage.Type == domain.StageTypeOther {
			a.currentView = stageInputLabel
		} else {
			a.currentView = stageInputType
		}
		return a, nil
	case tea.KeyEnter:
		a.draftStage.Notes = a.input.Value()
		a.draftStage.ApplicationID = a.currentApp.ID
		a.draftStage.Order = len(a.stages)
		a.input.Reset()
		return a, addStageCmd(a.svc, a.draftStage)
	default:
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(m)
		return a, cmd
	}
}

// updateInsightView handles key messages in the insight view.
func (a *App) updateInsightView(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		a.currentView = appDetail
		return a, nil
	}
	switch m.String() {
	case "r":
		if !a.insightLoading {
			a.insightLoading = true
			return a, generateInsightCmd(a.svc, a.currentApp.ID)
		}
	}
	return a, nil
}

// nextStatus cycles the application status in the canonical order:
// applied → interviewing → offer → accepted → rejected → withdrawn → applied
func nextStatus(s domain.ApplicationStatus) domain.ApplicationStatus {
	cycle := []domain.ApplicationStatus{
		domain.ApplicationStatusApplied,
		domain.ApplicationStatusInterviewing,
		domain.ApplicationStatusOffer,
		domain.ApplicationStatusAccepted,
		domain.ApplicationStatusRejected,
		domain.ApplicationStatusWithdrawn,
	}
	for i, status := range cycle {
		if status == s {
			return cycle[(i+1)%len(cycle)]
		}
	}
	// Unknown status — default to applied.
	return domain.ApplicationStatusApplied
}

// updateHuntDetail handles key messages in the hunt detail view.
func (a *App) updateHuntDetail(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEnter:
		if len(a.hunts) > 0 && a.cursor < len(a.hunts) {
			a.currentHunt = a.hunts[a.cursor]
			a.currentView = appList
			return a, loadApplicationsCmd(a.svc, a.currentHunt.ID)
		}
	case tea.KeyEsc:
		a.currentView = huntList
	}
	return a, nil
}

// View renders the current state as a string.
func (a *App) View() string {
	switch a.currentView {
	case huntInput:
		return a.viewHuntInput()
	case huntDetail:
		return a.viewHuntDetail()
	case appList:
		return a.viewAppList()
	case appInputCompany:
		return a.viewAppInput("Company name")
	case appInputRole:
		return a.viewAppInput("Role / job title")
	case appInputJobDesc:
		return a.viewAppInputJobDesc()
	case appDetail:
		return a.viewAppDetail()
	case stageList:
		return a.viewStageList()
	case stageInputType:
		return a.viewStageInputType()
	case stageInputLabel:
		return a.viewStageInput("Label (describe this stage)")
	case stageInputNotes:
		return a.viewStageInput("Notes")
	case insightView:
		return a.viewInsight()
	default:
		return a.viewHuntList()
	}
}

// viewAppList renders the application list screen for the current hunt.
func (a *App) viewAppList() string {
	title := lipgloss.NewStyle().Bold(true).Render(a.currentHunt.Title + " — Applications")
	if len(a.apps) == 0 {
		return title + "\n\nNo applications yet. Press 'n' to create one.\n\nEsc to go back • n to create"
	}
	var sb strings.Builder
	for i, ap := range a.apps {
		cursor := "  "
		if i == a.appCursor {
			cursor = "> "
		}
		fmt.Fprintf(&sb, "%s%s — %s [%s]\n", cursor, ap.CompanyName, ap.RoleTitle, ap.Status)
	}
	return title + "\n\n" + sb.String() + "\nj/k to move • Enter to open • n to create • Esc to go back"
}

// viewAppInput renders a single-line text input screen with the given prompt label.
func (a *App) viewAppInput(prompt string) string {
	title := lipgloss.NewStyle().Bold(true).Render("New Application — " + prompt)
	return title + "\n\n" + a.input.View() + "\n\nEnter to continue • Esc to cancel"
}

// viewAppInputJobDesc renders the multi-line job description input screen.
func (a *App) viewAppInputJobDesc() string {
	title := lipgloss.NewStyle().Bold(true).Render("New Application — Job description")
	return title + "\n\n" + a.jdInput.View() + "\n\nctrl+d to submit • Esc to cancel"
}

// viewAppDetail renders the application detail screen.
func (a *App) viewAppDetail() string {
	ap := a.currentApp
	title := lipgloss.NewStyle().Bold(true).Render("Application: " + ap.CompanyName + " \u2014 " + ap.RoleTitle)
	desc := ap.JobDescription
	if len(desc) > 200 {
		desc = desc[:200] + "..."
	}
	return fmt.Sprintf("%s\n\nStatus: %s\n\n%s\n\n[s] cycle status  [t] stages  [i] insight  [Esc] back", title, ap.Status, desc)
}

// viewStageList renders the stage list for the current application.
func (a *App) viewStageList() string {
	title := lipgloss.NewStyle().Bold(true).Render(a.currentApp.CompanyName + " — Stages")
	if len(a.stages) == 0 {
		return title + "\n\nNo stages yet. Press 'n' to add one.\n\nn to add • Esc to go back"
	}
	var sb strings.Builder
	for i, st := range a.stages {
		cursor := "  "
		if i == a.stageCursor {
			cursor = "> "
		}
		label := string(st.Type)
		if st.Type == domain.StageTypeOther && st.Label != "" {
			label = st.Label
		}
		notes := st.Notes
		if len(notes) > 40 {
			notes = notes[:40] + "…"
		}
		if notes != "" {
			fmt.Fprintf(&sb, "%s%s — %s\n", cursor, label, notes)
		} else {
			fmt.Fprintf(&sb, "%s%s\n", cursor, label)
		}
	}
	return title + "\n\n" + sb.String() + "\nj/k to move • n to add • d to delete • Esc to go back"
}

// viewStageInputType renders the stage type picker screen.
func (a *App) viewStageInputType() string {
	title := lipgloss.NewStyle().Bold(true).Render("New Stage — Select type")
	var sb strings.Builder
	for i, label := range stageTypeLabels {
		if i == a.stageTypeIdx {
			fmt.Fprintf(&sb, "> %s\n", label)
		} else {
			fmt.Fprintf(&sb, "  %s\n", label)
		}
	}
	return title + "\n\n" + sb.String() + "\nj/k to move • Enter to select • Esc to cancel"
}

// viewStageInput renders a single-line text input for stage label or notes.
func (a *App) viewStageInput(prompt string) string {
	title := lipgloss.NewStyle().Bold(true).Render("New Stage — " + prompt)
	return title + "\n\n" + a.input.View() + "\n\nEnter to continue • Esc to go back"
}

// viewInsight renders the insight view.
func (a *App) viewInsight() string {
	title := lipgloss.NewStyle().Bold(true).Render("Insight: " + a.currentApp.CompanyName + " \u2014 " + a.currentApp.RoleTitle)
	if a.insightLoading {
		return title + "\n\nGenerating insight…\n\nEsc to go back"
	}
	content := a.currentInsight.Content
	if content == "" {
		content = "(No insight available)"
	}
	return title + "\n\n" + content + "\n\n[r] regenerate  [Esc] back"
}

// viewHuntList renders the hunt list screen.
func (a *App) viewHuntList() string {
	title := lipgloss.NewStyle().Bold(true).Render("Hunts")
	var status string
	if a.statusMsg != "" {
		status = "\n" + a.statusMsg
	}
	if a.err != nil {
		return title + "\n\nError: " + a.err.Error() + "\n\nPress q to quit."
	}
	if len(a.hunts) == 0 {
		return title + "\n\nNo hunts yet. Press n to create one.\n\nj/k to move • n to create • q to quit" + status
	}

	var sb strings.Builder
	for i, h := range a.hunts {
		n := 0
		if a.counts != nil {
			n = a.counts[h.ID]
		}
		if i == a.cursor {
			fmt.Fprintf(&sb, "  > %s (%s) \u2014 %d applications\n", h.Title, h.Status, n)
		} else {
			fmt.Fprintf(&sb, "    %s (%s) \u2014 %d applications\n", h.Title, h.Status, n)
		}
	}
	return title + "\n\n" + sb.String() + "\nj/k to move • Enter to open • n to create • c to close • q to quit" + status
}

// viewHuntInput renders the hunt name input screen.
func (a *App) viewHuntInput() string {
	title := lipgloss.NewStyle().Bold(true).Render("New Hunt")
	return title + "\n\n" + a.input.View() + "\n\nEnter to create • Esc to cancel"
}

// viewHuntDetail renders the hunt detail screen.
func (a *App) viewHuntDetail() string {
	h := a.currentHunt
	if h.ID == "" {
		return "No hunt selected.\n\nEsc to go back"
	}
	n := 0
	if a.counts != nil {
		n = a.counts[h.ID]
	}
	title := lipgloss.NewStyle().Bold(true).Render(h.Title)
	return fmt.Sprintf("%s\n\nStatus: %s\nApplications: %d\n\nEnter to open applications • Esc to go back", title, h.Status, n)
}

// statusMsg is a tea.Msg that carries an error or status string for display.
type statusMsg string

// applicationsLoadedMsg carries the result of loading applications for a hunt.
type applicationsLoadedMsg struct{ apps []domain.Application }

// applicationCreatedMsg carries the result of creating an application.
type applicationCreatedMsg struct{ app domain.Application }

// applicationUpdatedMsg carries the result of updating an application.
type applicationUpdatedMsg struct{ app domain.Application }

// loadApplicationsCmd returns a Bubble Tea command that loads applications for a hunt.
func loadApplicationsCmd(svc serviceIface, huntID string) tea.Cmd {
	return func() tea.Msg {
		apps, err := svc.ListApplications(context.Background(), huntID)
		if err != nil {
			return statusMsg(err.Error())
		}
		return applicationsLoadedMsg{apps: apps}
	}
}

// createApplicationCmd returns a Bubble Tea command that creates an application.
func createApplicationCmd(svc serviceIface, huntID, company, role, jobDesc string) tea.Cmd {
	return func() tea.Msg {
		app, err := svc.CreateApplication(context.Background(), huntID, company, role, jobDesc)
		if err != nil {
			return statusMsg(err.Error())
		}
		return applicationCreatedMsg{app: app}
	}
}

// updateApplicationCmd returns a Bubble Tea command that updates an application.
func updateApplicationCmd(svc serviceIface, app domain.Application) tea.Cmd {
	return func() tea.Msg {
		updated, err := svc.UpdateApplication(context.Background(), app)
		if err != nil {
			return statusMsg(err.Error())
		}
		return applicationUpdatedMsg{app: updated}
	}
}

// huntsLoadedMsg carries the result of loading hunts and their application counts.
type huntsLoadedMsg struct {
	hunts  []domain.Hunt
	counts map[string]int
	err    error
}

// huntCreatedMsg carries the result of creating a hunt.
type huntCreatedMsg struct {
	hunt domain.Hunt
	err  error
}

// huntClosedMsg carries the result of closing a hunt.
type huntClosedMsg struct {
	hunt domain.Hunt
	err  error
}

// loadHuntsAndCountsCmd returns a Bubble Tea command that loads all hunts
// and fetches the application count for each hunt.
func loadHuntsAndCountsCmd(svc serviceIface) tea.Cmd {
	return func() tea.Msg {
		hunts, err := svc.ListHunts(context.Background())
		if err != nil {
			return huntsLoadedMsg{err: err}
		}
		counts := make(map[string]int, len(hunts))
		for _, h := range hunts {
			apps, err := svc.ListApplications(context.Background(), h.ID)
			if err != nil {
				return huntsLoadedMsg{err: fmt.Errorf("load counts for hunt %s: %w", h.ID, err)}
			}
			counts[h.ID] = len(apps)
		}
		return huntsLoadedMsg{hunts: hunts, counts: counts}
	}
}

// createHuntCmd returns a Bubble Tea command that creates a hunt with the given name.
func createHuntCmd(svc serviceIface, name string) tea.Cmd {
	return func() tea.Msg {
		h, err := svc.CreateHunt(context.Background(), name)
		return huntCreatedMsg{hunt: h, err: err}
	}
}

// closeHuntCmd returns a Bubble Tea command that closes the hunt with the given ID.
func closeHuntCmd(svc serviceIface, id string) tea.Cmd {
	return func() tea.Msg {
		h, err := svc.CloseHunt(context.Background(), id)
		return huntClosedMsg{hunt: h, err: err}
	}
}

// stagesLoadedMsg carries the result of listing stages for an application.
type stagesLoadedMsg struct {
	stages []domain.Stage
	err    error
}

// stageCreatedMsg carries the result of adding a stage.
type stageCreatedMsg struct {
	stage domain.Stage
	err   error
}

// stageDeletedMsg carries the result of deleting a stage.
type stageDeletedMsg struct {
	id  string
	err error
}

// insightLoadedMsg carries the result of getting an insight.
// If insight.ID is empty and err is nil, no insight exists yet.
type insightLoadedMsg struct {
	insight domain.Insight
	err     error
}

// insightGeneratedMsg carries the result of generating an insight.
type insightGeneratedMsg struct {
	insight domain.Insight
	err     error
}

// loadStagesCmd returns a Bubble Tea command that lists stages for an application.
func loadStagesCmd(svc serviceIface, applicationID string) tea.Cmd {
	return func() tea.Msg {
		stages, err := svc.ListStages(context.Background(), applicationID)
		return stagesLoadedMsg{stages: stages, err: err}
	}
}

// addStageCmd returns a Bubble Tea command that adds a stage.
func addStageCmd(svc serviceIface, st domain.Stage) tea.Cmd {
	return func() tea.Msg {
		created, err := svc.AddStage(context.Background(), st)
		return stageCreatedMsg{stage: created, err: err}
	}
}

// deleteStageCmd returns a Bubble Tea command that deletes a stage by ID.
func deleteStageCmd(svc serviceIface, id string) tea.Cmd {
	return func() tea.Msg {
		err := svc.DeleteStage(context.Background(), id)
		return stageDeletedMsg{id: id, err: err}
	}
}

// getInsightCmd returns a Bubble Tea command that fetches the insight for an application.
// If no insight exists, the returned msg will have an empty insight.ID.
func getInsightCmd(svc serviceIface, applicationID string) tea.Cmd {
	return func() tea.Msg {
		insight, err := svc.GetInsight(context.Background(), applicationID)
		if err != nil {
			// ErrNotFound is a normal condition — return empty insight with no error.
			if isNotFound(err) {
				return insightLoadedMsg{insight: domain.Insight{}}
			}
			return insightLoadedMsg{err: err}
		}
		return insightLoadedMsg{insight: insight}
	}
}

// generateInsightCmd returns a Bubble Tea command that generates an insight via LLM.
func generateInsightCmd(svc serviceIface, applicationID string) tea.Cmd {
	return func() tea.Msg {
		insight, err := svc.GenerateInsight(context.Background(), applicationID)
		return insightGeneratedMsg{insight: insight, err: err}
	}
}

// isNotFound reports whether err wraps domain.ErrNotFound.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found")
}
