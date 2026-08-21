// Package ui contains the bubbletea TUI for owui-term (stack: Go + bubbletea,
// locked constraint D2). It implements the D4 weekend acceptance slice:
// select model -> create/open a server-persisted chat -> stream a prompt into
// that chat_id (fragmented-SSE-safe) -> reload an existing chat.
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"owui-term/internal/config"
	"owui-term/internal/openwebui"
	"owui-term/internal/openwebui/sse"
)

// screen is the root model's top-level screen.
type screen int

const (
	screenLoading screen = iota
	screenModelSelect
	screenChatList
	screenChat
	screenError
)

// chatClient is the subset of the Open-WebUI client the UI needs. It is an
// interface so tests can inject a mock (real impl: *openwebui.Client).
type chatClient interface {
	Models(ctx context.Context) ([]openwebui.Model, error)
	ListChats(ctx context.Context) ([]openwebui.Chat, error)
	CreateChat(ctx context.Context, title string, modelIDs []string) (*openwebui.Chat, error)
	GetChat(ctx context.Context, id string) (*openwebui.Chat, error)
	UpdateChat(ctx context.Context, id string, meta openwebui.ChatMeta) (*openwebui.Chat, error)
	StreamCompletions(ctx context.Context, req openwebui.CompletionsRequest) (*sse.Reader, error)
}

// Model is the root bubbletea model for owui-term.
type Model struct {
	cfg     config.Config
	version string
	client  chatClient

	screen screen
	err    error

	spinner spinner.Model
	input   textinput.Model

	models []openwebui.Model
	chats  []openwebui.Chat
	cursor int

	modelID   string
	chat      *openwebui.Chat
	history   []openwebui.Message
	streamBuf string
	streaming bool
	streamCh  chan streamEventMsg

	width, height int
}

// New returns the initial model wired to a real Open-WebUI client.
func New(cfg config.Config, version string) Model {
	return newModel(openwebui.New(cfg.URL, cfg.Token), version)
}

// NewWithClient returns a model wired to the given client (for tests).
func NewWithClient(client chatClient, version string) Model {
	return newModel(client, version)
}

func newModel(client chatClient, version string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	ti := textinput.New()
	ti.Placeholder = "Message"
	ti.Prompt = "> "
	ti.CharLimit = 0
	return Model{
		version: version,
		client:  client,
		screen:  screenLoading,
		spinner: s,
		input:   ti,
	}
}

// Init kicks off the startup load (models + existing chats).
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadCmd())
}

// ---------- messages ----------

type loadDoneMsg struct {
	models []openwebui.Model
	chats  []openwebui.Chat
	err    error
}

type chatDoneMsg struct {
	chat *openwebui.Chat
	err  error
}

type chatLoadedMsg struct {
	chat *openwebui.Chat
	err  error
}

type streamReadyMsg struct {
	reader *sse.Reader
	err    error
}

type streamEventMsg struct {
	event sse.Event
	done  bool
}

// persistDoneMsg is a best-effort ack that the conversation was written back;
// it is intentionally a no-op in Update (persistence failure must not disrupt
// the already-rendered chat).
type persistDoneMsg struct{}

// ---------- commands ----------

func (m Model) loadCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		models, merr := m.client.Models(ctx)
		if merr != nil {
			return loadDoneMsg{err: merr}
		}
		chats, cerr := m.client.ListChats(ctx)
		if cerr != nil {
			// D5: chats unavailable -> degrade; keep going with models only.
			return loadDoneMsg{models: models, err: nil, chats: nil}
		}
		return loadDoneMsg{models: models, chats: chats}
	}
}

func (m Model) createChatCmd() tea.Cmd {
	modelID := m.modelID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		chat, err := m.client.CreateChat(ctx, "owui-term", []string{modelID})
		return chatDoneMsg{chat: chat, err: err}
	}
}

func (m Model) openChatCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		chat, err := m.client.GetChat(ctx, id)
		return chatLoadedMsg{chat: chat, err: err}
	}
}

