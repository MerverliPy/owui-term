package openwebui

import "encoding/json"

// Model is an Open-WebUI model returned by GET /api/models.
type Model struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Object string `json:"object,omitempty"`
}

// ModelsResponse is the envelope for GET /api/models.
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Message is one chat message in a completions request.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionsRequest mirrors the documented OpenAI-compatible
// POST /api/chat/completions payload.
type CompletionsRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	// ChatID binds the completion to an existing server chat (D1 session sync).
	ChatID string `json:"chat_id,omitempty"`
}

// Chat is a server-side chat/session — the source of truth (D1).
type Chat struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Title     string          `json:"title"`
	CreatedAt json.RawMessage `json:"created_at"` // numeric on live 0.11.0 (D5 tolerant)
	UpdatedAt json.RawMessage `json:"updated_at"` // numeric on live 0.11.0 (D5 tolerant)
	Archived  bool            `json:"archived"`
	Pinned    bool            `json:"pinned"`
	// Chat preserves the full message-history payload verbatim for later phases.
	Chat json.RawMessage `json:"chat"`
}

// NewChatRequest is the documented POST /api/v1/chats/new payload: the chat
// metadata lives under a nested "chat" object (verified in docs/api-notes.md).
type NewChatRequest struct {
	Chat ChatMeta `json:"chat"`
}

// ChatMeta is the chat metadata carried in the ChatForm payload (POST
// /api/v1/chats/new and POST /api/v1/chats/{id}). Messages is omitted on
// creation and populated when persisting a conversation (D1/D4).
type ChatMeta struct {
	Title    string    `json:"title"`
	Models   []string  `json:"models"`
	Messages []Message `json:"messages,omitempty"`
}
