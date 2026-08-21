# Owner dogfooding log (D8)

## Status

- **Started:** 2026-08-21
- **Owner workflow cadence:** 1–2 weeks, target 1–2 real usage sessions per day
- **Current stage:** Launching trial pack + logging loop

## Trial pack for operators

Goal: validate that the create → open/reload workflow is sticky enough for v1.
Each participant runs:

1. Launch `owui-term`, select a model.
2. Create a new chat and send one prompt.
3. Exit and relaunch (`chat_id` persistence is server-backed).
4. Open the previously created chat in a web browser.
5. Resume from the same chat and verify the last user/assistant pair is present.

**Go/no-go threshold (D8):** majority complete the workflow with positive follow-up use without prompting.

## Operator log

| Date | Operator | Action | Result | Notes |
|---|---|---|---|---|
| 2026-08-21 | owner (starter) | executed weekend acceptance in repo phase 4 | pass | chat create/stream/list/reload persisted in web UI |
| 2026-08-21 | owner (first daily session) | started phase-5 validation log | pending | initial baseline created |

## Daily notes

- _No additional operators contacted yet._
- Target: add ~5 operator sessions over the next 2 weeks.