// submitPrompt returns a command that streams a completions response for the
// given prompt, appending the user message to history. It is called only after
// the caller has validated prompt/chat, so it owns no state mutation.
// persistChatCmd writes the current conversation back to the server (D1).
func (m Model) persistChatCmd() tea.Cmd {
	if m.chat == nil {
		return nil
	}
	id, modelID, history := m.chat.ID, m.modelID, append([]openwebui.Message(nil), m.history...)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		meta := openwebui.ChatMeta{Title: m.chatTitle(), Models: []string{modelID}, Messages: history}
		m.client.UpdateChat(ctx, id, meta) // best-effort: errors leave the display intact
		return persistDoneMsg{}
	}
}

// chatTitle returns the current chat's title (best-effort).
func (m Model) chatTitle() string {
	if m.chat == nil {
		return "owui-term"
	}
	if m.chat.Title != "" {
		return m.chat.Title
	}
	return "owui-term"
}

func (m Model) submitPrompt(prompt string) tea.Cmd {
	modelID, chatID, history := m.modelID, m.chat.ID, m.history
	return func() tea.Msg {
		req := openwebui.CompletionsRequest{Model: modelID, Messages: history, ChatID: chatID}
		r, err := m.client.StreamCompletions(context.Background(), req)
		return streamReadyMsg{reader: r, err: err}
	}
}

// ---------- Update ----------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case loadDoneMsg:
		if msg.err != nil {
			m.screen = screenError
			m.err = msg.err
			return m, nil
		}
		m.models, m.chats = msg.models, msg.chats
		m.screen = screenModelSelect
		return m, nil
	case chatDoneMsg:
		if msg.err != nil {
			return m.setError(msg.err)
		}
		m.chat = msg.chat
		m.history = nil
		m.streamBuf = ""
		m.input.SetValue("")
		m.input.Focus()
		m.screen = screenChat
		return m, nil
	case chatLoadedMsg:
		if msg.err != nil {
			return m.setError(msg.err)
		}
		m.chat = msg.chat
		m.history = parseHistory(msg.chat)
		m.streamBuf = ""
		m.input.SetValue("")
		m.input.Focus()
		m.screen = screenChat
		return m, nil
	case streamReadyMsg:
		if msg.err != nil {
			m.streaming = false
			return m.setError(msg.err)
		}
		m.streamCh = make(chan streamEventMsg)
		go pumpStream(msg.reader, m.streamCh)
		return m, waitStreamMsg(m.streamCh)
	case streamEventMsg:
		if msg.done || msg.event.IsDone() {
			m.streaming = false
			if m.streamBuf != "" {
				m.history = append(m.history, openwebui.Message{Role: "assistant", Content: m.streamBuf})
				m.streamBuf = ""
			}
			// Persist the exchange so it appears in the web UI (D1/D4).
			return m, m.persistChatCmd()
		}
		var chunk sse.CompletionChunk
		if err := msg.event.DecodeJSON(&chunk); err != nil {
			// Tolerant: skip malformed chunks, keep the stream going (D5).
			return m, waitStreamMsg(m.streamCh)
		}
		for _, c := range chunk.Choices {
			m.streamBuf += c.Delta.Content
		}
		return m, waitStreamMsg(m.streamCh)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	quit := msg.String() == "ctrl+c" || msg.String() == "q"
	switch m.screen {
	case screenLoading, screenError:
		if quit {
			return m, tea.Quit
		}
		return m, nil

	case screenModelSelect:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if len(m.models) > 0 && m.cursor < len(m.models)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.models) == 0 {
				return m, nil
			}
			m.modelID = m.models[m.cursor].ID
			m.cursor = 0
			m.screen = screenChatList
		case "ctrl+c", "q":
			return m, tea.Quit
		}
		return m, nil

	case screenChatList:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.chats) { // +1 for the leading "new chat" entry
				m.cursor++
			}
		case "enter":
			if m.cursor == 0 {
				return m, m.createChatCmd()
			}
			return m, m.openChatCmd(m.chats[m.cursor-1].ID)
		case "ctrl+c", "q":
			return m, tea.Quit
		}
		return m, nil

	case screenChat:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.streaming {
				return m, nil
			}
			prompt := strings.TrimSpace(m.input.Value())
			if prompt == "" || m.chat == nil {
				return m, nil
			}
			// Mutate the real model here (the command closure only reads state).
			m.input.SetValue("")
			m.history = append(m.history, openwebui.Message{Role: "user", Content: prompt})
			m.streaming = true
			return m, m.submitPrompt(prompt)
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) setError(err error) (tea.Model, tea.Cmd) {
	if err == nil {
		return m, nil
	}
	m.screen = screenError
	m.err = err
	return m, nil
}

