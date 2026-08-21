# API setup (authentication for owui-term)

Status: **READY** — dedicated user + API key working (2026-08-21).

## Verified state (instance `localhost:3000` v0.11.0)

| Item | State |
|---|---|
| Dedicated non-admin user | ✅ `owui-term-test` / `owui-term.test@localhost` (created via admin `POST /api/v1/auths/add`) |
| Dev credentials | `~/.config/owui-term-dev/credentials.env` (0600, outside repo) |
| **API keys** | ✅ **enabled** — `ENABLE_API_KEYS` flipped to `true` via the admin config API (`POST /api/v1/auths/admin/config`), persisted, no restart needed |
| API key created | ✅ `sk-…` (redacted; stored in credentials file) — verified authenticating (`GET /api/models` → 200) |
| Model access | ✅ test user lists all **10 models** via both JWT and API key |
| Chat CRUD via key | ✅ `POST /api/v1/chats/new` / `DELETE` → 200 |
| JWT signin | ✅ works (dev fallback) |
| Signup | ❌ disabled (`ENABLE_SIGNUP=false`) — intentional; users provisioned via admin |

## How API keys were enabled

`ENABLE_API_KEYS` is configurable at runtime via the admin API (no systemd edit / restart required):
`POST /api/v1/auths/admin/config` with the full config object + `"ENABLE_API_KEYS": true` (the endpoint requires the full body — a partial `{ENABLE_API_KEYS:true}` returns 422). Persisted to the config store; applies immediately.

## Key facts for the client (D6)

- One key per account; creating a new key invalidates the old one.
- API keys act as their creating user (permissions re-checked per request).
- Export for local dev:
  ```sh
  export OWUI_URL=http://localhost:3000
  export OWUI_TOKEN=sk-<key>
  ```

## Dev credential storage (this project)

Test credentials live in `~/.config/owui-term-dev/credentials.env` (0600), **outside** the repo — never commit tokens. The client must follow D6: env → OS keyring → non-echoing prompt; XDG config stores credential *references*, not tokens.
