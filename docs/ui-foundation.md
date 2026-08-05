# UI foundation

FND-05 provides the shared presentation and browser-state foundation for later monitoring features. It deliberately shows Unknown or not-yet-available states instead of invented measurements.

## Routes

| Route | Current purpose |
| --- | --- |
| `/` | Honest overview shell and links to primary monitoring areas |
| `/cpu` | CPU feature placeholder |
| `/memory` | Memory feature placeholder |
| `/filesystems` | Filesystem feature placeholder |
| `/docker` | Docker feature placeholder |
| `/events` | Warning/Critical events placeholder |
| `/audit` | Administrative audit placeholder |
| `/backups` | Backup and recovery placeholder |
| Any other route | Localized not-found page |

Wouter owns client-side navigation. The Go embedded-file handler returns the SPA entry point for extensionless paths, so a direct browser request to a known route can load the application. The earlier React Router recommendation was replaced because the npm registry reported high-severity advisories for every compatible published line checked during FND-05; the locked Wouter dependency passed the high-severity audit.

## Localization

- English and French resources contain the same tested key set.
- A stored `pim.language` value takes priority.
- On a first visit, the first supported English or French browser language is selected.
- Unsupported browser languages fall back to English.
- Switching language updates visible content, the root HTML `lang` attribute, and the document title without reloading.
- No browser preference other than `pim.language` is persisted.
- `formatDashboardTime` always formats future API timestamps in `Africa/Tunis`.

## Shared UI states

`StatusBadge` supports Healthy, Warning, Critical, and Unknown. Every state combines a distinct Lucide icon, a written label, and color; color is never the only signal.

`StatePanel` supports loading, empty, stale, unavailable, and error states. Feature tasks should reuse these primitives rather than creating ambiguous blank sections or fake zero values.

## API-query foundation

- `requestAPI` accepts only same-origin `/api/v1/` paths.
- Requests ask for JSON, disable browser caching, retain same-origin credentials, and reject redirects.
- Failed responses produce a bounded status error and do not reflect arbitrary backend response bodies.
- TanStack Query defaults to the approved one-minute foreground refresh interval, one retry for reads, cancellation through query signals, and no mutation retry.
- No monitoring query is active in FND-05 because the data endpoints do not exist yet.

## Responsive and accessibility behavior

- Dark theme only, with global design tokens and component-scoped CSS Modules.
- Desktop uses a stable side navigation; widths at or below 48 rem use a horizontally scrollable navigation row.
- Monitoring cards collapse from four columns to two and then one.
- Interactive controls meet a 44 px-class touch target where practical.
- A keyboard skip link, visible focus ring, semantic landmarks, active-route `aria-current`, and reduced-motion handling are included.
- The language buttons retain full accessible names while showing compact `EN`/`FR` labels on narrow mobile screens.

## Production embedding

Normal Go tests use the tracked fallback assets through the default build tag. The release quality command performs these steps in order:

1. Type-check and build the Vite frontend into ignored `internal/web/dist/`.
2. Run the production-tag Go embed smoke test, including retrieval of the generated JavaScript asset.
3. Build `cmd/pim` with the `production` tag so the real React bundle is embedded in the executable.

This keeps clean Go development independent of Node while preventing release binaries from accidentally containing only the fallback page.

## Remaining browser boundary

FND-05 component tests cover routing, runtime localization, preference persistence, state semantics, query defaults, keyboard route activation, and an Android-sized render. Browser inspection additionally verifies the current desktop/mobile visual layout and console state.

Release-wide automated Chromium/Firefox workflows and the real Opera GX smoke test remain assigned to the later testing and release tasks. Feature-specific pages must add their own bilingual, responsive, loading, stale, unavailable, and keyboard tests.
