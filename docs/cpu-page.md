# CPU page

CORE-03 replaces the `/cpu` placeholder with the first data-backed monitoring page. It consumes only the validated same-origin CORE-02 API and does not add backend operations or browser storage.

## Current evidence

Overall CPU appears first with its freshness-aware health badge and the confirmed 85% warning and 95% Critical thresholds. Stale measurements remain visible as last-known evidence with their timestamp; unavailable values remain `N/A` and are never estimated.

The page also shows Linux 1, 5, and 15-minute load averages. Load is deliberately presented without a percent sign because it represents runnable or waiting work rather than utilization.

Logical vCPUs are discovered from the current response rather than assuming six fixed indexes. Their disclosure starts collapsed, preserves non-contiguous kernel indexes, and shows every returned core when expanded.

## History interaction

The measurement selector supports overall CPU, every currently detected logical vCPU, and all three load averages. The range selector supports:

- last minute;
- last 5, 15, and 30 minutes;
- last 1, 6, and 24 hours;
- last 7 and 14 days;
- a custom start and end interpreted in `Africa/Tunis` and sent as UTC.

Custom input rejects malformed values, an end before the start, future periods, and periods longer than 14 days before contacting the API. The backend remains authoritative and validates the request again.

Minimum and peak use the extrema returned for observed buckets. The period average is weighted by each bucket's available-sample count, so longer aggregated buckets are not accidentally given the same weight as a single short bucket.

## Chart honesty and accessibility

Apache ECharts renders minimum, average, and peak series. Percent charts show the warning and Critical lines. Unavailable buckets and collection gaps become `null` chart points with line connection disabled, so the browser cannot visually invent measurements.

The canvas is decorative for assistive technology. Every chart includes a localized text summary and an expandable semantic table containing timestamps, evidence states, and observed values. Tooltips use ECharts rich-text rendering rather than inserting API labels as HTML.

## Responsive and localization behavior

The desktop layout uses paired current cards and multi-column vCPU/statistic grids. Breakpoints collapse these into one column for Android-sized screens while retaining full controls and horizontally scrollable history tables. English and French strings cover headings, controls, empty/error/stale states, thresholds, summaries, and table labels. All timestamps display in `Africa/Tunis`.

## Verification

Run focused CPU page tests:

```text
npm --prefix web test -- --run src/features/cpu/model.test.ts src/features/cpu/CPUPage.test.tsx
```

Run complete repository verification:

```text
npm run check:full
```
