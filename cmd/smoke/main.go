// Command smoke probes the documented Open-WebUI API surface against a live
// instance (D5). CI runs it against pinned containers (0.11.0 + 0.10.2, see
// .github/workflows/ci.yml); it also works against any instance the operator
// can reach:
//
//	OWUI_URL=http://localhost:3000 OWUI_TOKEN=<jwt-or-api-key> go run ./cmd/smoke
//
// It exercises the full documented chat CRUD + write-back round-trip but NOT
// completions (CI containers have no inference backend; the SSE parser is
// covered by fixture tests). Prints [PASS]/[FAIL] per step, exits non-zero on
// failure.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"owui-term/internal/openwebui"
)

func main() {
	client := openwebui.New(os.Getenv("OWUI_URL"), os.Getenv("OWUI_TOKEN"))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	failures := 0
	ok := func(step string, err error) {
		if err != nil {
			fmt.Printf("[FAIL] %s: %v\n", step, err)
			failures++
			return
		}
		fmt.Printf("[PASS] %s\n", step)
	}

	// 1. Version must be in the pinned compatibility window (D5).
	version, err := client.Version(ctx)
	ok("GET /api/version", err)
	if err != nil {
		exit(failures)
	}
	if supported, notice := openwebui.Supported(version); !supported {
		fmt.Printf("[FAIL] version %q outside pinned window: %s\n", version, notice)
		os.Exit(1)
	}
	fmt.Printf("       server version %q (pinned window 0.10–0.11)\n", version)

	// 2. Create a chat (no models needed for CRUD).
	title := fmt.Sprintf("smoke-%d", time.Now().Unix())
	chat, err := client.CreateChat(ctx, title, []string{"ci-smoke"})
	ok("POST /api/v1/chats/new", err)
	if err != nil {
		exit(failures)
	}
	fmt.Printf("       chat_id %q\n", chat.ID)

	// 3. Write-back the exchange (D1 persistence path).
	msgs := []openwebui.Message{{Role: "user", Content: "ping"}, {Role: "assistant", Content: "pong"}}
	if _, err := client.UpdateChat(ctx, chat.ID, openwebui.ChatMeta{Title: title, Models: []string{"ci-smoke"}, Messages: msgs}); err != nil {
		fmt.Printf("[FAIL] POST /api/v1/chats/{id} (write-back): %v\n", err)
		failures++
	} else {
		fmt.Println("[PASS] POST /api/v1/chats/{id} (write-back)")
	}

	// 4. Reload and verify both roles persisted.
	reloaded, err := client.GetChat(ctx, chat.ID)
	ok("GET /api/v1/chats/{id}", err)
	if err == nil && hasRoles(reloaded, "user", "assistant") {
		fmt.Println("[PASS] write-back survived reload (user+assistant)")
	} else if err == nil {
		fmt.Printf("[FAIL] persisted history missing roles (raw: %s)\n", truncate(string(reloaded.Chat), 200))
		failures++
	}

	// 5. The created chat must appear in the list.
	chats, err := client.ListChats(ctx)
	ok("GET /api/v1/chats/list", err)
	if err == nil {
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
	}

	// 6. Cleanup.
	ok("DELETE /api/v1/chats/{id}", client.DeleteChat(ctx, chat.ID))

	exit(failures)
}

func exit(failures int) {
	fmt.Printf("\n%d failure(s)\n", failures)
	if failures > 0 {
		os.Exit(1)
	}
	fmt.Println("SMOKE PASSED")
	os.Exit(0)
}

// hasRoles reports whether both roles are present in a chat's persisted
// history (raw "chat" payload, decoded tolerantly).
func hasRoles(c *openwebui.Chat, roles ...string) bool {
	if c == nil || len(c.Chat) == 0 {
		return false
	}
	want := map[string]bool{}
	for _, r := range roles {
		want[r] = true
	}
	var h struct {
		Messages []struct {
			Role string `json:"role"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(c.Chat, &h); err != nil {
		return false
	}
	seen := map[string]bool{}
	for _, m := range h.Messages {
		if m.Role != "" {
			seen[m.Role] = true
		}
	}
	for r := range want {
		if !seen[r] {
			return false
		}
	}
	return true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
