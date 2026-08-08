# Infrastructure Monitor

Infrastructure Monitor is a private dashboard for monitoring and administering one Ubuntu VPS through a Headscale-managed Tailscale network.

The former Windows/Python/Vinext/Cloudflare prototype has been retired from the active development line. It remains recoverable from the Git tag `legacy-prototype-v0.2.0`.

The replacement application uses Go for the backend and React/TypeScript/Vite for the frontend. Its private HTTP lifecycle, configuration and storage foundations, API contracts, responsive bilingual UI shell, collection scheduler, operational self-status, bounded history repository, real CPU backend/page, and RAM/swap/PSI backend are implemented. Other monitoring collectors and data-backed product pages are not implemented yet.

The approved project documents are:

- [`PROJECT_CONTEXT.md`](PROJECT_CONTEXT.md)
- [`TECHNICAL_PLAN.md`](TECHNICAL_PLAN.md)
- [`TASKS.md`](TASKS.md)
- [`docs/implementation-prerequisites.md`](docs/implementation-prerequisites.md)
- [`docs/api-contracts.md`](docs/api-contracts.md)
- [`docs/runtime-foundation.md`](docs/runtime-foundation.md)
- [`docs/storage-schema.md`](docs/storage-schema.md)
- [`docs/ui-foundation.md`](docs/ui-foundation.md)
- [`docs/scheduling.md`](docs/scheduling.md)
- [`docs/observability.md`](docs/observability.md)
- [`docs/history.md`](docs/history.md)
- [`docs/cpu-backend.md`](docs/cpu-backend.md)
- [`docs/cpu-page.md`](docs/cpu-page.md)
- [`docs/memory-backend.md`](docs/memory-backend.md)

Implementation is task-gated. Only a task explicitly started by the user may be implemented.

## Source layout

- `cmd/pim/` - application executable entry point.
- `internal/` - private backend package boundaries.
- `web/` - frontend source and dependency lockfile.
- `configs/` - administrator configuration example.
- `deploy/` - future systemd and firewall assets.
- `tests/` - future integration and end-to-end tests.

Ordinary Go tests and development builds embed a small tracked fallback page and do not require Node.js. `npm run build` builds the React application first, verifies the production embed, and then produces a Go binary containing the real ignored Vite output from `internal/web/dist/`.

## Frontend foundation

The dark responsive shell provides routes for Overview, CPU, Memory, Filesystems, Docker, Events, Audit, and Backups. CPU is now a complete data-backed monitoring page; the remaining product routes stay honest placeholders until their roadmap tasks add data.

English and French can be switched at runtime. The first visit follows a supported browser language and otherwise uses English; only the `pim.language` preference is stored in browser storage. Shared status badges use text, icons, and color together. See [`docs/ui-foundation.md`](docs/ui-foundation.md) for routes, state primitives, API-query defaults, and testing boundaries.

## Collection scheduling foundation

The scheduler aligns automatic collection to one-minute boundaries, rejects overlapping manual refreshes, runs host and Docker providers with explicit deadlines and cancellation, and retains partial results without inventing missed samples. The host provider now composes real CPU and memory collectors; Docker remains an explicit unavailable provider until its core-feature task. See [`docs/scheduling.md`](docs/scheduling.md) for the execution and failure rules.

## History foundation

The history repository atomically records collection-run, host CPU/memory, and dynamic per-vCPU rows. It resolves every approved UTC range, returns raw one-minute chart positions through six hours, aggregates longer ranges to at most 600 buckets while retaining minimum/average/maximum, and distinguishes unavailable evidence from true gaps. Rolling cleanup removes only data older than 14 days in bounded transactions. See [`docs/history.md`](docs/history.md).

## CPU backend

The production scheduler now reads cumulative CPU counters and load averages once per minute, discovers logical CPUs dynamically, and persists each completed CPU snapshot. `GET /api/v1/cpu` exposes current/freshness-aware evidence, while `GET /api/v1/cpu/history` exposes bounded overall, per-vCPU, and load history. First readings, topology changes, counter resets, unavailable data, and gaps remain explicit. See [`docs/cpu-backend.md`](docs/cpu-backend.md).

The dedicated CPU page presents the current overall reading first, expandable logical-vCPU details, Linux load averages, 85%/95% threshold indicators, every approved history range, custom Africa/Tunis periods, and minimum/average/peak summaries. Its ECharts visualization leaves gaps disconnected and is paired with a readable summary and expandable table. See [`docs/cpu-page.md`](docs/cpu-page.md).

## Memory backend

The production host collector now reads aggregate Linux memory counters and PSI once per minute. It keeps total, used, available, free, cached, and buffered memory distinct; reports configured or absent swap honestly; and retains ordinary RAM evidence when PSI alone becomes unavailable. `GET /api/v1/memory` exposes current/freshness-aware evidence, while `GET /api/v1/memory/history` exposes bounded memory, swap, and selected pressure history. See [`docs/memory-backend.md`](docs/memory-backend.md).

## Private runtime foundation

On Linux, `pim` discovers the IPv4 address assigned to `tailscale0` and binds its configured port only to that exact address. Startup fails if the interface is missing, down, has no suitable IPv4 address, or cannot be bound. There is no wildcard, public-interface, or localhost fallback.

The current private endpoints are:

- `/` and frontend routes - embedded responsive React UI shell.
- `/api/v1/health/live` - process liveness.
- `/api/v1/health/ready` - readiness of the required configuration and history-storage dependencies.
- `/api/v1/health` - compatibility alias for readiness.
- `/api/v1/status` - bounded operational state, dependency states, database sizes, and collection timing.
- `/api/v1/cpu` and `/api/v1/cpu/history` - current and historical CPU evidence.
- `/api/v1/memory` and `/api/v1/memory/history` - current and historical RAM, swap, and pressure evidence.

Production defaults use `/etc/pim/settings.toml`, `/var/lib/pim/last-valid-settings.toml`, `/var/lib/pim/history.db`, and `/var/lib/pim/audit.db`. The settings directory must exist before startup; later deployment tasks will provision it and the service account. See [`docs/runtime-foundation.md`](docs/runtime-foundation.md) for lifecycle and security details.

Requests receive a server-generated `X-Request-ID`, and operational events are emitted as allowlisted JSON to stdout or stderr. Raw client addresses are hashed and request headers, tokens, environment values, container logs, and arbitrary error text are never accepted as log fields. See [`docs/observability.md`](docs/observability.md) for the exact status and logging semantics.

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

Component tests cover the current frontend foundation. Full integration and multi-browser suites will be added by their later roadmap tasks; empty future test directories are not reported as passing suites.

On Windows, the quality runner automatically uses `.tools/go/bin/go.exe` when that ignored portable toolchain exists. `PIM_GO` and `PIM_GOFMT` can point to alternative binaries without changing repository files.
