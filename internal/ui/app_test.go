package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"owui-term/internal/config"
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

// runCmd executes a tea.Cmd (as the bubbletea loop would) and feeds its
// resulting message back into Update — covers the async command closures that
// step() alone leaves unexecuted.
func runCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("runCmd: nil command")
	}
	return step(t, m, cmd())
}

func TestNewWiresClientAndVersion(t *testing.T) {
	m := New(config.Config{URL: "http://x", Token: "sk-test"}, "v-test")
	if m.screen != screenLoading {
		t.Errorf("screen = %v, want screenLoading", m.screen)
	}
	if m.version != "v-test" {
		t.Errorf("version = %q, want v-test", m.version)
	}
}

func TestLoadingViewRenders(t *testing.T) {
	m := testModel(&mockClient{})
	if !strings.Contains(m.View(), "loading") {
		t.Errorf("loading view should render a loading indicator")
	}
}

func TestModelSelectNavigationBounds(t *testing.T) {
	models := []openwebui.Model{{ID: "m1"}, {ID: "m2"}}
	m := step(t, testModel(&mockClient{}), loadDoneMsg{models: models})

	// Up at the top stays put.
	m = step(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after up at top", m.cursor)
	}
	// Down moves once, then stops at the last entry.
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after down x2 on 2 models", m.cursor)
	}
	// Enter with an empty model list must not advance or panic.
	m2 := step(t, testModel(&mockClient{}), loadDoneMsg{models: nil})
	if _, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Error("enter with no models should be a no-op")
	}
	if m2.screen != screenModelSelect {
		t.Errorf("screen = %v, want model select", m2.screen)
	}
}

func TestChatListEmptyAndNewChatEntry(t *testing.T) {
	m := testModel(&mockClient{})
	m.modelID = "m1"
	m.screen = screenChatList

	v := m.View()
	if !strings.Contains(v, "+ New chat") || !strings.Contains(v, "(no existing chats)") {
		t.Errorf("empty chat list should offer new chat entry:\n%s", v)
	}
	// Enter on the new-chat entry issues createChatCmd.
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Error("enter on new-chat entry should return createChatCmd")
	}
	// Navigation bounds with no chats: cursor stays at 0.
	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = step(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestChatListViewRendersTitles(t *testing.T) {
	m := testModel(&mockClient{})
	m.modelID = "m1"
	m.screen = screenChatList
	m.chats = []openwebui.Chat{{ID: "c1", Title: "Alpha"}, {ID: "c2", Title: "Beta"}}

	v := m.View()
	if !strings.Contains(v, "Alpha") || !strings.Contains(v, "Beta") || !strings.Contains(v, "+ New chat") {
		t.Errorf("chat list should render titles + new-chat entry:\n%s", v)
	}
	// Enter on an existing chat row issues openChatCmd.
	m.cursor = 1
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Error("enter on an existing chat should return openChatCmd")
	}
}

func TestCreateChatCmdError(t *testing.T) {
	m := testModel(&mockClient{err: errTest("create boom")})
	m.modelID = "m1"
	m.screen = screenChatList
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // cursor 0 -> createChatCmd
	m = runCmd(t, m, cmd)
	if m.screen != screenError {
		t.Fatalf("screen = %v, want screenError", m.screen)
	}
	if !strings.Contains(m.View(), "create boom") {
		t.Errorf("error view should render the create failure")
	}
	// q quits from the error screen.
	if _, qcmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}); qcmd == nil {
		t.Error("q on error screen should quit")
	}
}

func TestOpenChatCmdError(t *testing.T) {
	m := testModel(&mockClient{err: errTest("open boom")})
	m.modelID = "m1"
	m.screen = screenChatList
	m.chats = []openwebui.Chat{{ID: "c1"}}
	m.cursor = 1
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // openChatCmd for c1
	m = runCmd(t, m, cmd)
	if m.screen != screenError {
		t.Fatalf("screen = %v, want screenError", m.screen)
	}
	if !strings.Contains(m.View(), "open boom") {
		t.Errorf("error view should render the open failure")
	}
}

func TestStreamReadyError(t *testing.T) {
	m := testModel(&mockClient{})
	m.screen = screenChat
	m.streaming = true
	m = step(t, m, streamReadyMsg{err: errTest("stream boom")})
	if m.streaming {
		t.Error("streaming should stop on stream error")
	}
	if m.screen != screenError || !strings.Contains(m.View(), "stream boom") {
		t.Errorf("screen = %v, want error view with message", m.screen)
	}
}

func TestStreamToleratesMalformedChunk(t *testing.T) {
	m := testModel(&mockClient{})
	m.screen = screenChat
	m.streaming = true
	m.streamCh = make(chan streamEventMsg)
	// Malformed JSON chunks must be skipped, not fatal (D5 tolerant).
	m = step(t, m, streamEventMsg{event: sse.Event{Data: "{not json"}})
	if !m.streaming {
		t.Error("malformed chunk should not stop the stream")
	}
}

func TestWindowSizeStored(t *testing.T) {
	m := step(t, testModel(&mockClient{}), tea.WindowSizeMsg{Width: 101, Height: 37})
	if m.width != 101 || m.height != 37 {
		t.Errorf("size = %dx%d, want 101x37", m.width, m.height)
	}
}

func TestPersistCmdRunsEndToEnd(t *testing.T) {
	// Success path: the closure calls UpdateChat and the ack renders.
	m := testModel(&mockClient{})
	m.chat = &openwebui.Chat{ID: "c1", Title: "My chat"}
	m.modelID = "m1"
	m.history = []openwebui.Message{{Role: "user", Content: "hi"}}
	m.screen = screenChat
	m = runCmd(t, m, m.persistChatCmd())
	if !strings.Contains(m.saveMsg, "saved to Open-WebUI") {
		t.Errorf("saveMsg = %q, want saved confirmation", m.saveMsg)
	}
	if !strings.Contains(m.View(), "saved to Open-WebUI") {
		t.Errorf("chat view missing save confirmation")
	}

	// Failure path: UpdateChat error is surfaced, not silent.
	m2 := testModel(&mockClient{err: errTest("persist boom")})
	m2.chat = &openwebui.Chat{ID: "c1", Title: "My chat"}
	m2.modelID = "m1"
	m2.screen = screenChat
	m2 = runCmd(t, m2, m2.persistChatCmd())
	if !strings.Contains(m2.saveMsg, "not saved") || !strings.Contains(m2.saveMsg, "persist boom") {
		t.Errorf("saveMsg = %q, want failure indication", m2.saveMsg)
	}
}

func TestWaitStreamMsg(t *testing.T) {
	// A pending event is delivered.
	ch := make(chan streamEventMsg, 1)
	ch <- streamEventMsg{event: sse.Event{Data: "hello"}}
	if msg := waitStreamMsg(ch)().(streamEventMsg); msg.event.Data != "hello" {
		t.Errorf("msg = %+v, want the pending event", msg)
	}
	// A closed channel is reported as done.
	close(ch)
	if msg := waitStreamMsg(ch)().(streamEventMsg); !msg.done {
		t.Errorf("msg = %+v, want done on closed channel", msg)
	}
}

func TestChatSubmitGuards(t *testing.T) {
	m := testModel(&mockClient{})
	m.screen = screenChat
	m.chat = &openwebui.Chat{ID: "c1"}

	// Empty prompt: no command.
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Error("empty prompt should not submit")
	}
	// Streaming in progress: no command.
	m.streaming = true
	m.input.SetValue("hi")
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Error("enter while streaming should be ignored")
	}
}

// errTest is a trivial error for tests.
type errTest string

func (e errTest) Error() string { return string(e) }
