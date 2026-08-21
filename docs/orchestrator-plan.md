# Orchestrator Plan — Guard-Railed Multi-Agent Workflow

**Council-reviewed 2026-08-21** — `council-architect` + `council-operator` + `council-skeptic` (2 passes, converged).
See council memo in session transcript (`paseo`-adjacent council run `10a21a82…`, pass 2 `28b2081f…`) for the full
claim matrix, accepted/rejected feedback, and cross-exam refinements.

**Status: Draft (P0 artifacts: routing table + ledger schema + baseline recorded).** This drives PI-assisted authoring workflow generally; it does **not** modify
`PHASES.md` (content-locked, D8 PENDING) and it is **not** owui-term product work. Any application of this
orchestrator to owui-term stays non-v1 / process-only.

**Scope:** topology, decision loop, effective-token telemetry ledger, skill/agent routing, interactive
questioning protocol, guardrails, phased rollout. **Non-goals:** no new daemon/agent binary, no unbounded
autonomy, no benchmarking.

---

## 1. Shape (converged)

**Parent-session policy loop** built on existing pi-subagents / council-mode / context-mode primitives — not a
new process, not a peer mesh. One parent = sole decision authority + mutation dispatcher; read-only
specialists; **exactly one writer per cwd/worktree**; every launch is an async `workflowScript`
(`runs.run` / `runs.all`).

### Decision loop (per work unit U)

1. **Observe** — mission objective, lane board, `ctx_stats` baseline, fleet status (`totalTokens` / `totalCost` / `turnCount` / `toolCount`), lock gates.
2. **Classify** — recon | research | design/council | plan | implement | review | lock-compliance | phase-exec | loop/babysit | docs.
3. **Budget gate** — hard `usageBudget` exceeded → stop new launches; soft → degrade route (cheaper tier, less fanout, ctx_* only).
4. **Ambiguity gate** — missing product/scope/authority/success criteria → clarify before routing; else default + record assumption.
5. **Route** — pick skill+agent from the P0 routing table (§3); set context, cwd, authority, output path, stop rules.
6. **Execute** — bounded async wave; parent synthesizes; one writer per worktree.
7. **Sample + feedback** — fold child receipts into the ledger (§2); adjust next route class; checkpoint/stop.
8. **Stop** — objective met / pass, spawn, or budget caps hit / lock block / user interrupt / unresolved owner decision.

---

## 2. Effective-token telemetry ledger (v0)

### Definition

```
E_unit = admitted_parent_in + admitted_parent_out + Σ(child totalTokens) − ctx_stats_savings
```
- Computed **per accepted-and-verified unit** (tests / lock / D8 gates pass). Un-accepted or unverified units are
  **excluded** from the success mean (anti-Goodhart: shallow/skipped-verify answers are failures, not cheaper wins).
- Billed cost tracks the same fields in `$`.
- **Session-history growth is the dominant driver** (`docs/session-optimization.md` P2), not routing. Treat the
  ledger + router as **cost-control infrastructure, not assumed ROI**. Earned-ROI claim is deferred until
  `mean E_routed + O_route < mean E_direct` over ≥10 measured accepted-verified tasks.

### Ledger fields (identical at every sample point)

| Field | Meaning |
|---|---|
| `unit_id`, `mission_id`, `sample_point`, `ts` | identity + sampling context |
| `baseline_in_tokens` | in-tokens at mission/session open |
| `admitted_parent_in`, `admitted_parent_out` | parent context admitted to the model |
| `child_totalTokens`, `child_totalCost`, `child_turnCount`, `child_toolCount` | per child receipt |
| `Σ_children_tokens`, `Σ_children_cost` | rollup |
| `ctx_stats_sandbox_tokens`, `ctx_stats_admitted_tokens`, `ctx_savings` | sandbox − admitted (measured, not claimed) |
| `context_window_pct` | parent context % |
| `budget_tokens_cap`, `budget_cost_cap`, `budget_tokens_used`, `budget_cost_used`, `budget_pct` | `budget_pct = max(tokens/cap, cost/cap)` |
| `route_class`, `model_tier`, `fanout_degree` | routing outcome |
| `accepted` bool, `verified` bool | quality gate |

