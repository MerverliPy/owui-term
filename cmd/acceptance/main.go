// Command acceptance runs the Phase 4 (D4) acceptance slice against a live
// Open-WebUI instance and prints a pass/fail per step. It is NOT part of the
// normal test suite (it needs a live server); run it manually:
//
//	set -a; . ~/.config/owui-term-dev/credentials.env; set +a
//	export OWUI_URL OWUI_TOKEN=...
//	go run ./cmd/acceptance
//
// The recorded result lives in docs/acceptance-test.md.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"owui-term/internal/openwebui"
	"owui-term/internal/openwebui/sse"
)

func main() {
	cfg := configFromEnv()
	client := openwebui.New(cfg.url, cfg.token)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	failures := 0
	ok := func(step string, err error, extra string) {
		if err != nil {
			fmt.Printf("[FAIL] %s: %v\n", step, err)
			failures++
			return
		}
		fmt.Printf("[PASS] %s%s\n", step, extra)
	}

	// 1. Models
	models, err := client.Models(ctx)
	ok("GET /api/models", err, "")
	if err != nil {
		os.Exit(1)
	}
	if len(models) == 0 {
		fmt.Println("[FAIL] no models returned")
		os.Exit(1)
	}
	fmt.Printf("       %d model(s), using %q\n", len(models), models[0].ID)

	// 2. Create chat
	title := fmt.Sprintf("acceptance-%d", time.Now().Unix())
	chat, err := client.CreateChat(ctx, title, []string{models[0].ID})
	ok("POST /api/v1/chats/new", err, fmt.Sprintf(" -> chat_id %q", chatID(chat)))
	if err != nil {
		os.Exit(1)
	}

	// 3. Stream one completion bound to that chat_id
	req := openwebui.CompletionsRequest{
		Model:    models[0].ID,
		Messages: []openwebui.Message{{Role: "user", Content: "Reply with exactly one word: hello."}},
		ChatID:   chat.ID,
	}
	rd, err := client.StreamCompletions(ctx, req)
	if err != nil {
		fmt.Printf("[FAIL] POST /api/chat/completions: %v\n", err)
		os.Exit(1)
	}
	var reply strings.Builder
	chunks, done := 0, false
	for {
		ev, err := rd.Next()
		if err != nil {
			break
		}
		if ev.IsDone() {
			done = true
			break
		}
		var c sse.CompletionChunk
		if err := ev.DecodeJSON(&c); err != nil {
			continue
		}
		for _, choice := range c.Choices {
			reply.WriteString(choice.Delta.Content)
		}
		chunks++
	}
	ok("POST /api/chat/completions (stream)", nil, fmt.Sprintf(" -> %d chunks, [DONE]=%v", chunks, done))
	replyText := strings.TrimSpace(reply.String())
	fmt.Printf("       assistant reply: %q\n", replyText)

	// 3b. Persist the exchange (POST /api/v1/chats/{id}) so it appears in the
	//     web UI — the OpenAI-compatible endpoint does not save it on 0.11.0.
	persisted, err := client.UpdateChat(ctx, chat.ID, openwebui.ChatMeta{
		Title:    title,
		Models:   []string{models[0].ID},
		Messages: []openwebui.Message{{Role: "user", Content: "Reply with exactly one word: hello."}, {Role: "assistant", Content: replyText}},
	})
	ok("POST /api/v1/chats/{id} (persist)", err, "")
	_ = persisted

	// 4. Refresh list and find the chat
	chats, err := client.ListChats(ctx)
	ok("GET /api/v1/chats/list (array)", err, fmt.Sprintf(" -> %d chat(s)", len(chats)))
	found := false
	for _, c := range chats {
		if c.ID == chat.ID {
			found = true
			break
		}
	}
	if found {
		fmt.Println("[PASS] created chat found in list")
	} else {
		fmt.Println("[FAIL] created chat NOT found in refreshed list")
		failures++
	}

	// 5. Reload the chat and verify user+assistant messages persisted
	reloaded, err := client.GetChat(ctx, chat.ID)
	ok("GET /api/v1/chats/{id}", err, "")
	if err == nil && reloaded != nil {
		hist := chatHistory(reloaded)
		hasUser, hasAssistant := hist["user"], hist["assistant"]
		if hasUser && hasAssistant {
			fmt.Println("[PASS] user+assistant messages persisted on reload")
		} else {
			fmt.Printf("[FAIL] persisted history missing messages: user=%v assistant=%v raw=%s\n",
				hasUser, hasAssistant, truncate(string(reloaded.Chat), 200))
			failures++
		}
	} else {
		failures++
	}

	fmt.Printf("\n%d failure(s)\n", failures)
	if failures > 0 {
		os.Exit(1)
	}
	fmt.Println("ACCEPTANCE PASSED")
}

// --- helpers ---

type envConfig struct{ url, token string }

func configFromEnv() envConfig {
	return envConfig{url: os.Getenv("OWUI_URL"), token: os.Getenv("OWUI_TOKEN")}
}

func chatID(c *openwebui.Chat) string {
	if c == nil {
		return ""
	}
	return c.ID
}

// chatHistory reports which roles appear in a chat's persisted history.
func chatHistory(c *openwebui.Chat) map[string]bool {
	out := map[string]bool{}
	if c == nil || len(c.Chat) == 0 {
		return out
	}
	var h struct {
		Messages []struct {
			Role string `json:"role"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(c.Chat, &h); err != nil {
		return out
	}
	for _, m := range h.Messages {
		if m.Role != "" {
			out[m.Role] = true
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
