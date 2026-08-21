# Open-WebUI API notes (verified against live instance)

Server: `http://localhost:3000` — **v0.11.0**, verified 2026-08-21 via `GET /api/version` and the served `/openapi.json` (242 API paths of 476 total).

## Auth

| Endpoint | Notes |
|---|---|
| `POST /api/v1/auths/signin` | Email+password → `{token}` (JWT, Bearer) — **verified** |
| `POST /api/v1/auths/signup` | **Disabled on this instance** (403, `ENABLE_SIGNUP=false`) |
| `GET/POST/DELETE /api/v1/auths/api_key` | Per-user API key — **disabled** (403 "not allowed in the environment", `ENABLE_API_KEYS=false` env var) |
| `POST /api/v1/auths/add` | Admin-only user creation — **verified** (created `owui-term-test`) |
| `GET /api/v1/auths/admin/config` | Admin auth config — **verified** (`ENABLE_API_KEYS`, `ENABLE_SIGNUP`, endpoint restrictions) |

All requests: `Authorization: Bearer <token-or-key>`. JWT works for everything; API keys are unavailable until the admin flips `ENABLE_API_KEYS` and restarts.

## Chat completions (OpenAI-compatible, SSE) — **verified live**

- `POST /api/chat/completions` — primary third-party surface (also `/api/v1/chat/completions`, `/openai/chat/completions`, `/ollama/v1/chat/completions`)
- `stream: true` → `data:` SSE lines; `[DONE]` terminator; **verified** with `qwen2.5:1.5b`
- `POST /api/chat/completed` — exists in spec (needed for `outlet()` filters; not inline on stable releases)
- ⚠️ SSE chunks may split mid-line — the client must line-buffer defensively (fixture: `docs/api-fixtures/stream-0.11.0.sse`)

**Verified event shape** (captured 2026-08-21, fixture `docs/api-fixtures/stream-0.11.0.sse`):

```json
data: {"id":"chatcmpl-…","created":1787293897,"model":"qwen2.5:1.5b","object":"chat.completion.chunk",
       "choices":[{"index":0,"logprobs":null,"finish_reason":null,
                   "delta":{"role":"assistant","content":"P"}}]}
…
data: [DONE]
```

- Choice events: `id`, `created`, `model`, `object`, `choices[].delta.{role,content}` (+ `finish_reason` on the last chunk, `stop`)
- Final chunk carries `usage` (input/output tokens + Ollama timing fields + `completion_tokens_details`)
- Stream terminates with a literal `data: [DONE]`

## Models

- `GET /api/models` — list (auth required); admin sees 10 models (`qwen2.5:*`, `qwen3-vl:8b`, `gemma3:12b`, `nomic-embed-text`, `arena-model`)
- `GET /api/v1/models`, `GET /api/v1/models/list` — v1 variants
- **Model visibility:** verified after enabling API keys — the test user lists all **10 models** via both JWT and API key (`access_grants: []` on models = open; groups empty). *(An earlier probe showed 0 models for the fresh user — transient, resolved; see api-setup.md.)*
- ⚠️ `POST /api/v1/models/model/access/update` returns 200 for `{id, access_grants:[{user_ids:[…], group_ids:[…]}]}` but does **not** persist the grant (model `access_grants` stays empty). Access control is currently left at the open default; revisit only if per-model restrictions are needed (post-v1).

## Chats / sessions — **v1 namespace in 0.11.0** ⚠️

> **Deviation from research:** docs.openwebui.com documented `/api/chats`; the live 0.11.0 spec exposes the chat API under **`/api/v1/chats/*`**. Live spec is authoritative (D5).

| Endpoint | Purpose | Verified |
|---|---|---|
| `POST /api/v1/chats/new` | Create chat — payload **`{chat: {title, models: […]}}`** (not `{title}`!) → `{id, user_id, title, chat, updated_at, created_at, share_id, archived, pinned, meta, …}` | ✅ |
| `GET /api/v1/chats/list` | Paged chat list | spec |
| `GET/DELETE /api/v1/chats/{id}` | Fetch / delete chat | ✅ |
| `POST /api/v1/chats/read` | Mark read | spec |
| `GET /api/v1/chats/search` | Search | spec |
| `GET /api/v1/chats/all` | All chats | spec |

## ⚠️ Persistence (verified 0.11.0 — D1/D4 critical)

- `POST /api/chat/completions` with `chat_id` does **NOT** save the exchange into the chat on 0.11.0. A later `GET /api/v1/chats/{id}` shows only `chat:{title,models}` (no messages).
- To persist a conversation (so it appears in the web UI), the client must **write it back** via `POST /api/v1/chats/{id}` with a `ChatForm` payload `{chat:{title,models,messages:[{role,content},…]}}` (same schema as `/new`). Verified round-trip: after write-back, `GET /api/v1/chats/{id}` returns the messages.
- `GET /api/v1/chats/list` returns a **bare JSON array** of summaries (`id,title,updated_at,created_at,last_read_at,snippet,active`) — not `{items:[…]}`. (Client decodes `[]Chat` directly.)
- `created_at` / `updated_at` are **numeric unix timestamps** on 0.11.0 — decode tolerantly (the client uses `json.RawMessage`).

## Other surfaces (later phases)

`/api/v1/knowledge/*` (RAG), `/api/v1/files/*` (async processing + `GET /api/v1/files/{id}/process/status`), `/api/v1/tools/*`, `/api/v1/models/*` (declarative mgmt), `/api/v1/images/*`, `/api/v1/audio/*`.

## Fixtures

- `docs/api-fixtures/stream-0.11.0.sse` — real captured SSE stream (3 choice events + `[DONE]`, with usage) for Phase 3 parser tests.

## Open questions

- `POST /api/v1/models/model/access/update` accepts `{id, access_grants:[{user_ids:[…], group_ids:[…]}]}` (200) but does not persist the grant (`access_grants` stays empty). Per-model access control is left at the open default; revisit post-v1 if restrictions are needed.
- ~~Whether a non-admin user can create chats normally~~ — **resolved**: the non-admin API key created/deleted a chat (200).
