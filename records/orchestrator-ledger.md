# Orchestrator Ledger — Per-Unit Effective-Token Measurements

**Purpose:** P1 measurement log for `docs/orchestrator-plan.md`. One row per accepted-verified work unit.
Gate to P2 (any enforcement): ≥10 measured units, ≥20% net `E_unit` reduction vs raw-session baseline after
subtracting orchestrator meta-cost, zero edit-breaking incidents, no verified-completion drop.
**Status: 1/10 units measured — advisory only.**

```
E_unit = admitted_parent_in + admitted_parent_out + Σ(child totalTokens) − ctx_stats_savings
```
Per **accepted-AND-verified** unit only (tests/lock/D8 gates pass; council = converged + memo delivered).
Sample at: session open → pre-launch → child terminal → acceptance. `ctx_stats` after each unit.

## Measurements

| # | Date | Unit | Route class | Model tier(s) | Admitted in/out | Σ child tokens | ctx savings | Σ cost | E_unit (tok) | Accepted | Verified |
|---|------|------|-------------|----------------|-----------------|----------------|-------------|--------|--------------|----------|----------|
| 001 | 2026-08-21 | Council: orchestrator plan (2 passes, 3 advisors) | council | grok-4.5 / deepseek-v4-pro / gpt-5.6-sol | 125,853 / 29,316 | 214,541 / 25,514 (pass-1 only) | 0 | $1.24 (parent + pass-1 children) | ~395,224 | ✅ | ✅ |
| 002 | 2026-08-21 | Autonomy enactment: watchdog on (gpt-5.5:high), envelope policy recorded, settings write | config | openai-codex/gpt-5.5 (watchdog) | parent-only | 0 (no children) | 0 | ~$0.01 | ~small (parent-only) | ✅ | ✅ |

**Notes on 001:** child totals = pass-1 session summaries (`$0.2039` + `$0.0478` + `$0.9329`); pass-2 fallback
cross-exam runs + advisor read tool calls are not yet instrumented → row is approximate. ctx savings = 0 this
session (context-mode used once, 551 B entered). This row is the honest starting point: council-grade work is
expensive (~$1.24), context-mode contributed nothing yet.

**Notes on 002:** config task, no child agents; E_unit approximated from parent session delta; accepted via owner
grant, verified via `watchdog.status` (Main: on, gpt-5.5:high). Not a routed work unit — listed for ledger
completeness.

## Protocol reminders

- Prefer `ctx_execute_file`/`ctx_batch_execute`/`ctx_search` per routing table — then measure whether E_unit drops.
- Re-run `ctx_stats` after each measured task; update this table.
- Trigger table applies once `budget_pct` is configured (owner decision #1 open).