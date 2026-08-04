# Infrastructure Monitor

Infrastructure Monitor is a private dashboard for monitoring and administering one Ubuntu VPS through a Headscale-managed Tailscale network.

The former Windows/Python/Vinext/Cloudflare prototype has been retired from the active development line. It remains recoverable from the Git tag `legacy-prototype-v0.2.0`.

The replacement application uses Go for the backend and React/TypeScript/Vite for the frontend. The current source tree is a buildable scaffold only; monitoring features have not been implemented yet.

The approved project documents are:

- [`PROJECT_CONTEXT.md`](PROJECT_CONTEXT.md)
- [`TECHNICAL_PLAN.md`](TECHNICAL_PLAN.md)
- [`TASKS.md`](TASKS.md)
- [`docs/implementation-prerequisites.md`](docs/implementation-prerequisites.md)

Implementation is task-gated. Only a task explicitly started by the user may be implemented.

## Scaffold layout

- `cmd/pim/` - application executable entry point.
- `internal/` - private backend package boundaries.
- `web/` - frontend source and dependency lockfile.
- `configs/` - future administrator configuration examples.
- `deploy/` - future systemd and firewall assets.
- `tests/` - future integration and end-to-end tests.

The frontend build is written to `internal/web/dist/` for later embedding and is intentionally excluded from Git.
