# API Contract v1

Status: **Defined through CORE-04**
Wire version: `v1`
Transport timestamps: RFC 3339 in UTC

## Scope

This document defines the data boundary shared by the Go backend and TypeScript frontend. FND-07 exposes the operational contracts at the private routes listed below, CORE-02 adds CPU routes, and CORE-04 adds RAM/swap/PSI routes. Other monitoring domains, Docker calls, and data-backed UI screens remain later roadmap tasks.

The Go contract is under `internal/api/contracts/`. The TypeScript mirror and its safe runtime parsers are under `web/src/api/`. Shared examples are under `tests/fixtures/contracts/v1/`.

## Versioned envelope

Every bounded JSON response contains:

- `apiVersion`: exactly `v1`.
- `requestId`: an opaque identifier used to correlate a safe client error with server logs.
- `generatedAt`: the UTC response time.
- `data`: the successful payload, otherwise `null`.
- `error`: a structured error, otherwise `null`.

Exactly one of `data` and `error` is non-null. Implemented API routes live under `/api/v1`.

## Operational self-status

The private server currently exposes:

| Path | Successful response | Failure behavior |
| --- | --- | --- |
| `/api/v1/health/live` | `200` with `alive: true` while the process can serve HTTP | No response when the process or listener is down |
| `/api/v1/health/ready` | `200` when required dependencies are usable | `503` during maintenance or when configuration/history storage is unavailable |
| `/api/v1/health` | Compatibility alias for readiness | Same as `/api/v1/health/ready` |
| `/api/v1/status` | `200` with the validated operational snapshot | `500` with a safe API error if an invalid snapshot reaches the handler |
| `/api/v1/cpu` | `200` with current or honestly unavailable CPU evidence | `500` if invalid internal data reaches the handler; `503` if the CPU source is not wired |
| `/api/v1/cpu/history` | `200` with bounded raw/aggregated CPU history | `400` for invalid query parameters; `503` when history is unavailable |
| `/api/v1/memory` | `200` with current or honestly unavailable RAM, swap, and PSI evidence | `500` if invalid internal data reaches the handler; `503` if the memory source is not wired |
| `/api/v1/memory/history` | `200` with bounded raw/aggregated memory history | `400` for invalid query parameters; `503` when history is unavailable |

Readiness requires a usable current or last-valid configuration, an available history database, and inactive maintenance mode. Audit-storage failure, use of previous settings, collectors that have not started, and unchecked Docker connectivity produce honest degraded or placeholder states without making read-only service unready.

Operational state is one of `ok`, `degraded`, `maintenance`, or `not_ready`. Dependency fields use only `available`, `unavailable`, `not_started`, `not_implemented`, or `not_checked` where applicable. Database sizes may be `null` when they cannot be read safely. Collection timing is reported only after a completed run, and its duration must match its UTC start and finish timestamps.

All served envelopes and response headers carry the server-generated request ID. Incoming `X-Request-ID` values are discarded rather than trusted.

## Honest measurement states

### Health

Health states are `healthy`, `warning`, `critical`, and `unknown`.

- `healthy` appears alone and has no causes.
- Warning, Critical, and Unknown may appear together.
- Every visible problem state has at least one matching structured cause.
- Cause and error codes are stable machine-readable values. The UI uses `messageKey` for English/French text.

### Freshness

Freshness is independent of health and measurement availability:

| State | Meaning | Required evidence |
| --- | --- | --- |
| `fresh` | A recent collection succeeded for the resource. | `observedAt` and `lastSuccessfulAt` |
| `stale` | Last-known evidence is older than the stale boundary. | `lastSuccessfulAt`; `observedAt` may identify the retained observation |
| `unavailable` | The resource has never produced usable evidence. | Both timestamps are `null` |

The configured two-minute rule decides when data becomes stale. That calculation belongs to the later health evaluator, not the wire type.

### Metric availability

Every numeric metric is a discriminated object:

- Available: `availability` is `available`, `value` is finite, and `reasonCode` is `null`.
- Unavailable: `availability` is `unavailable`, `value` is `null`, and `reasonCode` explains why.

