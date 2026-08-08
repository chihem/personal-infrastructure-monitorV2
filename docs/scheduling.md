# Collection scheduling

FND-06 established the orchestration boundary. CORE-02 connects the real CPU
provider, history recorder, and production runtime. Memory, filesystem, and
real Docker providers remain later tasks.

## Schedule

- Scheduled collection starts on UTC minute boundaries, which are also exact
  minute boundaries in Africa/Tunis.
- The first scheduled run waits for the next boundary instead of inventing an
  earlier sample.
- After a slow run, the scheduler advances to the next future boundary. Missed
  periods remain gaps and are never replayed or backfilled.
- Only one scheduler loop may run for a service instance.

## Manual refresh and overlap

Manual refresh uses the same collection path and produces a run marked
`manual`. If any scheduled or manual collection is already active, a new
manual request returns `ErrRunInProgress`. It does not queue or start duplicate
work.

The service exposes `CollectionInProgress` for later API/UI wiring and keeps the
latest completed run in memory through `LastRun`. `LastSuccessfulRun` retains
the most recent fully successful run so a later partial or failed run does not
erase the last-success evidence used by operational status.

## Provider isolation

Host and Docker providers implement separate interfaces. They run concurrently,
so the maximum collector concurrency is two. Both receive cancellable contexts
and the same explicitly configured deadline.

Provider snapshots remain opaque to the scheduler. The host provider now
combines validated CPU and memory snapshots; one host subcollector may fail
without discarding evidence from the other. A later feature task will replace
the explicit Docker-unavailable provider. The scheduler retains a successful
or partial snapshot even when the other provider fails.

Provider results use bounded machine-readable error codes rather than raw error
messages. A deadline, cancellation, or invalid provider result is converted to
one of these stable codes:

- `collector_timeout`
- `collection_cancelled`
- `invalid_collector_result`

Provider implementations must honor context cancellation. Go cannot forcibly
stop a provider that ignores its context, so tests for every concrete provider
must verify cancellation before production wiring.

## Run outcomes

Every completed run contains validated start and finish timestamps, its trigger,
and independent host and Docker outcomes.

- Both providers succeed: `succeeded`.
- At least one provider returns usable success or partial data while another is
  partial or failed: `partial`.
- Neither provider returns usable data: `failed`.

The result envelope keeps host and Docker snapshots separate. After a run is
stored in memory, an optional recorder persists its metadata and supplied sample
rows. Recorder errors are returned without erasing the in-memory run evidence.
The production runtime retries the scheduler on the next minute cycle and logs
only a stable event code rather than database or collector details.

## Testing

The scheduler tests use a fake clock; they do not wait for real minutes. They
cover minute alignment, skipped periods, overlap rejection, deadline expiry,
cancellation, partial results, shutdown before the first run, and duplicate
scheduler-loop prevention.

Run the focused checks with:

```text
go test ./internal/domain ./internal/collector/... ./internal/scheduler
```

Use the repository quality command for the full project:

```text
npm run check
```
