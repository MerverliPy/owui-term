# owui-term

Terminal client for [Open-WebUI](https://openwebui.com) — chat with your self-hosted models from the terminal. **Server state is the source of truth**: chats you start or resume here persist on the server, exactly like the web UI.

> 🚧 Early prototype — status: Phase 1 (reconnaissance) in progress. See [`PHASES.md`](PHASES.md) for the plan and `.brainstorm/DECISION-MEMO.md` for the design rationale.

## Install (not yet available)

```sh
go install <module-path>/cmd/owui-term@latest   # placeholder
```

## Quickstart (planned)

```sh
export OWUI_URL=http://localhost:3000
export OWUI_TOKEN=sk-...
owui-term
```

## Docs

- [`PHASES.md`](PHASES.md) — implementation plan (D1–D8 locked constraints)
- [`docs/api-notes.md`](docs/api-notes.md) — verified Open-WebUI API surface
- [`docs/supported-versions.md`](docs/supported-versions.md) — version pinning policy
- [`docs/environment.md`](docs/environment.md) — verified dev environment
- [`docs/api-setup.md`](docs/api-setup.md) — authentication setup
- [`.brainstorm/DECISION-MEMO.md`](.brainstorm/DECISION-MEMO.md) — council decision memo

## License

MIT (to be added).
