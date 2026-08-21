package openwebui

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Edge-case and error-path coverage for the API client: non-2xx statuses
// (statusError paths), path escaping, and the truncate/Error helpers.

func TestDeleteChatErrorStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
	})
	err := c.DeleteChat(context.Background(), "missing-1")
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("DeleteChat error = %T, want *HTTPError", err)
	}
	if he.Method != http.MethodDelete || he.Path != "/api/v1/chats/missing-1" || he.Status != http.StatusNotFound {
		t.Errorf("HTTPError = %+v", he)
	}
	if !strings.Contains(he.Error(), "not found") {
		t.Errorf("error %q should include the response body", he.Error())
	}
}

func TestDeleteChatEscapedID(t *testing.T) {
	// An id with reserved characters must be path-escaped on the wire. Note:
	// Go's http.Server decodes %2F into r.URL.Path, so compare EscapedPath().
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/chats/" + url.PathEscape("a/b c")
		if got := r.URL.EscapedPath(); got != want {
			t.Errorf("escaped path = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteChat(context.Background(), "a/b c"); err != nil {
		t.Fatalf("DeleteChat escaped id: %v", err)
	}
}

func TestGetChatEscapedID(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/chats/" + url.PathEscape("a/b")
		if got := r.URL.EscapedPath(); got != want {
			t.Errorf("escaped path = %q, want %q", got, want)
		}
		io.WriteString(w, `{"id":"a/b","title":"t"}`)
	})
	chat, err := c.GetChat(context.Background(), "a/b")
	if err != nil {
		t.Fatalf("GetChat escaped id: %v", err)
	}
	if chat.ID != "a/b" {
		t.Errorf("chat.ID = %q", chat.ID)
	}
}

func TestListChatsErrorStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	})
	_, err := c.ListChats(context.Background())
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("ListChats error = %T, want *HTTPError", err)
	}
	if he.Status != http.StatusInternalServerError || he.Path != "/api/v1/chats/list" {
		t.Errorf("HTTPError = %+v", he)
	}
}

func TestGetChatErrorStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusGone)
	})
	_, err := c.GetChat(context.Background(), "x")
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("GetChat error = %T, want *HTTPError", err)
	}
	if he.Status != http.StatusGone {
		t.Errorf("HTTPError = %+v", he)
	}
}

func TestUpdateChatErrorStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"bad chat"}`, http.StatusBadRequest)
	})
	_, err := c.UpdateChat(context.Background(), "c1", ChatMeta{Title: "t"})
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("UpdateChat error = %T, want *HTTPError", err)
	}
	if he.Method != http.MethodPost || he.Status != http.StatusBadRequest || he.Path != "/api/v1/chats/c1" {
		t.Errorf("HTTPError = %+v", he)
	}
}

func TestCreateChatErrorStatus(t *testing.T) {
	// Exercises the postJSON non-2xx branch.
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"failed"}`, http.StatusUnprocessableEntity)
	})
	_, err := c.CreateChat(context.Background(), "t", []string{"m1"})
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("CreateChat error = %T, want *HTTPError", err)
	}
	if he.Status != http.StatusUnprocessableEntity || he.Path != "/api/v1/chats/new" {
		t.Errorf("HTTPError = %+v", he)
	}
}

func TestStreamCompletionsErrorStatus(t *testing.T) {
	// Non-2xx on the streaming path must surface an *HTTPError and close the body.
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat/completions" {
			t.Errorf("path = %q, want /api/chat/completions", r.URL.Path)
		}
		http.Error(w, `{"detail":"overloaded"}`, http.StatusTooManyRequests)
	})
	_, err := c.StreamCompletions(context.Background(), CompletionsRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("StreamCompletions error = %T, want *HTTPError", err)
	}
	if he.Status != http.StatusTooManyRequests || he.Path != "/api/chat/completions" {
		t.Errorf("HTTPError = %+v", he)
	}
	if !strings.Contains(he.Error(), "overloaded") {
		t.Errorf("error %q should include the response body", he.Error())
	}
}

func TestListChatsMalformedJSON(t *testing.T) {
	// 2xx with malformed JSON must surface a decode error, not a silent zero value.
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{not json`)
	})
	if _, err := c.ListChats(context.Background()); err == nil {
		t.Error("ListChats on malformed 2xx body should fail")
	}
}

func TestGetChatMalformedJSON(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{`)
	})
	if _, err := c.GetChat(context.Background(), "x"); err == nil {
		t.Error("GetChat on malformed 2xx body should fail")
	}
}

func TestUpdateChatMalformedJSON(t *testing.T) {
	// Exercises the postJSON decode-error branch via a public caller.
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `nope`)
	})
	if _, err := c.UpdateChat(context.Background(), "c1", ChatMeta{Title: "t"}); err == nil {
		t.Error("UpdateChat on malformed 2xx body should fail")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate(short, 10) = %q", got)
	}
	if got := truncate("abcdefghij", 5); got != "abcde…" {
		t.Errorf("truncate(abcdefghij, 5) = %q, want abcde…", got)
	}
	if got := truncate("", 3); got != "" {
		t.Errorf("truncate(\"\", 3) = %q", got)
	}
}

func TestHTTPErrorWithoutBody(t *testing.T) {
	e := &HTTPError{Method: http.MethodGet, Path: "/api/models", Status: http.StatusForbidden}
	if got, want := e.Error(), "GET /api/models: 403"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestHTTPErrorTrimmedBody(t *testing.T) {
	e := &HTTPError{Method: http.MethodPost, Path: "/p", Status: 500, Body: "  \n boom \n "}
	if got, want := e.Error(), "POST /p: 500: boom"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
