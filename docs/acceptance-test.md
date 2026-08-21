# Acceptance test — D4 weekend slice

**Status: PASSED** against the pinned instance `http://localhost:3000` **v0.11.0** (2026-08-21), using the dedicated test user's API key.

Run with:

```sh
set -a; . ~/.config/owui-term-dev/credentials.env; set +a
export OWUI_TOKEN="$OWUI_TEST_API_KEY"
go run ./cmd/acceptance
```

## Result (captured run)

```
[PASS] GET /api/models                                  -> 10 model(s), using "qwen2.5:1.5b-pi"
[PASS] POST /api/v1/chats/new                          -> chat_id "7e5087a1-…"
[PASS] POST /api/chat/completions (stream)             -> 3 chunks, [DONE]=true
       assistant reply: "Hello!"
[PASS] POST /api/v1/chats/{id} (persist)
[PASS] GET /api/v1/chats/list (array)                  -> 3 chat(s)
[PASS] created chat found in list
[PASS] GET /api/v1/chats/{id}
[PASS] user+assistant messages persisted on reload
0 failure(s)  ACCEPTANCE PASSED
```

## D4 mapping

| D4 step | How | Result |
|---|---|---|
| env config → select model | `GET /api/models` (Bearer) | ✅ 10 models |
| create one server-persisted chat | `POST /api/v1/chats/new` `{chat:{title,models}}` | ✅ chat_id returned |
| stream one prompt into that chat_id | `POST /api/chat/completions` `{model,messages,chat_id,stream:true}` | ✅ 3 chunks, `[DONE]` |
| refresh chat list | `GET /api/v1/chats/list` | ✅ created chat present |
| open/reload the created chat | `GET /api/v1/chats/{id}` | ✅ returns full chat |
| verify user+assistant messages persist | parse `chat.messages` on reload | ✅ both present |

## Critical persistence finding (affects D1/D4 design)

On **v0.11.0**, the OpenAI-compatible `POST /api/chat/completions` with `chat_id` does **NOT** save the exchange into the chat — a subsequent `GET /api/v1/chats/{id}` shows only `chat:{title,models}` (no messages).

To make the conversation appear in the web UI (D1 server-state-is-source-of-truth), the client must **write the conversation back** via the documented `POST /api/v1/chats/{id}` with a `ChatForm` payload:

```json
{ "chat": { "title": "…", "models": ["…"], "messages": [ {"role":"user","content":"…"}, {"role":"assistant","content":"…"} ] } }
```

Verified: this round-trips — after the write-back, `GET /api/v1/chats/{id}` returns the messages, and the web UI reads from the same store.

## Other observed shapes (verified, 0.11.0)

- `GET /api/v1/chats/list` returns a **bare JSON array** of chat summaries (`id,title,updated_at,created_at,last_read_at,snippet,active`) — not `{items:[…]}`.
- `created_at` / `updated_at` are **numeric unix timestamps** on 0.11.0 (client decodes them tolerantly as raw JSON).
- `POST /api/v1/chats/{id}` uses the same `ChatForm` schema as `/new`.
