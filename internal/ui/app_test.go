package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"owui-term/internal/openwebui"
	"owui-term/internal/openwebui/sse"
)

// mockClient is a scriptable chatClient for tests.
type mockClient struct {
	models   []openwebui.Model
	chats    []openwebui.Chat
	chat     *openwebui.Chat
	sse      string
	err      error
	chatsErr error // fails ListChats only (D5 degradation path)
	version  string
}

func (m *mockClient) Version(ctx context.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.version, nil
}
func (m *mockClient) Models(ctx context.Context) ([]openwebui.Model, error) {
	return m.models, m.err
}
func (m *mockClient) ListChats(ctx context.Context) ([]openwebui.Chat, error) {
	if m.chatsErr != nil {
		return nil, m.chatsErr
	}
	return m.chats, m.err
}
func (m *mockClient) CreateChat(ctx context.Context, title string, modelIDs []string) (*openwebui.Chat, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &openwebui.Chat{ID: "new-1", Title: title}, nil
}
func (m *mockClient) GetChat(ctx context.Context, id string) (*openwebui.Chat, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.chat, nil
}
func (m *mockClient) UpdateChat(ctx context.Context, id string, meta openwebui.ChatMeta) (*openwebui.Chat, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &openwebui.Chat{ID: id, Title: meta.Title}, nil
}
func (m *mockClient) DeleteChat(ctx context.Context, id string) error {
	return m.err
}
func (m *mockClient) StreamCompletions(ctx context.Context, req openwebui.CompletionsRequest) (*sse.Reader, error) {
	if m.err != nil {
		return nil, m.err
	}
	return sse.NewReader(strings.NewReader(m.sse)), nil
}

func testModel(client chatClient) Model {
	return NewWithClient(client, "test")
}

// step applies msg and returns the updated Model (helper to keep tests terse).
func step(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := m.Update(msg)
	um, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	return um
}

func TestNewStartsLoading(t *testing.T) {
	if m := testModel(&mockClient{}); m.screen != screenLoading {
		t.Errorf("screen = %v, want screenLoading", m.screen)
	}
}

func TestLoadPopulatesModels(t *testing.T) {
	m := testModel(&mockClient{models: []openwebui.Model{{ID: "qwen2.5:1.5b"}, {ID: "gemma3:12b"}}})
	m = step(t, m, loadDoneMsg{models: []openwebui.Model{{ID: "qwen2.5:1.5b"}, {ID: "gemma3:12b"}}})
	if m.screen != screenModelSelect {
		t.Fatalf("screen = %v, want screenModelSelect", m.screen)
	}
	if len(m.models) != 2 {
		t.Fatalf("models = %+v", m.models)
	}
	if !strings.Contains(m.View(), "qwen2.5:1.5b") {
		t.Errorf("model select view should list models")
	}
}

func TestLoadError(t *testing.T) {
	m := step(t, testModel(&mockClient{}), loadDoneMsg{err: errTest("no instance")})
	if m.screen != screenError {
		t.Fatalf("screen = %v, want screenError", m.screen)
	}
	if !strings.Contains(m.View(), "no instance") {
		t.Errorf("error view should render the error")
	}
}

func TestModelSelectAndCreateChat(t *testing.T) {
	m := testModel(&mockClient{models: []openwebui.Model{{ID: "m1"}, {ID: "m2"}}})
	m = step(t, m, loadDoneMsg{models: []openwebui.Model{{ID: "m1"}, {ID: "m2"}}})
	// Move down to m2 and select it.
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.modelID != "m2" {
		t.Fatalf("modelID = %q, want m2", m.modelID)
	}
	if m.screen != screenChatList {
		t.Fatalf("screen = %v, want screenChatList", m.screen)
	}
	// Enter on the "new chat" entry (cursor 0) creates a chat.
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	// Drain the async createChatCmd by sending its result.
	m = step(t, m, chatDoneMsg{chat: &openwebui.Chat{ID: "new-1", Title: "owui-term"}})
	if m.screen != screenChat || m.chat == nil || m.chat.ID != "new-1" {
		t.Fatalf("screen=%v chat=%+v", m.screen, m.chat)
	}
}

