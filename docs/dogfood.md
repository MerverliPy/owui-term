# Owner dogfooding log (D8)

## Status

- **Started:** 2026-08-21
- **Owner cadence:** 1–2 real usage sessions/day over 1–2 weeks; D8 collapse trigger evaluated at the end of that window (~2026-09-04).
- **Interim verdict: PENDING** (`docs/validation.md`) — no operator or public evidence yet.
- **Current stage:** trial pack ready; recruiting ~5 Open-WebUI operators.

## Trial pack for operators (task-based trial, D8b)

**Prerequisites (coordinator provisions, or operator brings):**
- Open-WebUI **v0.11.0** — the only version live-tested (see `docs/supported-versions.md`; 0.10.2 chat CRUD is CI-verified, completions unverified).
- The operator's **own server + account**. An admin must enable API keys (`ENABLE_API_KEYS`), and the operator uses a **dedicated non-admin API key** — never an admin key (see `docs/api-setup.md`).
- Built binary: `go build -o owui-term ./cmd/owui-term` (Go ≥ 1.23) or `go install ./cmd/owui-term`.
- `OWUI_URL` = the server's browser origin (e.g. `http://localhost:3000`); `OWUI_TOKEN` = the operator's own API key from that same account.

> ⚠️ **Persistence caveat:** the exchange is written back to the server *after* streaming finishes. Do not `Ctrl+C` mid-stream — wait for the full reply to render, watch for the **"✓ saved to Open-WebUI"** status line, then exit. If "⚠ not saved" appears, record the error and do not count the run as complete.

Each participant runs:

1. Launch `owui-term`, select a model.
2. Create a new chat and send one prompt containing a **unique marker** (e.g. `trial-<name>-<date>`) — every chat is titled `owui-term`, so the marker is how you identify yours later.
3. Wait for the full reply to render and the **save confirmation** to appear, then exit.
4. In a web browser, open the newest chat matching your marker and verify the user/assistant pair is present.
5. At least 24 hours later, relaunch `owui-term` unprompted, open that chat, and send a follow-up.

The coordinator records per operator: setup outcome, steps 1–5 results, facilitator prompts needed, duration, blocking errors, and whether the operator returned unprompted.

**Go/no-go threshold (D8):** PASS only when **≥3 of 5 operators complete all steps without task-specific help** AND **≥2 of 5 independently start another real session within 7 days, without reminders**. Record completion time, assistance count, and return timestamps. Public traction (30-day window) is evaluated separately.

## Owner dogfood log

| Date | Real task | TUI vs web UI | Outcome | Notes |
|---|---|---|---|---|
| 2026-08-21 | Weekend acceptance rerun (Phase 4) | TUI | ✅ | baseline; 1–2 wk window opens |

## Operator log

| Operator (anon) | OWUI version | Setup outcome | Steps 1–5 | Prompts | Duration | Blocking error | Unprompted return | Classification |
|---|---|---|---|---|---|---|---|---|
| — | — | — | — | — | — | — | — | — |

## Daily notes

- 2026-08-21: no operators contacted yet; recruit ~5 over the next 2 weeks.