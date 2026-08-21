package openwebui

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testClient spins up an httptest server and a client pointed at it.
func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL, "sk-test")
}

func TestModels(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/models" {
			t.Errorf("path = %q, want /api/models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"object":"list","data":[{"id":"qwen2.5:1.5b","name":"qwen2.5:1.5b"},{"id":"gemma3:12b"}]}`)
	})

	models, err := c.Models(context.Background())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
	if models[0].ID != "qwen2.5:1.5b" || models[1].ID != "gemma3:12b" {
		t.Errorf("models = %+v", models)
	}
}

func TestCreateChat(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chats/new" {
			t.Errorf("path = %q, want /api/v1/chats/new", r.URL.Path)
		}
		var body NewChatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Chat.Title != "Test" || len(body.Chat.Models) != 1 || body.Chat.Models[0] != "m1" {
			t.Errorf("payload = %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"chat-1","user_id":"u1","title":"Test"}`)
	})

	chat, err := c.CreateChat(context.Background(), "Test", []string{"m1"})
	if err != nil {
		t.Fatalf("CreateChat: %v", err)
	}
	if chat.ID != "chat-1" || chat.Title != "Test" {
		t.Errorf("chat = %+v", chat)
	}
}

func TestListChats(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chats/list" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"id":"a","title":"one","updated_at":123},{"id":"b","title":"two"}]`)
	})

	chats, err := c.ListChats(context.Background())
	if err != nil {
		t.Fatalf("ListChats: %v", err)
	}
	if len(chats) != 2 || chats[0].ID != "a" || chats[1].ID != "b" {
		t.Errorf("chats = %+v", chats)
	}
}

func TestGetChat(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chats/chat-42" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"chat-42","title":"hi","created_at":"2026-08-21T00:00:00Z"}`)
	})

	chat, err := c.GetChat(context.Background(), "chat-42")
	if err != nil {
		t.Fatalf("GetChat: %v", err)
	}
	if chat.ID != "chat-42" || chat.Title != "hi" {
		t.Errorf("chat = %+v", chat)
	}
}

func TestUpdateChat(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/chats/c1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var body NewChatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Chat.Title != "t" || len(body.Chat.Messages) != 2 || body.Chat.Messages[1].Content != "ok" {
			t.Errorf("payload = %+v", body.Chat)
		}
		io.WriteString(w, `{"id":"c1","title":"t"}`)
	})

	_, err := c.UpdateChat(context.Background(), "c1", ChatMeta{
		Title:    "t",
		Models:   []string{"m1"},
		Messages: []Message{{Role: "user", Content: "hi"}, {Role: "assistant", Content: "ok"}},
	})
	if err != nil {
		t.Fatalf("UpdateChat: %v", err)
	}
}

func TestStreamCompletions(t *testing.T) {
	sseBody := `data: {"id":"c1","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"hi"}}]}` + "\n\n" +
		`data: [DONE]` + "\n\n"
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var body CompletionsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !body.Stream {
			t.Error("Stream must be forced true")
		}
		if body.ChatID != "chat-42" {
			t.Errorf("ChatID = %q", body.ChatID)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseBody)
	})

	rd, err := c.StreamCompletions(context.Background(), CompletionsRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hi"}},
		ChatID:   "chat-42",
	})
	if err != nil {
		t.Fatalf("StreamCompletions: %v", err)
	}

	var contents []string
	for {
		ev, err := rd.Next()
		if err != nil {
			break
		}
		if ev.IsDone() {
			continue
		}
		var chunk sseCompletionChunk
		if err := ev.DecodeJSON(&chunk); err != nil {
			t.Fatalf("decode chunk: %v", err)
		}
		if len(chunk.Choices) > 0 {
			contents = append(contents, chunk.Choices[0].Delta.Content)
		}
	}
	if strings.Join(contents, "") != "hi" {
		t.Errorf("streamed content = %q, want %q", strings.Join(contents, ""), "hi")
	}
}

// sseCompletionChunk is a minimal local mirror to avoid importing sse types here;
// it exercises the same public DecodeJSON path.
type sseCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func TestHTTPErrorOnBadStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"detail":"internal error"}`)
	})

	_, err := c.Models(context.Background())
	if err == nil {
		t.Fatal("expected an error for 500, got nil")
	}
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if he.Status != 500 {
		t.Errorf("Status = %d, want 500", he.Status)
	}
	if !strings.Contains(he.Error(), "/api/models") {
		t.Errorf("error %q should mention the path", he)
	}
}

func TestVersionJSONEnvelope(t *testing.T) {
	// Shape captured from live 0.11.0.
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/version" {
			t.Errorf("path = %q, want /api/version", r.URL.Path)
		}
		io.WriteString(w, `{"version":"0.11.0","deployment_id":""}`)
	})
	v, err := c.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != "0.11.0" {
		t.Errorf("Version = %q, want 0.11.0", v)
	}
}

func TestVersionPlainStringFallback(t *testing.T) {
	// Tolerant decoding: some prior minors return a bare string.
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `0.10.2`)
	})
	v, err := c.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != "0.10.2" {
		t.Errorf("Version = %q, want 0.10.2", v)
	}
}

func TestVersionUnrecognizedResponse(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{}`)
	})
	if _, err := c.Version(context.Background()); err == nil {
		t.Error("Version on {} should fail (no version field)")
	}
}

func TestVersionErrorStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	})
	_, err := c.Version(context.Background())
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("Version error = %T, want *HTTPError", err)
	}
	if he.Status != http.StatusForbidden || he.Path != "/api/version" {
		t.Errorf("HTTPError = %+v", he)
	}
}

func TestDeleteChat(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/chats/chat-1" {
			t.Errorf("path = %q, want /api/v1/chats/chat-1", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteChat(context.Background(), "chat-1"); err != nil {
		t.Fatalf("DeleteChat: %v", err)
	}
}

func TestChatUnavailable(t *testing.T) {
	if !ChatUnavailable(&HTTPError{Method: "GET", Path: "/api/v1/chats/list", Status: 404}) {
		t.Error("404 on /chats should be reported as chats unavailable")
	}
	if ChatUnavailable(&HTTPError{Method: "GET", Path: "/api/v1/chats/list", Status: 500}) {
		t.Error("500 on /chats should NOT be reported as chats unavailable")
	}
	if ChatUnavailable(&HTTPError{Method: "GET", Path: "/api/models", Status: 404}) {
		t.Error("404 on /api/models should NOT be reported as chats unavailable")
	}
	if ChatUnavailable(nil) {
		t.Error("nil should not be chats unavailable")
	}
}

func TestBaseURLTrailingSlashNormalized(t *testing.T) {
	// A trailing slash on the base URL must not produce // in the request path.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/models" {
			t.Errorf("path = %q, want /api/models (no doubled slash)", r.URL.Path)
		}
		io.WriteString(w, `{"object":"list","data":[]}`)
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL+"/", "sk-test")
	if _, err := c.Models(context.Background()); err != nil {
		t.Fatalf("Models with trailing-slash base: %v", err)
	}
}
