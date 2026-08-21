# Project Phases — owui-term

<!-- META
created: 2026-08-21
repository: MerverliPy/dotfiles (current scratch worktree; target repo to be created as <owner>/owui-term)
branch: main
generated_from:
  - .brainstorm/DECISION-MEMO.md
locked_constraints: concept-native, stack-go-bubbletea, name-owui-term, weekend-acceptance, api-documented-only, token-safety, roadmap-cutlines, demand-gate
active_milestone: Weekend prototype (create + reload one server-persisted chat)
milestone_state: PLANNED
next_action: Phase 1 — verify a live Open-WebUI instance against a pinned version and document the API surface
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

## Phase 2: Project scaffold (Go + bubbletea) <!-- PENDING -->
*Goal: a compiling, configurable skeleton with the build hygiene the later phases rely on.*

- [ ] `go mod init` (module `owui-term`); create `cmd/owui-term/main.go`; `go build ./...` passes.
- [ ] Add bubbletea, lipgloss, bubbles, glamour (versions pinned in `go.mod`); a minimal Model/Update/View skeleton compiles and renders a placeholder screen.
- [ ] Config layer per D6: read `OWUI_URL` / `OWUI_TOKEN` env vars in `internal/config`; unit tests for missing/invalid config produce clear, actionable errors.
- [ ] Basic loading/error UI states (D4 requires basic error states in the slice) and a `.gitignore` covering build artifacts and local secrets.
- [ ] `--version` flag and a supported-versions note wired into `docs/supported-versions.md` and README.

*Done = `go build ./...` and `go vet ./...` clean; config tests green.*

## Phase 3: API client layer + SSE parser <!-- PENDING -->
*Goal: the highest-risk component — a defensive, documented-surface-only client with fixture-tested SSE parsing (D5).*

- [ ] `internal/openwebui` client: base URL, `Authorization: Bearer`, timeouts, JSON helpers — documented endpoints only (D5).
- [ ] Typed endpoints matching `docs/api-notes.md`: `GET /api/models`, `POST /api/v1/chats/new`, `GET /api/v1/chats/list`, `GET /api/v1/chats/{id}`, streamed `POST /api/chat/completions`.
- [ ] Defensive SSE parser in `internal/openwebui/sse`: line-buffered, tolerant of fragmented chunks, handles `data:` JSON events incl. `[DONE]`, usage, error events, tool deltas (ignore-unknown), never blocks on incomplete lines.
- [ ] Fixture-driven parser tests: complete lines, fragmented lines, usage chunks, malformed events, error events, `[DONE]` — table tests against captured fixtures in `internal/openwebui/testdata/`.
- [ ] Graceful degradation: if chats endpoints fail, fall back to completions-only mode (models + stream, no session sync) with a clear user notice (D5).

*Done = `go test ./internal/openwebui/...` green (incl. SSE fixtures); `go vet` clean.*

## Phase 4: Weekend acceptance slice (TUI) <!-- PENDING -->
*Goal: the exact acceptance test from D4 — prove the Open-WebUI-native thesis end-to-end.*

- [ ] Model list/select view backed by `GET /api/models`, with loading/error states.
- [ ] Create chat via `POST /api/chats/new`; retain the returned `chat_id` for the session.
- [ ] Chat view: submit one prompt to streamed completions bound to that `chat_id`; render streamed text incrementally (plain text is acceptable this phase — markdown polish is excluded per D4).
- [ ] Refresh list (`GET /api/chats`) and open/reload a chat (`GET /api/chats/{id}`) after a fresh process start.
- [ ] Run the acceptance test end-to-end and record the result in `docs/acceptance-test.md`: env config → model select → create → stream → refresh → reload → verify user+assistant messages persist and are visible in the web UI.

*Done = acceptance test passes against the pinned instance; result documented with captured evidence.*

## Phase 5: Validation & demand gate <!-- PENDING -->
*Goal: quality bar + kick off D8 validation; record the go/no-go verdict that gates all v1 work.*

- [ ] Full checks: `go test ./...`, `go vet ./...`, `gofmt -l` clean; add `make test` / `make lint` / `make build` targets.
- [ ] Repo hygiene: complete README (quickstart via `go install`), LICENSE (MIT), final `.gitignore`.
- [ ] Start the 1–2 week owner dogfooding log (`docs/dogfood.md`); prepare the task-based trial pack (~5 Open-WebUI operators: create in TUI → reload a browser chat → resume later) with the go/no-go threshold from D8.
- [ ] Optional public step: rename repo to `owui-term`, tag `v0.1.0`, watch ~30-day traction; record signal in `docs/validation.md`.
- [ ] Write the `docs/validation.md` verdict: **STICKY** (proceed to v1 M1 per D7) or **NOT STICKY** (collapse to thin completions client or stop, per D8).

*Done = all checks green; validation started; verdict recorded.*

---

## Not yet scheduled (gated on the D8 validation verdict)

- **v1 M1:** API-client hardening + compatibility matrix (D5), named profiles + OS-keyring tokens (D6), chat list/new/open/switch/rename/delete UI.
- **v1 M2:** daily chat UX — reliable incremental markdown, model switcher, cancel/retry/regenerate, non-blocking loading/error states.
- **v1 M3:** fuzzy history/session search, external editor, slash prompt library, configurable keybindings, contextual footer, tmux/SSH-safe + inline rendering, release automation, XDG config/state/cache conventions.
- **Beyond v1:** RAG/files & knowledge workflows, multi-model responses, image/audio, structured tool/MCP read-only activity view (late v1, only after a verified event schema; no approval UX without a safety model), folders/starred/pinned, export, pipe/one-shot mode, Homebrew/AUR packaging (only after a stable v1 binary).
