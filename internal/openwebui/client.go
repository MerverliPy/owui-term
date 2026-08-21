// Package openwebui is a documented-endpoints-only client for Open-WebUI
// (locked constraint D5). It talks to the pinned API surface verified in
// docs/api-notes.md and never to internal /api/chat event schemas.
package openwebui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"owui-term/internal/openwebui/sse"
)

// Client is an authenticated HTTP client for the documented Open-WebUI API.
type Client struct {
	baseURL string
	token   string

	http       *http.Client // non-streaming requests (bounded timeout)
	streamHTTP *http.Client // streaming requests (no global timeout)
}

// New returns a client for the given base URL and bearer token.
func New(baseURL, token string) *Client {
	base := strings.TrimRight(baseURL, "/")
	tr := &http.Transport{Proxy: http.ProxyFromEnvironment}
	return &Client{
		baseURL:    base,
		token:      token,
		http:       &http.Client{Transport: tr, Timeout: 30 * time.Second},
		streamHTTP: &http.Client{Transport: tr}, // no global timeout for long SSE streams
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return req, nil
}

// Models lists available models (GET /api/models).
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	var out ModelsResponse
	if err := c.getJSON(ctx, "/api/models", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CreateChat creates a chat on the server (POST /api/v1/chats/new) and returns
// the persisted chat with its server-assigned id.
func (c *Client) CreateChat(ctx context.Context, title string, modelIDs []string) (*Chat, error) {
	reqBody := NewChatRequest{Chat: ChatMeta{Title: title, Models: modelIDs}}
	var chat Chat
	if err := c.postJSON(ctx, "/api/v1/chats/new", reqBody, &chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

// ListChats returns the paged chat list (GET /api/v1/chats/list).
func (c *Client) ListChats(ctx context.Context) ([]Chat, error) {
	var out ChatListResponse
	if err := c.getJSON(ctx, "/api/v1/chats/list", &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// GetChat fetches one chat by id (GET /api/v1/chats/{id}).
func (c *Client) GetChat(ctx context.Context, id string) (*Chat, error) {
	var chat Chat
	if err := c.getJSON(ctx, "/api/v1/chats/"+url.PathEscape(id), &chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

// StreamCompletions streams an OpenAI-compatible completions response (POST
// /api/chat/completions, stream:true) and returns an SSE reader over the body.
func (c *Client) StreamCompletions(ctx context.Context, req CompletionsRequest) (*sse.Reader, error) {
	req.Stream = true
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/api/chat/completions", &buf)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.streamHTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if !isSuccess(resp.StatusCode) {
		err := statusError(http.MethodPost, "/api/chat/completions", resp)
		resp.Body.Close()
		return nil, err
	}
	return sse.NewReader(resp.Body), nil
}

// getJSON issues an authenticated GET and decodes a 2xx JSON body into out.
func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !isSuccess(resp.StatusCode) {
		return statusError(http.MethodGet, path, resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// postJSON issues an authenticated POST with a JSON body and decodes a 2xx
// JSON response into out.
func (c *Client) postJSON(ctx context.Context, path string, body, out any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !isSuccess(resp.StatusCode) {
		return statusError(http.MethodPost, path, resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func isSuccess(code int) bool {
	return code >= 200 && code < 300
}
