# owui-term

Terminal client for [Open-WebUI](https://openwebui.com) — chat with your self-hosted models from the terminal. **Server state is the source of truth**: chats you start or resume here persist on the server, exactly like the web UI.

> 🚧 Prototype slice: Phase 5 (validation) now active — **create + stream + reload verified against a live Open-WebUI instance**, with quality gates and demand validation docs started. See [`PHASES.md`](PHASES.md) and `docs/validation.md` for plan + verdict.

## Build

Requires Go ≥ 1.23. `owui-term --version` reports the build version.

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

## Docs

- [`PHASES.md`](PHASES.md) — implementation plan (D1–D8 locked constraints)
- [`docs/acceptance-test.md`](docs/acceptance-test.md) — D4 acceptance run (create → stream → reload → verify)
- [`docs/api-notes.md`](docs/api-notes.md) — verified Open-WebUI API surface
- [`docs/supported-versions.md`](docs/supported-versions.md) — version pinning policy
- [`docs/environment.md`](docs/environment.md) — verified dev environment
- [`docs/api-setup.md`](docs/api-setup.md) — authentication setup
- [`docs/dogfood.md`](docs/dogfood.md) — owner dogfooding log (1–2 week D8 validation trial)
- [`docs/validation.md`](docs/validation.md) — quality gates and go/no-go verdict record
- [`.brainstorm/DECISION-MEMO.md`](.brainstorm/DECISION-MEMO.md) — council decision memo

## License

MIT.

