# Supported Open-WebUI versions

Policy (D5): pin **one current minor + one prior**, test against both, degrade gracefully.

| Version | Status | Notes |
|---|---|---|
| **0.11.0** | ✅ Pinned (verified live 2026-08-21) | `GET /api/version` on `localhost:3000`; API surface verified from served `/openapi.json` |
| **0.10.2** | ✅ Selected as prior minor (latest 0.10.x, released 2026-07-01) | Add a pinned 0.10.2 instance to CI for smoke-testing the compatibility matrix |

Client behavior on unsupported versions: clear "unsupported Open-WebUI version" notice + best-effort completions-only mode (D5), never silent failure. *(Client landed in Phase 3 with the `HTTPError`/`ChatUnavailable` degradation signals; version detection + the user notice are wired in the Phase 4 TUI.)*

## owui-term client (Phase 2)

- Build-time version: `-ldflags "-X main.version=v0.1.0"`; report via `owui-term --version` (default `dev`).
- Go floor: `go 1.23.0` (charmbracelet `golang.org/x/*` transitive deps).
- Pinned UI deps (compatible with Go ≥1.23): bubbletea v1.2.4, lipgloss v1.1.0, bubbles v0.20.0. glamour v0.9.1 is added when markdown rendering lands (v1 M2) — Phase 4 accepts plain text per D4.
