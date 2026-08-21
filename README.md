# owui-term

Terminal client for [Open-WebUI](https://openwebui.com) — chat with your self-hosted models from the terminal. **Server state is the source of truth**: chats you start or resume here persist on the server (write-back verified against Open-WebUI **v0.11.0**; the prototype is silent on write-back failure).

> 🚧 Phase 5 (validation) **complete** — create + stream + reload verified against a live Open-WebUI v0.11.0; quality bar green. D8 demand validation (owner dogfooding + ~5-operator trial) is **PENDING** — see [`PHASES.md`](PHASES.md) and `docs/validation.md`.

## Build

Requires Go ≥ 1.23 and an Open-WebUI instance you can reach (verified: **v0.11.0**; v0.10.2 not yet verified) with API keys enabled. `owui-term --version` reports the build version.

```sh
go build -o owui-term ./cmd/owui-term
```

## Install

From source:

```sh
# local, fast path
go install ./cmd/owui-term

# or explicit build
go build -o owui-term ./cmd/owui-term
```

## Quickstart

```sh
export OWUI_URL=http://localhost:3000
export OWUI_TOKEN=sk-...
# then run
owui-term
```

`OWUI_URL` is your server's browser origin; `OWUI_TOKEN` is your own non-admin API key from that instance. If API keys are disabled on the server, an admin must enable `ENABLE_API_KEYS` first (see `docs/api-setup.md`).

## Docs

- [`PHASES.md`](PHASES.md) — implementation plan (D1–D8 locked constraints)
- [`docs/acceptance-test.md`](docs/acceptance-test.md) — D4 acceptance run (create → stream → reload → verify)
- [`docs/api-notes.md`](docs/api-notes.md) — verified Open-WebUI API surface
- [`docs/supported-versions.md`](docs/supported-versions.md) — version pinning policy
- [`docs/environment.md`](docs/environment.md) — verified dev environment
- [`docs/api-setup.md`](docs/api-setup.md) — authentication setup
- [`docs/dogfood.md`](docs/dogfood.md) — owner dogfooding log (1–2 week D8 validation trial)
- [`docs/validation.md`](docs/validation.md) — quality gates and go/no-go verdict record
- `.brainstorm/DECISION-MEMO.md` — council decision memo (**private, not in this repo**; its locked constraints D1–D8 are reproduced in [`PHASES.md`](PHASES.md))

## License

MIT.