**Sample points:** mission/session open → pre-launch → each child terminal → phase/milestone boundary →
before clarify rounds → acceptance → `mission.close`. Sampling happens at these natural boundaries via the parent,
**never a token-consuming monitor agent**.

### Signal → decision bindings (exactly one decision each; reporting-only signals dropped)

| Signal | Decision it changes |
|---|---|
| S1 `context_window_pct` | budget action / session split (trigger table rows 1, 3) |
| S2 `budget_pct` | fanout degree → freeze → stop (rows 2–4) |
| S3 `E_unit` vs class baseline | route_class / model_tier degrade on next unit |
| S4 `ctx_savings` ratio | force context-mode path vs raw `read` |
| S5 spawn count vs cap | fanout-degree hard cap |

**Reporting-only (do not drive routing):** disk MB, raw `toolCount` alone, lifetime cost-event counts,
error-event totals, git-op counts.

### Unified trigger → action table (top-down; first match wins)

| # | Trigger | Action |
|---|---|---|
| 1 | `context_window_pct ≥ 35%` OR `in_tokens ≥ 2× baseline` | prefer `ctx_execute_file`/`ctx_batch_execute` + index/search; cut fanout (advisory) |
| 2 | `budget_pct ≥ 50%` | disable speculative fanout; cheapest-sufficient only |
| 3 | `context_window_pct ≥ 55%` OR `budget_pct ≥ 80%` | freeze new mutation launches; checkpoint/verification/read-only wrap-up only |
| 4 | `budget_pct ≥ 100%` OR mission `goal.budget.tokens` exhausted | stop new launches; `needs_attention`; no silent success |
| 5 | always | no `turnBudget` / hard `toolBudget` / tight per-child `usageBudget` on mutation workers |

**Quality reserve:** the final 20% of budget is reserved for verification + one remediation, never speculative
work. **Calibration:** 35%/55% and 2× are data-calibrated after ≥10 measured tasks; 50/80/100% rungs and the
worker-budget forbid are owner-set policy aligned with platform `usageBudget` semantics (soft report + hard gate
*laters* launches; cannot stop an already-running child — no reservation model).

---

## 3. P0 routing table (what to use, when)

### Inventory (agents)

| Agent | Purpose | Authority | Context | Cost tier |
|---|---|---|---|---|
| `scout` | recon, compressed context handoff | read | fresh | cheap |
| `delegate` | generic lightweight; skill carrier | read | — | cheap |
| `researcher` | web/research briefs | read | fresh | medium |
| `reviewer` | fresh diff/plan/proposal review | read | fresh | medium |
| `council-architect` · `council-operator` · `council-skeptic` | material tradeoff council (pass cap 2) | read | fresh | high |
| `lock-reviewer` | content-lock compliance | read | fresh | cheap |
| `oracle` | trajectory / drift advisory | read | fork | high |
| `worker` / `plan-worker` | implementation (sole writer) | write | fork | high |
| `polisher` | docs / user-facing polish | write | fork | medium |
| `paseo-loop` | external babysit loop (bounded: max-iters/time) | read-only | — | medium |

### Inventory (skills)

- **Analysis/design:** `code-review`, `codebase-design`, `domain-modeling`, `diagnosing-bugs`, `tdd`, `find-docs`, `prototype`
- **Process:** `phase-executor`, `phases-creator`, `paseo-*` (advisory/committee/handoff/loop), `council-mode`, `quick-audit`
- **Context/token:** `context-mode`/`ctx-*` (`ctx_execute_file`, `ctx_batch_execute`, `ctx_search`, `ctx_index`, `ctx_stats`)
- **Browser:** `agent-browser` (web/app interaction)

