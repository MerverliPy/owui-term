# Demand & quality validation (D8)

## Quality bar

- `go test ./...` ✅ (2026-08-21)
- `go vet ./...` ✅
- `gofmt -l .` clean ✅
- `make test` / `make lint` / `make build` ✅

## Validation signals

### Owner dogfooding (started)
- Phase 4 acceptance workflow is complete and repeatable at least once.
- 1–2 week owner log started in `docs/dogfood.md`.

### Trial operators
- Not yet started (needs ~5 Open-WebUI operators).

### Public traction
- **Published 2026-08-21:** `v0.1.0` tagged on [MerverliPy/owui-term](https://github.com/MerverliPy/owui-term) — repo is currently **PRIVATE**, so the D8c 30-day traction window opens only after the owner makes it public; signal recorded back here ~30 days later.

## Interim verdict

**Verdict: PENDING — no demand evidence yet (day 1).**

D8's go/no-go is evidence-based: (a) 1–2 weeks of owner dogfooding, (b) the ~5-operator task trial, (c) ~30 days of public traction. None of that evidence exists yet, so the gate is undetermined:

- **No v1 work begins while PENDING — v1 M1 included** (D8 gates all v1 expansion).
- Owner dogfooding runs 1–2 weeks (log: `docs/dogfood.md`); the D8 collapse trigger — owner create/reload not sticky in their own workflow — is evaluated at the end of that window (~2026-09-04).
- If the collapse trigger fires, record **NOT STICKY** and collapse to a thin completions client or stop.
- Otherwise, evaluate the full gate after the operator trial and 30 days of public traction, then record **STICKY** (proceed to v1 M1) or **NOT STICKY** (collapse or stop).
- This page is re-run and updated when evidence lands.