func TestSubmitPromptStreams(t *testing.T) {
	sseBody := `data: {"id":"c1","model":"m1","choices":[{"index":0,"delta":{"content":"Hel"}}]}` + "\n\n" +
		`data: {"id":"c1","model":"m1","choices":[{"index":0,"delta":{"content":"lo"}}]}` + "\n\n" +
		`data: [DONE]` + "\n\n"
	m := testModel(&mockClient{sse: sseBody})
	m.chat = &openwebui.Chat{ID: "c1"}
	m.modelID = "m1"
	m.screen = screenChat
	m.input.Focus()

	// Type a prompt and hit Enter.
	m = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'i'}})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // returns submitPromptCmd

	// Drive the stream: streamReadyMsg then the events.
	m = step(t, m, streamReadyMsg{reader: sse.NewReader(strings.NewReader(sseBody))})
	if !m.streaming {
		t.Fatal("expected streaming to start")
	}
	// Pump events through Update until done.
	m = drainStream(t, m)

	if m.streaming {
		t.Fatal("stream should have ended")
	}
	if len(m.history) != 2 {
		t.Fatalf("history = %+v, want user+assistant", m.history)
	}
	if m.history[0].Role != "user" || m.history[0].Content != "hi" {
		t.Errorf("user msg = %+v", m.history[0])
	}
	if m.history[1].Role != "assistant" || m.history[1].Content != "Hello" {
		t.Errorf("assistant msg = %+v", m.history[1])
	}
}

// drainStream feeds streamEventMsg messages from the model's channel until it
// reports done (mirroring how the bubbletea loop would consume them).
func drainStream(t *testing.T, m Model) Model {
	for i := 0; i < 10; i++ {
		ch := m.streamCh
		if ch == nil {
			break
		}
		ev, ok := <-ch
		if !ok {
			break
		}
		m = step(t, m, streamEventMsg{event: ev.event, done: ev.done})
		if ev.done || !m.streaming {
			break
		}
	}
	return m
}

func TestPersistAckReportsSaveResult(t *testing.T) {
	m := testModel(&mockClient{})
	m.chat = &openwebui.Chat{ID: "c1"}
	m.screen = screenChat

	m = step(t, m, persistDoneMsg{err: nil})
	if !strings.Contains(m.saveMsg, "saved") {
		t.Errorf("saveMsg = %q, want success indicator", m.saveMsg)
	}
	if !strings.Contains(m.View(), "saved to Open-WebUI") {
		t.Errorf("chat view missing save confirmation")
	}

	m = step(t, m, persistDoneMsg{err: errors.New("boom")})
	if !strings.Contains(m.saveMsg, "not saved") {
		t.Errorf("saveMsg = %q, want failure indicator", m.saveMsg)
	}
	if !strings.Contains(m.View(), "boom") {
		t.Errorf("chat view missing save error detail")
	}
}

func TestLoadChatsFailDegradesToCompletionsOnly(t *testing.T) {
	m := testModel(&mockClient{})
	degradeErr := errors.New("chats down")
	m = step(t, m, loadDoneMsg{models: []openwebui.Model{{ID: "m1"}}, chatsErr: degradeErr})

	if m.screen != screenModelSelect {
		t.Fatalf("screen = %v, want model select", m.screen)
	}
	if !strings.Contains(m.notice, "completions-only") {
		t.Errorf("notice = %q, want completions-only", m.notice)
	}
	if !strings.Contains(m.View(), "chats unavailable") {
		t.Errorf("model-select view missing degradation banner")
	}

	// Enter must skip the chat list and land directly in the chat screen
	// with no server chat (D5 completions-only mode).
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenChat {
		t.Fatalf("screen = %v, want chat", m.screen)
	}
	if m.chat != nil {
		t.Fatalf("chat = %+v, want nil in completions-only mode", m.chat)
	}
	if !strings.Contains(m.View(), "(none)") {
		t.Errorf("chat view should show chat id (none):\n%s", m.View())
	}
	if !strings.Contains(m.View(), "completions-only") {
		t.Errorf("chat view missing notice banner")
	}

	// Submitting a prompt must not panic with a nil chat.
	m.input.SetValue("hi")
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.history) != 1 || m.history[0].Role != "user" || m.history[0].Content != "hi" {
		t.Errorf("history = %+v, want the submitted user prompt", m.history)
	}
	if m.streaming != true {
		t.Errorf("streaming = %v, want true after submit", m.streaming)
	}
}