### Selection criteria

Score U on: mutation? lock-sensitive? ambiguity? evidence locality? parallelism safety? cost tier.

### Cheapest-sufficient order

```
parent-local ctx_* → scout → specialist skill (via delegate) → role agent → council (material tradeoffs only) → single writer (only after approval)
```

### Fallbacks

- Missing council profile → `oracle` (fork) + `reviewer`.
- Non-resumable cross-exam → fresh context, same role, own pass-1 report + challenge packet (labeled fallback).
- Writer failure → one checkpoint + one retry, then escalate; never respawn a writer on dirty unknown ownership.
- Tool/protocol failure → `status`/`doctor`; do not guess.

---

## 4. Interactive questioning protocol

- **Ask only when ambiguity changes:** acceptance, externally visible product/API/schema behavior,
  security/privacy, credentials, irreversible/destructive effects, migration/data loss, material cost,
  release/merge authority, or locked content.
- **How:** parent uses `ask_user_question` (≤4 questions/invocation, 2–4 options, recommended-first);
  children use `contact_supervisor` (`need_decision` / `interview_request`), serialized — one pending ask at a time.
- **Cost bounds:** ≤2 clarify rounds per unit before escalate/stop. Clarify tokens count toward `usageBudget`.
  Never launch writers under open owner decisions.
- **ASK vs DEFAULT (converged rule):** DEFAULT only when **all** five hold —
  1. reversible (cheaply, e.g. git-revertible local edit, non-published artifact);
  2. low spend (≤ ~$0.50 or <1% lifetime budget);
  3. within already-granted authority;
  4. within already-approved scope class (recorded assumption in the approved plan);
  5. no safety/destructive/irreversible/external consequence.
  Any failure → ASK, recording an open decision via `mission.update` decisions.
- **Escalation:** at the 2-invocation/8-question cap, any material/user-owned unknown **blocks work and
  escalates to the user** — the cap is a fatigue bound, never permission to guess. Unattended execution stops.

---

## 5. Guardrails (always-on invariants; never relaxed by data)

- Parent is the sole final authority and synthesizer; no peer chat / transcript sharing.
- Council pass cap 2 (3 only with `--max-passes 3` + evidence settleable); review rounds ≤3.
- Nesting depth ≤2; spawn/concurrency caps.
- Exactly one writer per cwd/worktree (`worktree: true` only for intentional parallel writes).
- **Never** pass `turnBudget` / hard `toolBudget` / tight per-child `usageBudget` to mutation-capable workers.
- Root/mission `usageBudget`: soft report + hard gate on later launches.
- Merge/release/authority approvals = receipts (evidence), not permission.
- Content-locked plans (`PHASES.md`) immutable; status-level changes only through the project's stated channel.
- No unbounded loops (`paseo` bounded by iterations/time); `mission.close` at termination.

---

## 6. Phased rollout + engineering gates

