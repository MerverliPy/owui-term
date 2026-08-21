# Project Phases — owui-term

<!-- META
created: 2026-08-21
repository: MerverliPy/dotfiles (current scratch worktree; target repo to be created as <owner>/owui-term)
branch: main
generated_from:
  - .brainstorm/DECISION-MEMO.md
locked_constraints: concept-native, stack-go-bubbletea, name-owui-term, weekend-acceptance, api-documented-only, token-safety, roadmap-cutlines, demand-gate
active_milestone: Weekend prototype (create + reload one server-persisted chat)
milestone_state: ACCEPTED
next_action: D8 demand validation running — 1–2 wk owner dogfood, ~5-operator trial, 30-day traction; re-run docs/validation.md verdict → gates v1 M1
-->

> **WARNING:** No AGENTS.md / README.md / ADRs exist. Direction is derived solely from `.brainstorm/DECISION-MEMO.md` (council decision record, 2026-08-21). Locked constraints D1–D8 below map to memo sections; they are binding for every phase.

## Locked constraints

- **D1 `concept-native`** — Open-WebUI server state is the source of truth (two-way chat/session sync via `chat_id` persistence); NOT a generic model-API chat REPL.
- **D2 `stack-go-bubbletea`** — Go + bubbletea (+ lipgloss, bubbles, glamour). Rust/ratatui only if the primary goal changes (flip threshold in memo §4.2). TypeScript/ink rejected.
- **D3 `name-owui-term`** — product / binary / package name `owui-term`. The directory was renamed `OpenTUI` → `owui-term` (2026-08-21) to avoid the `sst/opentui` collision. (Still to do when publishing: create the standalone repo, set the git remote, check package-name availability on GitHub/brew/AUR.)
- **D4 `weekend-acceptance`** — env config → select model → **create one server-persisted chat** → stream one prompt into that `chat_id` (buffered, fragmented-SSE-safe) → refresh chat list → **open/reload the created chat** → verify user+assistant messages persist (and appear in the web UI). Excluded from the weekend slice: YAML profiles, keyring, fuzzy search, rename/delete UI, external editor, tools, files/RAG, polished markdown, theming.
- **D5 `api-documented-only`** — only documented third-party surfaces: `POST /api/chat/completions`, chat CRUD, `GET /api/models`. Never internal `/api/chat` event schemas. Pin a supported OWUI minor (+ one prior). Defensive SSE line buffering + tolerant JSON. Graceful degradation to completions-only mode if chats endpoints break. *(Verified 2026-08-21 against live v0.11.0: chat API lives under `/api/v1/chats/*` — `GET /api/v1/chats/list`, `POST /api/v1/chats/new`, `GET/DELETE /api/v1/chats/{id}` — deviating from docs.openwebui.com's `/api/chats`. See `docs/api-notes.md`.)*
- **D6 `token-safety`** — env vars for CI/overrides; OS keyring for persistent named-profile tokens; interactive non-echoing prompt fallback. XDG config stores URL/profile/credential-*reference*, never the bearer token. No automatic rotation, no browser-JWT extraction. First-run diagnostics explain admin toggle + one-key-per-account + dedicated non-admin account.
- **D7 `roadmap-cutlines`** — weekend prototype → v1 M1/M2/M3 → beyond (memo §2). Homebrew/AUR packaging follows a stable v1 binary.
- **D8 `demand-gate`** — v1 expansion is gated on validation signals: (a) 1–2 weeks owner dogfooding; (b) task-based trial with ~5 Open-WebUI operators (create in TUI → reload a browser chat → resume later); go/no-go = majority completes the workflow and several return unprompted; (c) public prototype traction within ~30 days. If create+reload is not sticky in the owner's own workflow within 2 weeks → collapse to a thin completions client or stop.

---

## Phase 1: Reconnaissance & environment verification <!-- COMPLETE -->
*Goal: verify the council's API/SSE assumptions against a live instance before writing code; pin the target version.*

> **Phase 1 outcome (2026-08-21):** complete — the two formerly-blocked items are resolved (owner/server actions, no code):
> - **API keys ✅ enabled** — `ENABLE_API_KEYS` flipped to `true` via admin config API (`POST /api/v1/auths/admin/config`, full body), persisted, no restart. Test user's key (`sk-…`) verified: auth 200, lists all 10 models, creates/deletes chats.
> - **Test-user model access ✅ resolved** — both JWT and API key list all **10 models** (`access_grants: []` = open). *(The earlier 0-models JWT view was transient, cleared once API keys were enabled.)*
> - **Prior minor ✅ selected** — **0.10.2** (latest 0.10.x, 2026-07-01) for the compatibility matrix; CI to stand up a pinned instance for smoke tests.
> - Remaining: `POST /api/v1/models/model/access/update` returns 200 but doesn't persist grants — per-model restrictions deferred to post-v1 (open default).

- [x] Verify toolchain: `go version` (≥ 1.22), `git status` clean; record versions in `docs/environment.md`.
- [x] Confirm access to a live Open-WebUI instance; record its exact version and pin it in `docs/supported-versions.md` (target: one current minor + one prior).
- [x] Create a dedicated non-admin Open-WebUI user + API key; verify `ENABLE_API_KEYS` and the `API Keys` feature permission; record steps in `docs/api-setup.md`. *(✅ user `owui-term-test` + API key `sk-…`; `ENABLE_API_KEYS` set true via admin config API, no restart)*
- [x] Probe documented endpoints with `curl` against the pinned instance: `GET /api/models` (Bearer auth), `POST /api/v1/chats/new`, `GET /api/v1/chats/list`, `GET /api/v1/chats/{id}`, and one streamed `POST /api/chat/completions` (`stream:true`); capture response shapes into `docs/api-notes.md`.
- [x] Confirm the OpenAI-compatible SSE event shape (delta / usage / `[DONE]` / tool deltas) and whether `POST /api/chat/completed` is required for outlet filters; record exact event fields in `docs/api-notes.md`. *(fixture captured: `docs/api-fixtures/stream-0.11.0.sse`)*
- [x] Create `README.md` stub (name, one-line pitch, install placeholder) as the project's first truth document.

*Done = all Phase 1 checklist items complete; artifacts exist under `docs/`.*

## Phase 2: Project scaffold (Go + bubbletea) <!-- COMPLETE -->
*Goal: a compiling, configurable skeleton with the build hygiene the later phases rely on.*

> **Phase 2 outcome (2026-08-21):** complete. `go build ./...`, `go vet ./...`, `gofmt -l .` clean; 11 config unit tests green; `--version` and all config error paths verified on the built binary.
> - Module `owui-term`; pinned Go ≥1.23 (charmbracelet `golang.org/x/*` transitives require it; 1.22.2 auto-downloads the 1.23 toolchain). Pinned deps: bubbletea v1.2.4, lipgloss v1.1.0, bubbles v0.20.0.
> - **glamour deferred** (not to Phase 4) — not imported yet; markdown polish is excluded from the weekend slice (D4 accepts plain text), so it's added only when markdown rendering lands (v1 M2). `go mod tidy` correctly drops the unused dep.
> - Structure: `cmd/owui-term/main.go` (entry + `--version`), `internal/config` (D6 env config + tests), `internal/ui` (Model/Update/View skeleton with loading/ready/error states, spinner). Token never rendered (D6).

- [x] `go mod init` (module `owui-term`); create `cmd/owui-term/main.go`; `go build ./...` passes.
- [x] Add bubbletea, lipgloss, bubbles, glamour (versions pinned in `go.mod`); a minimal Model/Update/View skeleton compiles and renders a placeholder screen. *(glamour added when markdown rendering lands, later than Phase 4)*
- [x] Config layer per D6: read `OWUI_URL` / `OWUI_TOKEN` env vars in `internal/config`; unit tests for missing/invalid config produce clear, actionable errors.
- [x] Basic loading/error UI states (D4 requires basic error states in the slice) and a `.gitignore` covering build artifacts and local secrets.
- [x] `--version` flag and a supported-versions note wired into `docs/supported-versions.md` and README.

*Done = `go build ./...` and `go vet ./...` clean; config tests green.*

## Phase 3: API client layer + SSE parser <!-- COMPLETE -->
*Goal: the highest-risk component — a defensive, documented-surface-only client with fixture-tested SSE parsing (D5).*

> **Phase 3 outcome (2026-08-21):** complete. `go test ./internal/openwebui/...` green (17 tests: 8 client + 9 SSE), `go vet`/`gofmt -l .` clean.
> - `internal/openwebui`: typed documented endpoints only (D5) — `GET /api/models`, `POST /api/v1/chats/new`, `GET /api/v1/chats/list`, `GET /api/v1/chats/{id}`, streamed `POST /api/chat/completions`.
> - Client is httptest-tested; `HTTPError` carries status/path so Phase 4 can branch; `ChatUnavailable()` is the D5 degradation signal.
> - `internal/openwebui/sse`: one event per `data:` line (the framing verified on live 0.11.0), tolerant of fragmented reads, never dispatches an incomplete line. Captured fixture + fragmented/malformed/error/`[DONE]`/comment tests. *(The blank-line-separator model was corrected against the real fixture, which uses newline-delimited `data:` lines.)*
> - Graceful degradation: typed `HTTPError` + `ChatUnavailable` helper ready; TUI wiring lands in Phase 4.

- [x] `internal/openwebui` client: base URL, `Authorization: Bearer`, timeouts, JSON helpers — documented endpoints only (D5).
- [x] Typed endpoints matching `docs/api-notes.md`: `GET /api/models`, `POST /api/v1/chats/new`, `GET /api/v1/chats/list`, `GET /api/v1/chats/{id}`, streamed `POST /api/chat/completions`.
- [x] Defensive SSE parser in `internal/openwebui/sse`: line-buffered, tolerant of fragmented chunks, handles `data:` JSON events incl. `[DONE]`, usage, error events, tool deltas (ignore-unknown), never blocks on incomplete lines.
- [x] Fixture-driven parser tests: complete lines, fragmented lines, usage chunks, malformed events, error events, `[DONE]` — table tests against captured fixtures in `internal/openwebui/testdata/`.
- [x] Graceful degradation: if chats endpoints fail, fall back to completions-only mode (models + stream, no session sync) with a clear user notice (D5). *(Signal ready via `ChatUnavailable`; TUI notice wired in Phase 4.)*

*Done = `go test ./internal/openwebui/...` green (incl. SSE fixtures); `go vet` clean.*

## Phase 4: Weekend acceptance slice (TUI) <!-- COMPLETE -->
*Goal: the exact acceptance test from D4 — prove the Open-WebUI-native thesis end-to-end.*

> **Phase 4 outcome (2026-08-21):** complete. Acceptance test **PASSED** end-to-end against live v0.11.0 (`docs/acceptance-test.md`); TUI verified rendering live (model select, 10 models).
> - TUI (bubbletea): loading → model select → chat list (new/open) → chat view with incremental SSE streaming (fragmented-safe). Injectable `chatClient` interface (mock-tested, 8 UI tests).
> - **Critical persistence finding:** `POST /api/chat/completions` with `chat_id` does NOT save the exchange on 0.11.0. Added documented write-back `POST /api/v1/chats/{id}` (`ChatForm`) after each stream — this is what makes messages appear in the web UI (D1). Also: `/api/v1/chats/list` returns a bare array; `created_at`/`updated_at` are numeric (decoded tolerantly).
> - New client method `UpdateChat`; `ChatMeta.Messages` added; `ChatListResponse` removed (array shape).

- [x] Model list/select view backed by `GET /api/models`, with loading/error states.
- [x] Create chat via `POST /api/v1/chats/new`; retain the returned `chat_id` for the session.
- [x] Chat view: submit one prompt to streamed completions bound to that `chat_id`; render streamed text incrementally (plain text is acceptable this phase — markdown polish is excluded per D4).
- [x] Refresh list (`GET /api/v1/chats/list`) and open/reload a chat (`GET /api/v1/chats/{id}`) after a fresh process start.
- [x] Run the acceptance test end-to-end and record the result in `docs/acceptance-test.md`: env config → model select → create → stream → refresh → reload → verify user+assistant messages persist and are visible in the web UI. *(Persistence via `POST /api/v1/chats/{id}` write-back — see acceptance doc.)*

*Done = acceptance test passes against the pinned instance; result documented with captured evidence.*

## Phase 5: Validation & demand gate <!-- COMPLETE --> <!-- PHASE_TIME: ~600s -->
*Goal: quality bar + kick off D8 validation; record the go/no-go verdict that gates all v1 work.*

> **Phase 5 outcome (2026-08-21):** complete — quality bar verified green this session: `go test ./...`, `go vet ./...`, `gofmt -l .` clean; `make test` / `make lint` / `make build` all exit 0. Repo hygiene done (README quickstart, MIT LICENSE, final `.gitignore`). D8 validation started: `docs/dogfood.md` (owner log + ~5-operator trial pack with go/no-go threshold) and interim verdict recorded in `docs/validation.md` (**NOT STICKY — awaiting demand proof**; re-run after the 1–2 wk window per D8). v1 M1 remains gated. Optional public step deferred: standalone repo `MerverliPy/owui-term` exists on GitHub (main @ Phase 4) — tagging `v0.1.0` + push is an owner publish decision; the 30-day traction window is time-gated. Handoff: `records/phase-5-handoff.md`.

- [x] Full checks: `go test ./...`, `go vet ./...`, `gofmt -l .` clean; add `make test` / `make lint` / `make build` targets.
- [x] Repo hygiene: complete README (quickstart via `go install`), LICENSE (MIT), final `.gitignore`.
- [x] Start the 1–2 week owner dogfooding log (`docs/dogfood.md`); prepare the task-based trial pack (~5 Open-WebUI operators: create in TUI → reload a browser chat → resume later) with the go/no-go threshold from D8.
- [-] Optional public step: ~~rename repo to `owui-term`~~ (standalone repo exists: `MerverliPy/owui-term`), tag `v0.1.0`, watch ~30-day traction; record signal in `docs/validation.md`. <!-- DEFERRED: v0.1.0 tag + push is an owner publish decision; 30-day traction window is time-gated (D8c) -->
- [x] Write the `docs/validation.md` verdict: **STICKY** (proceed to v1 M1 per D7) or **NOT STICKY** (collapse to thin completions client or stop, per D8).

*Done = all checks green; validation started; verdict recorded.*

---

## Not yet scheduled (gated on the D8 validation verdict)

- **v1 M1:** API-client hardening + compatibility matrix (D5), named profiles + OS-keyring tokens (D6), chat list/new/open/switch/rename/delete UI.
- **v1 M2:** daily chat UX — reliable incremental markdown, model switcher, cancel/retry/regenerate, non-blocking loading/error states.
- **v1 M3:** fuzzy history/session search, external editor, slash prompt library, configurable keybindings, contextual footer, tmux/SSH-safe + inline rendering, release automation, XDG config/state/cache conventions.
- **Beyond v1:** RAG/files & knowledge workflows, multi-model responses, image/audio, structured tool/MCP read-only activity view (late v1, only after a verified event schema; no approval UX without a safety model), folders/starred/pinned, export, pipe/one-shot mode, Homebrew/AUR packaging (only after a stable v1 binary).
