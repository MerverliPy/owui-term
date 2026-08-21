# Supported Open-WebUI versions

Policy (D5): pin **one current minor + one prior**, test against both, degrade gracefully.

| Version | Status | Notes |
|---|---|---|
| **0.11.0** | ✅ Pinned (verified live 2026-08-21) | `GET /api/version` on `localhost:3000`; API surface verified from served `/openapi.json`; chat CRUD + write-back via CI smoke; `GET /api/models` + `POST /api/chat/completions` verified live via `cmd/acceptance` |
| **0.10.2** | ✅ Prior minor — chat CRUD **verified 2026-08-21** via CI smoke | `.github/workflows/ci.yml` → `cmd/smoke` runs the documented chat CRUD + write-back round-trip against a pinned 0.10.2 container on every push to main. **Completions + models unverified on this pin**: the CI container has no inference backend (0 models), so `cmd/smoke` prints `[SKIP]` for the streaming probe (D5 never-silent); SSE parsing is covered by fixture tests only |

Client behavior on unsupported versions: clear "unsupported Open-WebUI version" notice + best-effort completions-only mode (D5), never silent failure. *(Wired 2026-08-21: `GET /api/version` probe in the startup load, version window `0.10–0.11` in `internal/openwebui/versions.go`, and a TUI banner + chat-list skip when the server is unsupported or the chats API fails. A failed version probe is non-fatal — it is not treated as an unsupported version.)*

## owui-term client (Phase 2)

- Build-time version: `-ldflags "-X main.version=v0.1.0"`; report via `owui-term --version` (default `dev`).
- Go floor: `go 1.23.0` (charmbracelet `golang.org/x/*` transitive deps).
- Pinned UI deps (compatible with Go ≥1.23): bubbletea v1.2.4, lipgloss v1.1.0, bubbles v0.20.0. glamour v0.9.1 is added when markdown rendering lands (v1 M2) — Phase 4 accepts plain text per D4.
