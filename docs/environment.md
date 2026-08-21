# Environment (verified 2026-08-21)

Phase 1 reconnaissance record.

| Item | Value | Verified |
|---|---|---|
| Go toolchain | `go1.22.2 linux/amd64` (`go version`) | ✅ |
| Git branch | `main` | ✅ |
| Git status | clean | ✅ |
| Repo | standalone `github.com/MerverliPy/owui-term` (created 2026-08-21); this dir is no longer a dotfiles scratch subdir | ✅ |
| Open-WebUI instance | `http://localhost:3000` (PID `open-webui`), **v0.11.0** via `/api/version` | ✅ |
| Ollama | `http://localhost:11434` (up) | ✅ |
| Swagger/OpenAPI | served at `/docs` + `/openapi.json` on this instance | ✅ |
| `OWUI_URL` / `OWUI_TOKEN` env vars | not set yet | ✅ (to be set) |

Notes:
- Auth is required for `/api/models` (401 without token) — all API probes need a Bearer token.
- The instance serves Swagger at `/docs` (research noted this is normally `ENV=dev`-gated; treat as instance-specific, don't rely on it in production clients).
- This directory is now a standalone repo (`github.com/MerverliPy/owui-term`), independent of the `MerverliPy/dotfiles` home work tree.
