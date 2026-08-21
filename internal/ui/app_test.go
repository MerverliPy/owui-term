package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"owui-term/internal/openwebui"
	"owui-term/internal/openwebui/sse"
)

// mockClient is a scriptable chatClient for tests.
type mockClient struct {
	models []openwebui.Model
	chats  []openwebui.Chat
	chat   *openwebui.Chat
	sse    string
	err    error
}

func (m *mockClient) Models(ctx context.Context) ([]openwebui.Model, error) {
	return m.models, m.err
}
func (m *mockClient) ListChats(ctx context.Context) ([]openwebui.Chat, error) {
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