| Phase | Build | Gate to exit |
|---|---|---|
| **P0 (now)** | inventory + routing table (§3) + `ctx_stats` baseline + ledger schema (§2) | baseline recorded; inventory accurate |
| **P1** | ledger wiring + per-unit measurement over ≥10 tasks | **measure-first gate:** ≥20% net `E_unit` reduction vs raw-session baseline *after* subtracting orchestrator meta-cost, AND zero edit-breaking incidents in advisory window, AND no verified-completion drop |
| **P2** | enforcement, gated on P1 data: `read`>2KB hard-block (*fails open* for small/edit-bound reads), budgets on read-only children only, spawn caps | P1 gate met + no misfires |
| **P3** | bounded autonomy for reversible low-risk work: `mission goal:true` with `budget.tokens`, watchdog on — **GRANTED 2026-08-21** (owner decision #2 resolved; watchdog on, envelope per §7) | explicit per-item owner grant — done; expansion requires P1 gate |
| **P4** | failure injection: budget exhaust, dirty worktree, non-resumable advisor, lock-verify fail | all injected failures land in owner decisions, not silent recovery |

**Build order:** routing table → ledger → clarify gate **last** (highest per-invocation meta-cost; defer until
routing+ledger delta is proven).

---

## 7. Owner decisions (open — not settled by evidence)

1. Budget values: per-mission/per-unit token & cost ceilings; `usageBudget` hard-stop default-on?
2. **Autonomy envelope — RESOLVED 2026-08-21 (owner grant).** Enacted per P3:
   - **In envelope (bounded autonomy):** the reversible low-risk class only — the 5-condition DEFAULT rule (§4) is
     the boundary; `mission goal:true` with `budget.tokens` for idle continuation; **watchdog ON**
     (`openai-codex/gpt-5.5:high`, user-scope settings, auto-follow for blockers).
   - **Stays gated (not granted):** P2 enforcement (read-guard, budgets) until the 10-unit measurement gate;
     scheduled runs (never enabled); unbounded loops; merge/release/credentials/irreversible actions;
     content-lock changes.
   - Note: grant was made with P1 measurement at 1/10 units — autonomy is owner-authorized but data-unproven;
     revert is one settings write away (`subagents.watchdog.enabled: false`).
3. `read`>2KB hard-block: experiment-for-N-weeks vs mandate (ties to `docs/session-optimization.md` P3)?
4. Session policy: new-session-per-phase vs `ctx_purge`-on-threshold.
5. Model tiers per route class + model-scope allowlist.
6. Application to owui-term stays process-only while D8 PENDING.

---

## 8. Confidence & what would change the decision

**High** on mechanism — every rail maps to an existing, file-verified primitive (pi-subagents execution
controls, council-mode, context-mode, repo `session-optimization.md`). **Lower** on unmeasured ROI and on
provider/cache usage comparability (thresholds are same-mission safety gates until accounting is validated
cross-provider).

Would reframe if: ≥10-task measurement shows <10% ctx savings with session growth dwarfing routing gains
(→ collapse to session-split + indexing; defer the router), or pi-coding-agent exposes a first-class
telemetry/ledger API (→ thin policy layer on it).

---

## 9. P0 baseline (snapshot — 2026-08-21)

| Metric | Value |
|---|---|
| Session (this council session) | 125,853 in / 29,316 out · $0.0576 · context 8% |
| context-mode use, this session | 1 call · 551 B entered context · **0 tokens saved** |
| Context-mode lifetime (7 days, 176 conversations) | 12.5K events captured · 2.3 MB kept out |
| Cost | $0.06 this session · $12.67 lifetime |

**Implication (measure-first, per §8):** this P0 session used context-mode almost not at all (0 tokens saved) —
consistent with `docs/session-optimization.md`. Savings claims stay hypotheses until the P1 gate
(≥10 measured accepted-verified units via the `E_unit` ledger) is met. Re-run `ctx_stats` after each measured
task; update this table at the P1 gate.

---

## 10. Evidence & run ids

- Council runs: pass 1 `10a21a82-4f35-45b7-abdd-b34adfc77e94` (children `a9ee8a07…`/`84ecc531…`/`843e4dbf…`),
  pass 2 `28b2081f-950d-458f-a23b-3b82ea99d846` (fresh-context fallback cross-exams `19683c4b…`/`0492f1bd…`/`58dcff14…`).
- Grounding sources advisors verified in-file: `docs/session-optimization.md`, pi-subagents
  `SKILL.md` + references (execution-controls, constraints-and-recipes, prompting-and-roles) + `CHANGELOG`,
  `council-mode/SKILL.md`, context-mode skill docs, `pi-subagents/src/shared/types.ts` +
  `missions/types.ts`/`goal-driver.ts` (telemetry fields), agent/skill inventories.
