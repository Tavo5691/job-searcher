// Package tui provides the terminal user interface built with Bubble Tea.
// The TUI calls only the app/ layer — never adapters directly.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Tavo5691/job-searcher/internal/domain"
)

// view is the discriminant for which screen is currently active.
type view int

const (
	huntList   view = iota // 0 — list of hunts
	huntInput              // 1 — text input to create a new hunt
	huntDetail             // 2 — detail view of a single hunt
)

// serviceIface is the minimal subset of app.Service that the TUI requires.
// Defined here (at the consumer) per idiomatic Go.
type serviceIface interface {
	ListHunts(ctx context.Context) ([]domain.Hunt, error)
	CreateHunt(ctx context.Context, title string) (domain.Hunt, error)
	GetHunt(ctx context.Context, id string) (domain.Hunt, error)
	CloseHunt(ctx context.Context, id string) (domain.Hunt, error)
	ListApplications(ctx context.Context, huntID string) ([]domain.Application, error)
}

// App is the root Bubble Tea model for the job-searcher TUI.
type App struct {
	svc         serviceIface
	hunts       []domain.Hunt
	cursor      int
	err         error
	currentView view
	input       textinput.Model
	counts      map[string]int
	statusMsg   string
}

// NewApp creates a new App with the given service.
func NewApp(svc serviceIface) *App {
	ti := textinput.New()
	ti.Placeholder = "Hunt name…"
	ti.CharLimit = 100
	return &App{
		svc:   svc,
		input: ti,
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

// updateHuntDetail handles key messages in the hunt detail view.
func (a *App) updateHuntDetail(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
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
	default:
		return a.viewHuntList()
	}
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

	var rows string
	for i, h := range a.hunts {
		n := 0
		if a.counts != nil {
			n = a.counts[h.ID]
		}
		if i == a.cursor {
			rows += fmt.Sprintf("  > %s (%s) \u2014 %d applications\n", h.Title, h.Status, n)
		} else {
			rows += fmt.Sprintf("    %s (%s) \u2014 %d applications\n", h.Title, h.Status, n)
		}
	}
	return title + "\n\n" + rows + "\nj/k to move • Enter to open • n to create • c to close • q to quit" + status
}

// viewHuntInput renders the hunt name input screen.
func (a *App) viewHuntInput() string {
	title := lipgloss.NewStyle().Bold(true).Render("New Hunt")
	return title + "\n\n" + a.input.View() + "\n\nEnter to create • Esc to cancel"
}

// viewHuntDetail renders the hunt detail screen.
func (a *App) viewHuntDetail() string {
	if len(a.hunts) == 0 || a.cursor >= len(a.hunts) {
		return "No hunt selected.\n\nEsc to go back"
	}
	h := a.hunts[a.cursor]
	n := 0
	if a.counts != nil {
		n = a.counts[h.ID]
	}
	title := lipgloss.NewStyle().Bold(true).Render(h.Title)
	return fmt.Sprintf("%s\n\nStatus: %s\nApplications: %d\n\nEsc to go back", title, h.Status, n)
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
