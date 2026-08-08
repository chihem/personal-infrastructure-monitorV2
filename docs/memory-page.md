# Memory page

CORE-05 replaces the `/memory` placeholder with a responsive, bilingual page backed by the CORE-04 current and history endpoints.

## Current evidence

The page presents RAM usage first and applies the approved health thresholds:

- Healthy: below 85%.
- Warning: 85% through 94.9%.
- Critical: 95% or higher.
- Unknown: the usage metric is unavailable or its evidence is stale/unavailable.

Total, used, available, free, cached, and buffered RAM remain separate. The help text explains that Linux `MemAvailable` estimates what applications can use without swapping, whereas `MemFree` is completely unused memory. The UI never adds those values together.

Swap has three explicit states:

- configured, with total/used/free measurements;
- not configured, shown as configuration evidence rather than zero usage;
- unknown, when the collector could not establish the configuration.

Linux PSI shows `some` and `full` pressure with 10-, 60-, and 300-second averages plus cumulative stall time. If every pressure metric is unavailable, the page shows one clear unavailable panel while retaining usable RAM evidence. Partial pressure data keeps individual unavailable values marked `N/A`; missing values are never rendered as zero.

## History

The history selector exposes every memory metric allowlisted by the backend:

- RAM total, used, available, free, cached, buffered, and usage;
- swap total and used;
- `some`/`full` PSI 10-second averages and cumulative totals.

All approved presets from the last minute through the last 14 days are available, plus a validated custom Africa/Tunis period. The page reports minimum, sample-weighted average, peak, and gap count. Charts do not connect unavailable or gap buckets. Warning and Critical lines appear only for RAM-usage history; PSI percentages are not compared with RAM-capacity thresholds.

Each canvas chart is paired with a text summary and expandable data table so the evidence is still available without reading the visualization.

## Deferred Docker attribution

The container RAM ranking region deliberately reports that Docker memory data is not connected. It also has a tested empty state for the later case where Docker is available but returns no current containers. CORE-08 will collect Docker evidence and CORE-09 will connect real container values; CORE-05 does not invent names, usage, or links.

## States and refresh

Current and history queries follow the shared one-minute refresh policy. Manual refresh invalidates only memory queries. Current request failure, history failure, stale current evidence, first-collection unavailability, no swap, missing PSI, gaps, and unavailable buckets remain visibly distinct.

## Responsive and localization behavior

All page copy is available in English and French. Desktop and Android-width checks cover the current cards, pressure detail, selectors, summaries, chart fallback, and deferred ranking. Long French pressure labels wrap rather than truncate, and the page does not create page-level horizontal overflow at a 390-pixel viewport.

## Verification

Run focused checks:

```text
npm --prefix web run typecheck
npm --prefix web test -- --run src/features/memory/model.test.ts src/features/memory/MemoryPage.test.tsx
```

Run the complete repository gate:

```text
npm run check:full
```

Browser verification can use a temporary same-origin fixture server for populated Linux evidence. The ordinary Windows Vite server intentionally exercises the API-error state because it does not provide the production Go endpoints. Real Linux `/proc` acceptance remains a later read-only Light-agent smoke check on the VPS.
