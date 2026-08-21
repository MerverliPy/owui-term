// Command smoke probes the documented Open-WebUI API surface against a live
// instance (D5). CI runs it against pinned containers (0.11.0 + 0.10.2, see
// .github/workflows/ci.yml); it also works against any instance the operator
// can reach:
//
//	OWUI_URL=http://localhost:3000 OWUI_TOKEN=<jwt-or-api-key> go run ./cmd/smoke
//
// It exercises the full documented chat CRUD + write-back round-trip plus a
// bounded GET /api/models + POST /api/chat/completions (stream) probe that runs
// only when a model is present and SKIPs explicitly otherwise (CI containers
// have no inference backend; the SSE parser is covered by fixture tests).
// Prints [PASS]/[FAIL]/[SKIP] per step, exits non-zero on failure.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"owui-term/internal/openwebui"
	"owui-term/internal/openwebui/sse"
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

	// 6. Models + bounded completions probe (D5). GET /api/models is a documented
	// surface; POST /api/chat/completions is the primary inference surface. CI
	// containers have no inference backend (0 models), so the streaming probe
	// runs only when a model is present and SKIPs explicitly otherwise — never
	// silent (D5). When a model is present the probe is falsifiable: zero data
	// chunks or a missing [DONE] sentinel is a hard failure, not a pass.
	models, err := client.Models(ctx)
	ok("GET /api/models", err)
	if err == nil && len(models) > 0 {
		switch res := runProbe(ctx, models[0].ID, client.StreamCompletions); {
		case res.failReason != "":
			fmt.Printf("[FAIL] %s\n", res.failReason)
			failures++
		default:
			fmt.Printf("[PASS] POST /api/chat/completions (stream) -> %d SSE chunks, [DONE]=true\n", res.chunks)
		}
	} else if err == nil {
		fmt.Println("[SKIP] POST /api/chat/completions (stream): no inference backend (0 models), surface unverified on this instance")
	}

	// 7. Cleanup.
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

// maxProbeChunks bounds the streaming probe so a chatty model can't stall CI.
const maxProbeChunks = 64

// probeResult is the outcome of the completions streaming probe.
type probeResult struct {
	chunks     int
	done       bool
	failReason string // non-empty ⇒ the probe is a hard failure, not a pass
}

// runProbe opens a bounded streaming completion against modelID and verifies
// the server delivered at least one data chunk and the [DONE] sentinel (D5
// never-silent; keeps [PASS] falsifiable). A stream that opens but produces no
// chunks or never closes cleanly is a hard failure, not a pass. The stream
// callback is injectable so the 0-chunk / missing-[DONE] / open-error branches
// are unit-testable without a server.
func runProbe(ctx context.Context, modelID string, stream func(context.Context, openwebui.CompletionsRequest) (*sse.Reader, error)) probeResult {
	req := openwebui.CompletionsRequest{
		Model:    modelID,
		Messages: []openwebui.Message{{Role: "user", Content: "Reply with exactly three words."}},
	}
	rd, err := stream(ctx, req)
	if err != nil {
		return probeResult{failReason: fmt.Sprintf("POST /api/chat/completions (stream): %v", err)}
	}
	chunks, done := 0, false
	for chunks < maxProbeChunks {
		ev, err := rd.Next()
		if err != nil {
			break
		}
		if ev.IsDone() {
			done = true
			break
		}
		chunks++
	}
	if chunks == 0 || !done {
		return probeResult{chunks: chunks, done: done,
			failReason: fmt.Sprintf("POST /api/chat/completions (stream): got %d chunks, [DONE]=%t (unexpected stream)", chunks, done)}
	}
	return probeResult{chunks: chunks, done: true}
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
