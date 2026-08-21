package ui

import (
	"encoding/json"

	"owui-term/internal/openwebui"
)

// chatHistory is the subset of a chat's nested "chat" payload that maps to
// renderable messages (part of the documented chat CRUD surface, D5).
type chatHistory struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

// parseHistory extracts renderable user/assistant messages from a chat's
// persisted history. It returns nil for empty or malformed payloads rather than
// failing the whole reload.
func parseHistory(chat *openwebui.Chat) []openwebui.Message {
	if chat == nil || len(chat.Chat) == 0 {
		return nil
	}
	var h chatHistory
	if err := json.Unmarshal(chat.Chat, &h); err != nil {
		return nil
	}
	var msgs []openwebui.Message
	for _, mm := range h.Messages {
		if mm.Role == "" {
			continue
		}
		msgs = append(msgs, openwebui.Message{Role: mm.Role, Content: mm.Content})
	}
	return msgs
}
