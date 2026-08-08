# Memory backend

CORE-04 adds aggregate Linux RAM, swap, and memory-pressure collection to the same one-minute host run used by CPU. It does not inspect processes, command lines, environments, or memory contents.

## Linux inputs and definitions

The collector reads only:

- `/proc/meminfo`
- `/proc/pressure/memory`

The accepted `meminfo` fields are `MemTotal`, `MemAvailable`, `MemFree`, `Cached`, `Buffers`, `SwapTotal`, and `SwapFree`. Values must be non-negative `kB` counters and are converted to integer bytes with overflow checks. Unknown fields are ignored.

The exposed RAM meanings are deliberately distinct:

- **Total:** `MemTotal`.
- **Available:** `MemAvailable`, the kernel estimate of memory usable by new applications without swapping.
- **Used:** `MemTotal - MemAvailable`.
- **Usage percentage:** `used / MemTotal * 100`.
- **Free:** `MemFree` only.
- **Cached:** `Cached` only.
- **Buffered:** `Buffers` only.

The collector does not add cache to free memory or estimate a missing `MemAvailable`. Missing, malformed, duplicated, inconsistent, or overflowing required counters become unavailable metrics instead of generated values.

When `SwapTotal` is zero, swap is explicitly not configured and its numeric metrics carry `not_configured`; zero is not presented as observed swap usage. When swap exists, used bytes are `SwapTotal - SwapFree`. Before `SwapTotal` has been read successfully, `configured` is `null` rather than a fabricated `false`.

## Pressure behavior

The kernel PSI format provides `some` and `full` rows. Each row exposes `avg10`, `avg60`, `avg300`, and cumulative `total` microseconds. Current API evidence retains all eight values.

- `some` means at least some non-idle tasks were stalled on memory.
- `full` means all non-idle tasks were stalled simultaneously, representing memory thrashing.

If the PSI file is missing, unreadable, or malformed, all pressure fields become unavailable with a reason. Ordinary RAM and swap evidence stays available, and the host collection is partial. The future health task will turn this into the approved Unknown pressure condition; zero is never substituted.

Retained history stores the selected high-signal PSI fields already reserved by the schema: `avg10` and cumulative `total` for both `some` and `full`. Current 60/300-second windows remain available through the current endpoint.

## Composite host collection and persistence

The production host provider composes independent CPU and memory collectors. A failure in one does not discard valid evidence from the other. Both snapshots are mapped into one host row and dynamic CPU-core rows in the same SQLite transaction.

Each metric history bucket remains one of:

- `observed`: at least one usable value exists;
- `unavailable`: collection ran, but the selected metric had no usable value;
- `gap`: no collection evidence exists for that position.

Long ranges retain minimum, average, maximum, observed count, and available count without interpolation.

## HTTP API

### Current memory

`GET /api/v1/memory`

Returns current/freshness-aware total, used, available, free, cached, buffered, percentage, swap, and all PSI windows. Before the first collection, the endpoint returns a validated unavailable snapshot rather than fabricated host values.

### Memory history

`GET /api/v1/memory/history?metric=<metric>&range=<range>`

Allowlisted metrics:

- `total`, `used`, `available`, `free`, `cached`, `buffered`, `usage`
- `swap_total`, `swap_used`
- `pressure_some_avg10`, `pressure_full_avg10`
- `pressure_some_total`, `pressure_full_total`

All approved ranges are accepted. Custom requests additionally require RFC 3339 UTC `start` and `end`. Unknown, repeated, non-UTC, future, expired, or oversized ranges are rejected. Responses are bounded to 600 buckets.

The frontend runtime parsers independently validate resource kind, units, freshness, metric availability, range shape, bucket counts, summaries, and gap semantics before the CORE-05 page renders the data.

## Deferred container ranking

CORE-04 does not invent container RAM usage. CORE-05 presents an explicit unavailable ranking section and a tested empty-state contract. CORE-08 later supplies normalized Docker memory measurements, and CORE-09 connects those measurements to the RAM page. See [`memory-page.md`](memory-page.md).

## Verification

Run focused backend and contract tests:

```text
.\.tools\go\bin\go.exe test ./internal/collector/host/... ./internal/storage/history ./internal/api/... ./internal/app -count=1
npm --prefix web test -- --run src/api/contracts.test.ts
```

Run the complete repository gate:

```text
npm run check:full
```

Local Windows tests use fixed Linux fixture text and injected read failures. Final acceptance still requires a read-only Hermes smoke check through Light against its real `/proc` files; no direct SSH or VPS change is part of CORE-04.
