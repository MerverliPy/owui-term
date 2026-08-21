// Package ui contains the bubbletea TUI for owui-term (stack: Go + bubbletea,
// locked constraint D2).
package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"owui-term/internal/config"
)

// state is the root model's top-level state machine.
type state int

const (
	// stateLoading is shown during startup (Phase 3+ will drive real work here).
	stateLoading state = iota
	// stateReady is the resting placeholder screen for the Phase 2 scaffold.
	stateReady
	// stateError renders a fatal startup error with guidance.
	stateError
)

// Model is the root bubbletea model for owui-term.
type Model struct {
	cfg     config.Config
	version string

	state state
	err   error

	spinner spinner.Model

	width, height int
}

// New returns the initial model, in the loading state.
func New(cfg config.Config, version string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	return Model{cfg: cfg, version: version, state: stateLoading, spinner: s}
}

// Init starts the startup sequence. The readyMsg placeholder will be replaced
// by a real "list models" command in Phase 3.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg { return readyMsg{} })
}

type readyMsg struct{}

// Update dispatches messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case readyMsg:
		m.state = stateReady
		return m, nil

	default:
		return m, nil
	}
}

// SetError moves the model to the error state so the view can render guidance.
func (m Model) SetError(err error) Model {
	m.state = stateError
	m.err = err
	return m
}

// ---------- views ----------

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
)

// View renders the current state.
func (m Model) View() string {
	switch m.state {
	case stateError:
		return m.errorView()
	case stateLoading:
		return m.loadingView()
	default:
		return m.readyView()
	}
}

func (m Model) loadingView() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("%s %s", m.spinner.View(), mutedStyle.Render("loading…")),
	)
}

func (m Model) errorView() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("owui-term"),
		"",
		errorStyle.Render("Error"),
		m.err.Error(),
		"",
		mutedStyle.Render("Press q to quit."),
	)
}

func (m Model) readyView() string {
	// Token safety (D6): never render cfg.Token, only that one is configured.
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("owui-term "+m.version),
		"",
		mutedStyle.Render("Open-WebUI client — Phase 2 scaffold."),
		mutedStyle.Render("Server: "+m.cfg.URL+" (token configured)"),
		"",
		mutedStyle.Render("Nothing to do yet. Press q to quit."),
	)
}