Unavailable metrics are never encoded as zero. A real measured zero remains an available value.

Units are explicit: `percent`, `bytes`, `bytes_per_second`, `count`, `seconds`, `load`, `microseconds`, or `none`.

### Chart gaps

- An `observed` point contains a metric, which may itself be available or unavailable.
- A `gap` point has `measurement: null` because no collection point exists.

This prevents missing periods from being interpolated or confused with a collector that ran but could not read one metric.

## Covered resources

The v1 types can represent:

- Overall/per-logical-CPU usage and 1, 5, and 15 minute load averages.
- RAM fields, swap configuration, and Linux memory pressure.
- Active/removed filesystems, capacity, mount mode, and optional block I/O.
- Docker communication, container state, health, uptime, restarts, usage, and ports.
- Container state/health events and bounded live-log SSE events.
- Warning, Critical, and Unknown incidents with stable causes.
- Preset/custom ranges, charts, statistics, and opaque cursor paging.
- CSV/JSON exports and administrative audit records/deletion scopes.
- Backup records, degraded recovery, and configuration status.
- Confirmation intents/results for approved Docker, backup, restore, and destructive audit actions. Export requests do not require confirmation.

The action enum intentionally excludes container creation/deletion, exec, image, volume, network, secret, and arbitrary Docker API operations.

## Time, ranges, IDs, and nulls

- API timestamps are RFC 3339 UTC values. Examples use `Z`.
- Africa/Tunis conversion happens only in the presentation layer.
- Preset selections do not carry custom boundaries.
- Custom selections require a UTC start/end with end after start.
- Resolved ranges always contain concrete UTC boundaries.
- Resource IDs and cursors are opaque; clients must not parse them.
- Page limits are bounded from 1 through 200.
- `nextCursor` is non-null only when `hasMore` is true.
- `null` deliberately represents unavailable values and absent timestamps.
- Empty arrays mean a successful result with no matching records.

CPU history accepts `metric=overall|core|load_1|load_5|load_15` and an approved `range`. The core metric additionally requires `core=<non-negative index>`. Custom ranges require UTC `start` and `end`. Unknown and repeated parameters are rejected. History buckets expose start/end, observed/available counts, and minimum/average/maximum; `observed`, `unavailable`, and `gap` remain separate states. See [`cpu-backend.md`](cpu-backend.md).

Memory history accepts the allowlisted metrics documented in [`memory-backend.md`](memory-backend.md) plus an approved `range`. Custom ranges require UTC `start` and `end`; unknown or repeated parameters are rejected. The same bucket and gap semantics apply. Current pressure contains the kernel's 10, 60, and 300-second averages plus cumulative microseconds; retained pressure history stores the selected 10-second averages and cumulative totals.

## Errors and privacy

Errors contain a stable code, localization key, safe English fallback, optional bounded technical detail, and field errors.

Contracts must never contain:

- Go stack traces, panic output, or environment dumps.
- Docker inspect responses or unrestricted Engine objects.
- Private keys, tokens, credentials, or configuration-file contents.
- Container logs inside history or export payloads.

Expandable Docker error detail is bounded to 4 KiB and must be sanitized by the later Docker adapter.

## Compatibility rules

- Breaking field, nullability, unit, enum, or semantic changes require a new API version.
- Additive v1 fields are allowed only when existing clients can safely ignore them.
- Existing enum meanings and stable codes are never repurposed.
- IDs remain opaque across versions.
- Rejected or unknown responses become unavailable/Unknown, never invented values.
- Go and TypeScript changes must update shared fixtures and pass both suites together.

## Shared examples

- `snapshot-complete.json`: current primary collector data.
- `snapshot-partial.json`: collected data with explicitly unavailable fields and previous settings retained.
- `snapshot-stale.json`: retained last-known evidence marked stale.
- `snapshot-unavailable.json`: never-collected evidence with simultaneous Critical and Unknown causes.
- `chart-with-gap.json`: available, collected-unavailable, and true-gap points.
