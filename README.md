# Infrastructure Monitor

Infrastructure Monitor is a private dashboard for monitoring and administering one Ubuntu VPS through a Headscale-managed Tailscale network.

The former Windows/Python/Vinext/Cloudflare prototype has been retired from the active development line. It remains recoverable from the Git tag `legacy-prototype-v0.2.0`.

The replacement application uses Go for the backend and React/TypeScript/Vite for the frontend. Its private HTTP lifecycle, configuration foundation, storage foundation, and API contracts are implemented; monitoring collectors and product pages are not implemented yet.

The approved project documents are:

- [`PROJECT_CONTEXT.md`](PROJECT_CONTEXT.md)
- [`TECHNICAL_PLAN.md`](TECHNICAL_PLAN.md)
- [`TASKS.md`](TASKS.md)
- [`docs/implementation-prerequisites.md`](docs/implementation-prerequisites.md)
- [`docs/api-contracts.md`](docs/api-contracts.md)
- [`docs/runtime-foundation.md`](docs/runtime-foundation.md)
- [`docs/storage-schema.md`](docs/storage-schema.md)

Implementation is task-gated. Only a task explicitly started by the user may be implemented.

## Source layout

- `cmd/pim/` - application executable entry point.
- `internal/` - private backend package boundaries.
- `web/` - frontend source and dependency lockfile.
- `configs/` - administrator configuration example.
- `deploy/` - future systemd and firewall assets.
- `tests/` - future integration and end-to-end tests.

The FND-04 Go binary embeds a tracked placeholder page so clean Go builds are self-contained. The React build is written to ignored `internal/web/dist/`; FND-05 will connect those production assets to the embedded handler.

## Private runtime foundation

On Linux, `pim` discovers the IPv4 address assigned to `tailscale0` and binds its configured port only to that exact address. Startup fails if the interface is missing, down, has no suitable IPv4 address, or cannot be bound. There is no wildcard, public-interface, or localhost fallback.

The current private endpoints are:

- `/` - embedded placeholder UI.
- `/api/v1/health` - foundation status for configuration, history storage, audit storage, and maintenance mode.

Production defaults use `/etc/pim/settings.toml`, `/var/lib/pim/last-valid-settings.toml`, `/var/lib/pim/history.db`, and `/var/lib/pim/audit.db`. The settings directory must exist before startup; later deployment tasks will provision it and the service account. See [`docs/runtime-foundation.md`](docs/runtime-foundation.md) for lifecycle and security details.

## Development and quality commands

Required development tools:

- Go 1.26.x.
- Node.js 24.x and npm.

Install the locked dependencies after cloning:

```text
npm ci --ignore-scripts --no-audit --no-fund
npm --prefix web ci --ignore-scripts --no-audit --no-fund
```

Run `npm run quality` from the repository root to print the available commands.

| Command | Result |
| --- | --- |
| `npm run format` | Formats Go and frontend source files. |
| `npm run format:check` | Checks formatting without changing files. |
| `npm run lint` | Runs Go vet and strict TypeScript checking. |
| `npm test` | Runs scoped Go unit tests and frontend Vitest tests. |
| `npm run build` | Builds the Go executable and production frontend assets into ignored local directories. |
| `npm run deps:review` | Verifies Go modules, lists direct frontend packages, and runs the npm vulnerability audit. This command contacts the npm registry. |
| `npm run check` | Fast local verification: formatting, static checks, and tests. |
| `npm run check:full` | Runs the fast check, both builds, and dependency review. |

`npm run check` is the default command during feature work. After dependencies are installed, it does not require network access, Docker, the production Docker socket, VPS access, configuration files, or secrets.

Integration and browser suites will be added by their later roadmap tasks. Empty test directories are not reported as passing test suites.

On Windows, the quality runner automatically uses `.tools/go/bin/go.exe` when that ignored portable toolchain exists. `PIM_GO` and `PIM_GOFMT` can point to alternative binaries without changing repository files.