func TestUnsupportedVersionShowsNoticeAndSkipsChatList(t *testing.T) {
	m := testModel(&mockClient{})
	m = step(t, m, loadDoneMsg{models: []openwebui.Model{{ID: "m1"}}, chats: []openwebui.Chat{}, version: "0.12.0"})

	if !strings.Contains(m.notice, "unsupported Open-WebUI version 0.12.0") {
		t.Errorf("notice = %q, want unsupported-version text", m.notice)
	}
	if !strings.Contains(m.View(), "supported: 0.10–0.11") {
		t.Errorf("view missing supported-window hint")
	}

	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenChat {
		t.Fatalf("screen = %v, want chat (unsupported version -> completions-only)", m.screen)
	}

	// A supported version must NOT degrade.
	m2 := testModel(&mockClient{})
	m2 = step(t, m2, loadDoneMsg{models: []openwebui.Model{{ID: "m1"}}, chats: []openwebui.Chat{}, version: "0.11.0"})
	if m2.notice != "" {
		t.Errorf("notice = %q, want empty for supported version", m2.notice)
	}
	m2 = step(t, m2, tea.KeyMsg{Type: tea.KeyEnter})
	if m2.screen != screenChatList {
		t.Fatalf("screen = %v, want chat list for supported version", m2.screen)
	}
}

func TestQuitKey(t *testing.T) {
	m := testModel(&mockClient{})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected a quit command for 'q', got nil")
	}
}

func TestViewNeverLeaksToken(t *testing.T) {
	// The model deliberately does not store cfg or the bearer token (D6), so
	// no screen may render a token-looking string.
	m := testModel(&mockClient{models: []openwebui.Model{{ID: "m1"}}})
	m = step(t, m, loadDoneMsg{models: []openwebui.Model{{ID: "m1"}}})
	for _, s := range []string{m.View()} {
		if strings.Contains(s, "sk-") {
			t.Errorf("view leaked a token-like string:\n%s", s)
		}
	}
}

func TestOpenChatRendersHistory(t *testing.T) {
	chat := &openwebui.Chat{ID: "c9", Title: "old"}
	chat.Chat = []byte(`{"messages":[{"role":"user","content":"first"},{"role":"assistant","content":"reply"}]}`)
	m := testModel(&mockClient{chat: chat})
	m.modelID = "m1"
	m.chat = nil
	m.screen = screenChatList
	m.cursor = 1
	m.chats = []openwebui.Chat{*chat}

	// Simulate opening chat c9.
	m = step(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // openChatCmd for chats[0]
	m = step(t, m, chatLoadedMsg{chat: chat})
	if m.screen != screenChat {
		t.Fatalf("screen = %v, want screenChat", m.screen)
	}
	if len(m.history) != 2 || m.history[0].Content != "first" || m.history[1].Content != "reply" {
		t.Fatalf("history = %+v", m.history)
	}
	if !strings.Contains(m.View(), "first") || !strings.Contains(m.View(), "reply") {
		t.Errorf("chat view should render reloaded messages:\n%s", m.View())
	}
}

// errTest is a trivial error for tests.
type errTest string

func (e errTest) Error() string { return string(e) }
