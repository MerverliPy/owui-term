# Environment (verified 2026-08-21)

Phase 1 reconnaissance record.

| Item | Value | Verified |
|---|---|---|
| Go toolchain | `go1.22.2 linux/amd64` (`go version`) | ✅ |
| Git branch | `main` (dotfiles scratch subdirectory at `/home/calvin/owui-term`) | ✅ |
| Git status | clean (all new files git-ignored by design) | ✅ |
| Open-WebUI instance | `http://localhost:3000` (PID `open-webui`), **v0.11.0** via `/api/version` | ✅ |
| Ollama | `http://localhost:11434` (up) | ✅ |
| Swagger/OpenAPI | served at `/docs` + `/openapi.json` on this instance | ✅ |
| `OWUI_URL` / `OWUI_TOKEN` env vars | not set yet | ✅ (to be set) |

Notes:
- Auth is required for `/api/models` (401 without token) — all API probes need a Bearer token.
- The instance serves Swagger at `/docs` (research noted this is normally `ENV=dev`-gated; treat as instance-specific, don't rely on it in production clients).
- This directory is a git-ignored subdirectory of the `MerverliPy/dotfiles` repo (which ignores everything with `*`); the standalone project repo still needs to be created as `owui-term`.
