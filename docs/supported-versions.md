# Supported Open-WebUI versions

Policy (D5): pin **one current minor + one prior**, test against both, degrade gracefully.

| Version | Status | Notes |
|---|---|---|
| **0.11.0** | ✅ Pinned (verified live 2026-08-21) | `GET /api/version` on `localhost:3000`; API surface verified from served `/openapi.json` |
| **0.10.2** | ✅ Selected as prior minor (latest 0.10.x, released 2026-07-01) | Add a pinned 0.10.2 instance to CI for smoke-testing the compatibility matrix |

Client behavior on unsupported versions: clear "unsupported Open-WebUI version" notice + best-effort completions-only mode (D5), never silent failure.