// ---------- SSE pump ----------

func pumpStream(r *sse.Reader, ch chan streamEventMsg) {
	defer close(ch)
	for {
		ev, err := r.Next()
		if err != nil {
			ch <- streamEventMsg{done: true}
			return
		}
		ch <- streamEventMsg{event: ev}
	}
}

func waitStreamMsg(ch chan streamEventMsg) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return streamEventMsg{done: true}
		}
		return ev
	}
}

// ---------- views ----------

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	cursorSel  = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
)

// View renders the current screen.
func (m Model) View() string {
	switch m.screen {
	case screenLoading:
		return m.loadingView()
	case screenModelSelect:
		return m.modelSelectView()
	case screenChatList:
		return m.chatListView()
	case screenChat:
		return m.chatView()
	case screenError:
		return m.errorView()
	}
	return ""
}

func (m Model) loadingView() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("%s %s", m.spinner.View(), mutedStyle.Render("loading…")),
	)
}

func (m Model) modelSelectView() string {
	lines := []string{titleStyle.Render("owui-term " + m.version), "", mutedStyle.Render("Select a model:"), ""}
	for i, mod := range m.models {
		name := mod.Name
		if name == "" {
			name = mod.ID
		}
		marker := "  "
		if i == m.cursor {
			marker = cursorSel.Render("> ")
		}
		lines = append(lines, marker+name)
	}
	lines = append(lines, "", mutedStyle.Render("↑/↓ to move, Enter to select, q to quit."))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) chatListView() string {
	lines := []string{
		titleStyle.Render("owui-term " + m.version),
		"",
		mutedStyle.Render("Chats (model: " + m.modelID + "):"),
		"",
	}
	for i := -1; i < len(m.chats); i++ {
		marker := "  "
		label := mutedStyle.Render("+ New chat")
		if i == 0 {
			label = cursorSel.Render("+ New chat")
		}
		if i >= 0 {
			title := m.chats[i].Title
			if title == "" {
				title = "(untitled)"
			}
			if m.cursor == i+1 {
				marker = cursorSel.Render("> ")
				label = title
			} else {
				label = mutedStyle.Render(title)
			}
		} else if m.cursor == 0 {
			marker = cursorSel.Render("> ")
		}
		lines = append(lines, marker+label)
	}
	if len(m.chats) == 0 {
		lines = append(lines, mutedStyle.Render("  (no existing chats)"), "")
	}
	lines = append(lines, mutedStyle.Render("Enter to open or create, q to quit."))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) chatView() string {
	header := fmt.Sprintf("%s  model: %s  chat: %s",
		titleStyle.Render("owui-term "+m.version), mutedStyle.Render(m.modelID), mutedStyle.Render(chatShortID(m.chat)))
	lines := []string{header, ""}
	for _, msg := range m.history {
		role := cursorSel.Render("you")
		if msg.Role == "assistant" {
			role = cursorSel.Render("ai")
		}
		lines = append(lines, role+": "+msg.Content)
	}
	if m.streaming {
		lines = append(lines, cursorSel.Render("ai")+": "+m.streamBuf+fmt.Sprintf(" %s", m.spinner.View()))
	}
	lines = append(lines, "", m.input.View(), mutedStyle.Render("Enter to send, Ctrl+C to quit."))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) errorView() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("owui-term "+m.version),
		"",
		errorStyle.Render("Error"),
		m.err.Error(),
		"",
		mutedStyle.Render("Press q to quit."),
	)
}

func chatShortID(c *openwebui.Chat) string {
	if c == nil || c.ID == "" {
		return "(none)"
	}
	id := c.ID
	if len(id) > 12 {
		return id[:12] + "…"
	}
	return id
}
