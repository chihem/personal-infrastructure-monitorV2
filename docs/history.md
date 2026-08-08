# History queries and rolling retention

CORE-01 provides the bounded SQLite history repository used by CPU, memory, filesystem, Docker, chart, and export tasks. CORE-02 stores CPU/load samples, and CORE-04 adds RAM, swap, and selected PSI samples. The remaining collectors and the RAM product page are still pending.

## Collection transactions

`RecordCollection` stores one validated collection run, its optional combined CPU/memory host sample, and its dynamic per-vCPU samples in one short transaction. Either every supplied row commits or none of them does. A failed or partial run retains independent host and Docker status/error codes.

The host record stores approved CPU, load, memory, swap, and memory-PSI fields. Current PSI exposes every kernel window; retained PSI stores `avg10` plus cumulative microseconds for both `some` and `full`. Filesystem, block-device, and container tables remain pending their collectors.

All product timestamps are UTC Unix seconds in SQLite. Integer byte counters remain integers in storage. Chart queries convert the selected numeric metric to a floating-point presentation value; later raw exports will read the typed stored columns directly so integer precision is not discarded.

## Range resolution

All approved presets are supported: one, five, fifteen, and thirty minutes; one, six, and twenty-four hours; seven and fourteen days; and custom UTC boundaries.

Preset ranges use a half-open interval, `[start, end)`. Their end is the next UTC minute boundary, producing exactly the named number of possible one-minute positions. For example, the last-five-minutes range contains five minute positions rather than six boundary points.

Custom boundaries are used exactly as supplied and must:

- be RFC 3339 UTC values at the API boundary;
- have an end after the start;
- end no later than the current time;
- start no earlier than the current 14-day retention cutoff.

## Metric queries and gaps

Only compile-time metric keys can select database tables and columns. Resource identities and all time boundaries remain SQL parameters. A client cannot provide a table name, column name, SQL expression, or unbounded row limit.

The metric allowlist covers all numeric fields already present in the approved history schema. Host metrics reject resource IDs, per-vCPU metrics accept only a non-negative integer index, and resource-specific metrics require a bounded opaque ID.

Ranges through six hours use one-minute buckets. Longer ranges choose the smallest whole-minute bucket that produces at most 600 chart positions. Examples:

- 24 hours: 480 three-minute buckets;
- 14 days: approximately 593 thirty-four-minute buckets.

Every bucket reports minimum, average, and maximum across its available samples, plus observed and available sample counts. Its state is:

- `observed` when at least one usable value exists;
- `unavailable` when collection evidence exists but the metric has no usable value;
- `gap` when no collection evidence exists in that position.

Unavailable samples and gaps remain distinct. No value is interpolated, estimated, or changed to zero.

## Retention

The retention cutoff is exactly `now - 14 days`. Rows older than the cutoff are eligible; rows exactly on the cutoff remain.

One cleanup batch transaction removes bounded groups of:

- expired container state events;
- incidents that ended more than 14 days ago, while keeping active incidents;
- expired collection runs, with their sample rows removed through foreign-key cascades;
- removed filesystem, block-device, and container identities only after their retained samples/events are gone.

Batch size is limited to 1–1,000. A bounded multi-batch call reports `More: true` if backlog remains, allowing the caller to continue later without one long write lock. Production runs cleanup once at startup and then every 24 hours. A remaining backlog is logged and left for the next bounded run instead of holding one long database write lock.

The later CORE-18 task remains responsible for the separate combined 5 GiB database/backup ceiling. Time-based cleanup never deletes unexpired evidence to satisfy that ceiling.

## Schema compatibility

History schema version 2 adds separate `host_error_code` and `docker_error_code` fields without deleting or rewriting version-1 collection records. Version-1 binaries intentionally refuse the newer schema, so application rollback requires restoration of the pre-migration history backup as documented by the deployment plan.

## Verification

Run the focused storage suite:

```text
.\.tools\go\bin\go.exe test ./internal/storage/history ./internal/storage/migrations ./internal/storage -count=1
```

Run the complete repository suite:

```text
npm run check
```
