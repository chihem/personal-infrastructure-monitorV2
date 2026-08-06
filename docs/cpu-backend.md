# CPU collection and API

CORE-02 provides the first real monitoring collector and data-backed product API. It collects overall CPU usage, every currently visible logical CPU, and Linux 1/5/15-minute load averages. CORE-03 now presents those contracts on the dedicated CPU page.

## Collection source and privacy

The collector uses pinned `github.com/shirou/gopsutil/v4` CPU-times and load-average APIs. On Linux these APIs read cumulative CPU counters from `/proc/stat` and load averages from `/proc/loadavg`. The monitor does not enumerate processes, read command lines, inspect environments, or execute shell commands.

Only the previous normalized counter set and latest CPU snapshot are held in memory. Collection remains bounded by the scheduler's ten-second provider deadline and one-minute cadence.

## CPU percentage calculation

CPU counters are cumulative, so a percentage requires two readings. For each stable logical CPU and the overall sum:

1. Calculate total counter growth.
2. Treat idle plus I/O-wait growth as non-busy time.
3. Calculate `(total delta - idle delta) / total delta * 100`.

Guest counters are not added separately because Linux already includes them in user/nice counters. Values must remain finite, monotonic, and between 0 and 100.

The first reading is a baseline: CPU percentages are explicitly unavailable with `not_collected`, while valid load averages remain available. A counter reset, wrap, or zero-total delta is unavailable with `collector_error`; it is never changed to zero.

Logical CPUs are rediscovered on every reading and identified by their numeric kernel index. Indexes need not be contiguous. When topology changes, existing cores with a valid prior counter remain usable, new cores are unavailable for one reading, removed cores disappear, and overall usage waits for one stable topology interval.

## Scheduling and persistence

The production runtime now starts the existing scheduler. Runs remain aligned to UTC minute boundaries, host and Docker providers retain their independent results, and every completed run is offered to the history recorder.

The CPU snapshot maps into one `host_samples` row plus one `cpu_core_samples` row per detected logical CPU in the same CORE-01 transaction. A total host-collector failure stores the collection-run failure but no fabricated sample, so history contains a real gap.

Docker remains outside CORE-02. Until CORE-08 replaces it, an explicit provider returns `docker_not_implemented` without opening the Docker socket or making a Docker request. This keeps operational status honest.

## Current endpoint

`GET /api/v1/cpu` returns the validated `CPUSnapshot` contract:

- overall percentage;
- sorted dynamic logical CPU indexes and percentages;
- 1/5/15-minute load averages;
- logical CPU count;
- independent metric availability and freshness.

Before the first collection, the endpoint returns `200` with unavailable `not_collected` metrics and an empty core array. After two minutes without a new usable CPU snapshot, retained evidence becomes stale instead of being presented as current.

## History endpoint

`GET /api/v1/cpu/history` requires one metric and one approved range.

Examples:

```text
/api/v1/cpu/history?metric=overall&range=last_1h
/api/v1/cpu/history?metric=core&core=4&range=last_6h
/api/v1/cpu/history?metric=load_5&range=last_24h
/api/v1/cpu/history?metric=overall&range=custom&start=2026-08-05T10:00:00Z&end=2026-08-05T11:00:00Z
```

Allowed metrics are `overall`, `core`, `load_1`, `load_5`, and `load_15`. `core` requires one non-negative index; other metrics reject it. Unknown, repeated, non-UTC, future, or out-of-retention parameters are rejected.

Each response carries the resolved UTC range, unit, bucket duration, at most 600 points, sample counts, and minimum/average/maximum for observed buckets. Unavailable buckets and collection gaps remain different states.

## Verification

Run the focused backend suites:

```text
.\.tools\go\bin\go.exe test ./internal/collector/host/cpu ./internal/scheduler ./internal/storage/history ./internal/api ./internal/app -count=1
```

Run all repository checks:

```text
npm run check
npm run check:full
```

The Linux-only source adapter test uses synthetic `/proc/stat` and `/proc/loadavg` fixtures. Final release testing still requires one read-only smoke test on the supported Linux environment; Windows unit tests do not prove access to the VPS kernel files.
