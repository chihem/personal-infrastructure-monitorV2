# Infrastructure Monitor - Implementation Roadmap

Status: **Approved for task-by-task implementation**
Based on: `PROJECT_CONTEXT.md` and `TECHNICAL_PLAN.md`
Created: 2026-08-03
Last updated: 2026-08-05
Implementation progress: **SET-01 through FND-07 and CORE-01 complete**

## Roadmap rules

- Work on one task at a time.
- Do not begin a task until the user says `START TASK [ID]`.
- Finish, test, review, and report one task before starting the next.
- Do not silently expand a task or remove an approved requirement.
- Preserve unrelated user changes in the workspace.
- The first read-only usable preview appears after `CORE-11`; the approved MVP is release-ready only after all required tasks through `REL-03` pass.
- The approved MVP remains medium-to-large. This roadmap makes it manageable by splitting it into small results; it does not pretend the full scope is a tiny dashboard.

## Phase 1 - Project setup

### SET-01 - Close implementation prerequisites [Complete]

- **Objective:** Produce a short, verified implementation-readiness record containing the remaining PSI behavior, tailnet address/interface, Docker socket availability, target CPU architecture, browser transport decision, and supported tool versions.
- **Why it is needed:** The technical plan still leaves missing-PSI behavior unresolved, and deployment details must be verified rather than guessed.
- **Files or components likely affected:** `TECHNICAL_PLAN.md`; `docs/implementation-prerequisites.md` (new); no application code.
- **Dependencies:** Approved `PROJECT_CONTEXT.md` and `TECHNICAL_PLAN.md`.
- **Implementation outline:** Resolve the PSI display/status behavior; collect read-only VPS facts through the user-approved Light workflow; confirm direct tailnet HTTP behavior; record exact decisions and blockers.
- **Acceptance criteria:** Every prerequisite is Confirmed or explicitly Blocked; the PSI rule is unambiguous; no secret value is written to the repository; the technical plan contains no unresolved approval question.
- **Tests required:** Review the recorded facts against Light's command evidence; verify the selected Go/Node release lines from official sources; check that documentation contains no real secret or complete Tailscale address.
- **Security considerations:** Never copy Docker secrets, private keys, credentials, or the complete Tailscale address into tracked documentation; do not change the VPS in this task.
- **Estimated difficulty:** Easy.

### SET-02 - Preserve and retire the legacy prototype [Complete]

- **Objective:** Preserve the existing Python/Next/Vinext/Cloudflare prototype in recoverable Git history and leave a clean starting point for the approved Go/React product.
- **Why it is needed:** The approved architecture replaces the prototype, but the old work must remain recoverable and must not be mixed into production accidentally.
- **Files or components likely affected:** Existing `app/`, `monitoring/`, `worker/`, `db/`, prototype tests and configuration; Git tag or branch; root documentation.
- **Dependencies:** `SET-01`.
- **Implementation outline:** Inventory user changes; record the legacy snapshot; verify recovery; remove legacy runtime paths from the active development line while retaining reusable approved UI assets or wording only.
- **Acceptance criteria:** The prototype can be restored from the recorded Git reference; the active tree contains no ambiguous legacy runtime entry point; no user work is lost.
- **Tests required:** Restore/list the preserved reference; run `git diff --check`; verify the active file inventory against the approved folder structure.
- **Security considerations:** Ensure legacy local configuration, tokens, databases, and generated state are not committed during preservation.
- **Estimated difficulty:** Medium.

### SET-03 - Scaffold the Go and React workspace [Complete]

- **Objective:** Create the approved directory structure and minimal buildable Go module plus React/TypeScript/Vite application.
- **Why it is needed:** All later tasks need stable package boundaries and repeatable dependency management.
- **Files or components likely affected:** `go.mod`, `go.sum`, `cmd/pim/`, `internal/`, `web/`, `package.json`, lockfile, TypeScript/Vite configuration, `.gitignore`.
- **Dependencies:** `SET-02`.
- **Implementation outline:** Initialize pinned modules; create empty package boundaries from the technical plan; configure TypeScript strictness; configure frontend asset output for later Go embedding; exclude secrets and runtime state.
- **Acceptance criteria:** Go packages compile; frontend type-checks and builds; dependency lockfiles exist; no product feature is implemented yet.
- **Tests required:** Go compile/test command; frontend type-check/build command; clean-install verification; `git diff --check`.
- **Security considerations:** Inspect initial dependencies; prohibit install scripts where unnecessary; ensure `.env`, databases, backups, credentials, and logs are ignored.
- **Estimated difficulty:** Easy.

### SET-04 - Establish local quality commands [Complete]

- **Objective:** Provide one documented command set for formatting, static checks, unit tests, frontend tests, builds, and dependency review.
- **Why it is needed:** A solo project stays maintainable when every task uses the same verification path.
- **Files or components likely affected:** `Makefile` or equivalent small task runner, lint/test configuration, `README.md` development section.
- **Dependencies:** `SET-03`.
- **Implementation outline:** Add non-destructive cross-platform task entry points; separate fast checks from full integration checks; make failure exit codes reliable.
- **Acceptance criteria:** A fresh checkout can discover and run every quality command; the fast check completes without needing Docker or production secrets.
- **Tests required:** Run every new command once on Windows and the selected Linux development environment where applicable.
- **Security considerations:** Commands must not print environment contents, contact production, modify firewall rules, or use the production Docker socket.
- **Estimated difficulty:** Easy.

## Phase 2 - Minimum technical foundation

### FND-01 - Define domain and API contracts [Complete]

- **Objective:** Define versioned internal models and JSON contracts for measurements, freshness, health states, errors, paging, ranges, and actions.
- **Why it is needed:** Stable contracts prevent backend and frontend features from inventing incompatible representations.
- **Files or components likely affected:** `internal/domain/`, `internal/api/contracts/`, `web/src/api/`.
- **Dependencies:** `SET-04`.
- **Implementation outline:** Define units, nullable/unavailable values, timestamps, status causes, stable error codes, IDs, and versioned response envelopes; document compatibility rules.
- **Acceptance criteria:** Every approved resource can be represented without fake values; Unknown, stale, unavailable, and chart gaps are distinct; contracts use Africa/Tunis only for display, not storage.
- **Tests required:** JSON round-trip and validation tests; contract examples for complete, partial, stale, and unavailable data.
- **Security considerations:** Contracts must not expose filesystem secrets, environment values, raw stack traces, or unrestricted Docker objects.
- **Estimated difficulty:** Medium.

### FND-02 - Implement validated TOML configuration [Complete]

- **Objective:** Load, validate, atomically reload, and retain the last valid application settings.
- **Why it is needed:** Thresholds and operational limits must change safely without restarts, and invalid edits must not corrupt live behavior.
- **Files or components likely affected:** `internal/config/`, `configs/settings.example.toml`, configuration tests.
- **Dependencies:** `FND-01`.
- **Implementation outline:** Define defaults and bounds; parse candidate settings; validate as a whole; atomically swap valid snapshots; debounce file events; maintain a protected last-valid copy; publish reload result events.
- **Acceptance criteria:** Valid edits take effect without restart; invalid edits leave all previous settings active; startup can use the last-valid copy; the UI can later receive a clear configuration-error state.
- **Tests required:** Unit tests for defaults, boundaries, malformed TOML, partial invalidity, reload races, duplicate events, and last-valid startup.
- **Security considerations:** Reject shell commands and arbitrary Docker endpoints; never allow configuration to weaken mandatory tailnet/security controls at runtime.
- **Estimated difficulty:** Medium.

### FND-03 - Create SQLite databases and migrations [Complete]

- **Objective:** Create separate WAL-mode `history.db` and `audit.db` stores with embedded, versioned migrations.
- **Why it is needed:** All history, incidents, administrative records, backup, and recovery work depends on a reliable schema foundation.
- **Files or components likely affected:** `internal/storage/`, `internal/storage/migrations/`, database test fixtures.
- **Dependencies:** `FND-01`.
- **Implementation outline:** Add connection policies, busy timeouts, foreign keys, schema-version tables, initial entities/indexes, transactional migration execution, and compatibility metadata.
- **Acceptance criteria:** Both databases initialize independently; migrations are repeatable and atomic; audit storage can fail without preventing read-only history startup; one-release rollback compatibility is documented per migration.
- **Tests required:** Fresh creation, repeated startup, upgrade, failed migration rollback, concurrency, WAL behavior, and foreign-key tests.
- **Security considerations:** Create database files with restrictive ownership/modes; never place them inside the web root; avoid sensitive Docker/log content in schemas.
- **Estimated difficulty:** Hard.

### FND-04 - Build the application lifecycle and private HTTP server [Complete]

- **Objective:** Start and stop one Go service cleanly, serve a versioned API and embedded frontend directly on the Tailscale address, and handle shutdown safely.
- **Why it is needed:** This is the executable foundation for every collector and browser feature.
- **Files or components likely affected:** `cmd/pim/`, `internal/app/`, `internal/api/`, `internal/web/`.
- **Dependencies:** `FND-02`, `FND-03`.
- **Implementation outline:** Wire dependencies; discover the IPv4 address on `tailscale0`; open only that TCP listener; serve placeholder embedded assets; configure timeouts and body limits; implement graceful shutdown and maintenance-mode plumbing.
- **Acceptance criteria:** The service has no wildcard, public-interface, or localhost production listener; it fails closed without `tailscale0`; starts with valid settings; serves UI/API placeholders; drains requests and closes databases on shutdown.
- **Tests required:** Startup/shutdown integration tests, interface discovery and fail-closed tests, listener-address inspection, timeout tests, and embedded-asset smoke test.
- **Security considerations:** Never fall back to `0.0.0.0` or `::`; set conservative HTTP limits from the start; ignore forwarded-client headers.
- **Estimated difficulty:** Medium.

### FND-05 - Build the responsive bilingual UI shell [Complete]

- **Objective:** Deliver the dark responsive application shell, routing, English/French switching, status primitives, and API-query foundation.
- **Why it is needed:** Feature pages need a consistent accessible presentation layer before data-specific UI work begins.
- **Files or components likely affected:** `web/src/pages/`, `components/`, `styles/`, `i18n/`, router and query-client setup.
- **Dependencies:** `SET-03`, `FND-01`, `FND-04`.
- **Implementation outline:** Add routes, navigation, dark tokens, responsive layout, loading/empty/error components, status badges using icon+text+color, language detection and switching.
- **Acceptance criteria:** Shell works at desktop and Android widths; English/French can switch at runtime; unsupported browser language falls back to English; no page relies on color alone.
- **Tests required:** Component tests for routing, language selection, persistence, status badges, keyboard navigation, and representative mobile layouts.
- **Security considerations:** Store only language preference locally; render server text safely; introduce no frontend secret variables.
- **Estimated difficulty:** Medium.

### FND-06 - Implement scheduler and collector boundaries [Complete]

- **Objective:** Run cancellable one-minute collection cycles through provider interfaces without overlapping runs.
- **Why it is needed:** All monitoring features require predictable scheduling, partial results, and testable collectors.
- **Files or components likely affected:** `internal/scheduler/`, `internal/collector/`, domain collection result models.
- **Dependencies:** `FND-01`, `FND-02`, `FND-03`.
- **Implementation outline:** Define host/Docker provider interfaces; add minute alignment, manual trigger, per-collector deadlines, overlap prevention, run records, cancellation, and partial-result aggregation.
- **Acceptance criteria:** One scheduled run occurs per minute; manual refresh cannot create unsafe overlap; one provider failure does not discard other results; missed periods are not backfilled with invented values.
- **Tests required:** Fake-clock tests for scheduling, overlap, cancellation, timeout, partial failure, shutdown, and manual refresh.
- **Security considerations:** Bound concurrency and duration to prevent resource exhaustion; do not execute shell commands from collector inputs.
- **Estimated difficulty:** Hard.

### FND-07 - Add operational logging and self-status [Complete]

- **Objective:** Emit safe structured logs and expose private liveness, readiness, and internal-status information.
- **Why it is needed:** Failures must be diagnosable without leaking monitored application data.
- **Files or components likely affected:** `internal/observability/`, API middleware, internal-status contracts/page.
- **Dependencies:** `FND-04`, `FND-06`.
- **Implementation outline:** Add request IDs, structured event codes, durations, safe resource identities, health endpoints, collector timing, database sizes, backup status placeholder, and Docker connectivity placeholder.
- **Acceptance criteria:** Logs reach stdout/stderr as structured records; health endpoints accurately distinguish alive from ready; sensitive values and container logs never appear.
- **Tests required:** Log field/redaction tests, request-ID propagation, liveness/readiness state tests, panic recovery, and cancellation tests.
- **Security considerations:** Keep internal status behind the same private boundary; never log headers, tokens, environment dumps, raw container logs, or exported content.
- **Estimated difficulty:** Medium.

## Phase 3 - Core features

### CORE-01 - Implement history queries and rolling retention [Complete]

- **Objective:** Store raw minute samples and return bounded raw or aggregated ranges while deleting expired history safely.
- **Why it is needed:** Every chart and the 14-day retention promise depend on common, efficient history behavior.
- **Files or components likely affected:** `internal/storage/history/`, `internal/scheduler/`, API range contracts.
- **Dependencies:** `FND-03`, `FND-06`.
- **Implementation outline:** Add collection-run transactions, range validation, min/average/max buckets, honest gaps, indexes, bounded cleanup batches, and 14-day cutoffs.
- **Acceptance criteria:** Raw minute samples remain available for exports; charts receive a bounded point count; gaps stay gaps; records older than 14 days are removed without long write locks.
- **Tests required:** Boundary-time, empty-range, gap, aggregation correctness, timezone-neutral storage, retention, and concurrent read/write tests.
- **Security considerations:** Reject excessive ranges and unbounded result sizes; use parameterized SQL exclusively.
- **Estimated difficulty:** Hard.

### CORE-02 - Implement CPU collection and API

- **Objective:** Collect and expose current/historical overall CPU, dynamic per-vCPU usage, and 1/5/15-minute load averages.
- **Why it is needed:** CPU monitoring is a primary MVP domain and dashboard input.
- **Files or components likely affected:** `internal/collector/host/cpu`, history repository, CPU API handlers/contracts.
- **Dependencies:** `CORE-01`.
- **Implementation outline:** Read counters safely; calculate deltas; discover logical CPUs every run; store overall/core samples; expose current and range endpoints.
- **Acceptance criteria:** Overall and every detected logical CPU are reported; CPU-count changes are handled; first-sample/unavailable states are honest; history uses one-minute samples.
- **Tests required:** Counter-delta fixtures, reset/wrap handling, CPU topology changes, zero elapsed time, load parsing, repository/API tests.
- **Security considerations:** Read only required `/proc` files; do not expose unrelated process or command-line data.
- **Estimated difficulty:** Medium.

### CORE-03 - Build the CPU page

- **Objective:** Present overall CPU, expandable per-vCPU details, load averages, thresholds, ranges, and historical charts.
- **Why it is needed:** The dedicated CPU page is an explicitly confirmed feature.
- **Files or components likely affected:** `web/src/features/cpu/`, CPU page, chart/status components, translations.
- **Dependencies:** `CORE-02`, `FND-05`.
- **Implementation outline:** Add summary, core list, threshold lines, all approved range controls, min/average/peak summaries, gaps, stale states, and accessible chart text.
- **Acceptance criteria:** Overall CPU appears first; all detected cores can be viewed; every approved range including custom works; stale/unavailable data is clear in English and French.
- **Tests required:** Component tests for range changes, dynamic core counts, gaps, threshold display, mobile layout, and translations.
- **Security considerations:** Treat API labels as data; avoid unsafe HTML in chart labels/tooltips.
- **Estimated difficulty:** Medium.

### CORE-04 - Implement RAM, swap, and PSI collection/API

- **Objective:** Collect and expose all approved memory fields, swap state, selected PSI behavior, period summaries, and container-ranking input.
- **Why it is needed:** RAM monitoring requires more than a single used percentage and must distinguish free from available memory.
- **Files or components likely affected:** `internal/collector/host/memory`, history repository, RAM API handlers/contracts.
- **Dependencies:** `SET-01`, `CORE-01`.
- **Implementation outline:** Parse Linux memory counters; calculate usage consistently; read swap and PSI when available; apply the approved missing-PSI rule; store raw samples and query summaries.
- **Acceptance criteria:** Total, used, available, free, cached, buffered, percentage, swap, and pressure are correct; missing PSI follows the locked rule; no value is fabricated.
- **Tests required:** `/proc/meminfo` and PSI fixtures, no-swap, missing-PSI, malformed/partial input, summary math, repository/API tests.
- **Security considerations:** Read aggregate kernel data only; do not expose per-process memory or environment information.
- **Estimated difficulty:** Medium.

### CORE-05 - Build the RAM page

- **Objective:** Present current/historical memory, swap, pressure, summaries, and thresholds, with a prepared section for later Docker RAM ranking integration.
- **Why it is needed:** Users need understandable memory evidence and navigation to responsible containers.
- **Files or components likely affected:** `web/src/features/memory/`, RAM page, chart/ranking components, translations.
- **Dependencies:** `CORE-04`, `FND-05`.
- **Implementation outline:** Build metric explanations, history/range controls, min/average/peak, warning line, swap/pressure states, and the empty/loading contract for the later container ranking.
- **Acceptance criteria:** All host-memory fields are shown without confusing free and available memory; missing data follows the approved Unknown/unavailable behavior; the ranking region handles unavailable Docker data; mobile layout remains usable.
- **Tests required:** Calculation display, no-swap, missing PSI, ranking unavailable/empty states, range controls, accessibility, and translations.
- **Security considerations:** Container names must be escaped and must not become unsafe URLs.
- **Estimated difficulty:** Medium.

### CORE-06 - Implement filesystem discovery, history, and API

- **Objective:** Discover every current mount and expose capacity, mount mode, history, lifecycle, and block I/O when measurable.
- **Why it is needed:** Monitoring all physical and virtual mounts is an approved requirement with unusual edge cases.
- **Files or components likely affected:** `internal/collector/host/filesystem`, block-device mapping, history repository, disk API.
- **Dependencies:** `CORE-01`.
- **Implementation outline:** Parse mounts; assign stable identities; collect statfs data; map devices to I/O counters where possible; track first/last seen and removed status; mark unsupported fields unavailable.
- **Acceptance criteria:** All mounts appear automatically; new/removed mounts are handled; virtual mounts do not fail collection; history remains for removed mounts until expiry.
- **Tests required:** Physical/virtual/bind/overlay/tmpfs fixtures, unusual path escaping, read-only mounts, device-mapping failures, mount lifecycle, repository/API tests.
- **Security considerations:** Do not traverse mount contents; expose paths/types only, never filenames or file contents.
- **Estimated difficulty:** Hard.

### CORE-07 - Build disk overview and filesystem pages

- **Objective:** Present all active mounts plus a dedicated detail page for each filesystem.
- **Why it is needed:** A root-only disk card would not satisfy the approved all-mount requirement.
- **Files or components likely affected:** `web/src/features/filesystems/`, disk overview/detail routes, translations.
- **Dependencies:** `CORE-06`, `FND-05`.
- **Implementation outline:** Add root summary, searchable mount list, capacity/status indicators, mount detail, history/range controls, I/O availability, and removed-mount labeling.
- **Acceptance criteria:** Every active mount is reachable; unsafe path characters do not break routing; unsupported I/O shows unavailable; threshold lines and period summaries work.
- **Tests required:** Large mount lists, virtual mounts, removed mounts, encoded route IDs, responsive tables/cards, charts, and translations.
- **Security considerations:** Do not create clickable local-file URLs or expose directory contents.
- **Estimated difficulty:** Medium.

### CORE-08 - Implement read-only Docker discovery and metrics

- **Objective:** Discover all containers and expose bounded current/history data for state, health, uptime, restart count, CPU, RAM, and published ports.
- **Why it is needed:** Docker visibility is the second primary MVP domain.
- **Files or components likely affected:** `internal/collector/docker/`, Docker repository/API, state-change storage.
- **Dependencies:** `CORE-01`, `FND-06`.
- **Implementation outline:** Connect through the official SDK; negotiate API version; list/inspect/stats existing containers; normalize states; track created/deleted containers and health/state changes; impose timeouts.
- **Acceptance criteria:** Running and stopped containers appear without registration; new containers are detected; deleted history remains marked; Docker-unavailable becomes a structured failure; no control operation is exposed yet.
- **Tests required:** Docker fixture containers for all states/health modes, stats math, ports, deletion/recreation, daemon unavailable/timeout, repository/API tests.
- **Security considerations:** Direct Docker access is root-equivalent; expose only normalized required fields and never Docker config secrets, environment arrays, mounts, labels, or arbitrary inspect responses.
- **Estimated difficulty:** Hard.

### CORE-09 - Build Docker overview and container details

- **Objective:** Present container counts, current states, resource history, ports, and per-container navigation without controls.
- **Why it is needed:** This creates safe read-only Docker visibility before privileged actions are added.
- **Files or components likely affected:** `web/src/features/docker/`, Docker overview/detail pages, charts, translations.
- **Dependencies:** `CORE-08`, `FND-05`.
- **Implementation outline:** Add state filters/counts, container cards/table, detail fields, CPU/RAM charts, health/state timeline, deleted state, loading/stale/error handling, and the linked top-containers-by-RAM ranking on the RAM page.
- **Acceptance criteria:** All containers and confirmed fields display; the RAM page ranking links to container details; stopped/no-healthcheck cases are clear; deleted entries leave active lists; all range controls work on desktop and Android layouts.
- **Tests required:** State/health variants, duplicate names/IDs, empty Docker host, deletion, charts, navigation, accessibility, and translations.
- **Security considerations:** Escape names/logical identifiers; do not expose raw Docker objects or secret-bearing configuration.
- **Estimated difficulty:** Medium.

### CORE-10 - Implement health evaluation and event history

- **Objective:** Evaluate one-sample Warning/Critical rules and retain open/closed incidents with simultaneous causes.
- **Why it is needed:** The dashboard must explain health, not merely display raw numbers.
- **Files or components likely affected:** `internal/health/`, incident history repository, event API.
- **Dependencies:** `CORE-02`, `CORE-04`, `CORE-06`, `CORE-08`.
- **Implementation outline:** Implement pure threshold/state transitions, resource identities, open/update/close logic after committed samples, Unknown/stale handling, and 14-day event queries.
- **Acceptance criteria:** Confirmed thresholds trigger and clear after one qualifying sample; simultaneous Warning/Critical/Unknown causes coexist; only Docker unhealthy creates container-health warning; event start/end/cause/resource are retained.
- **Tests required:** Full state-transition matrix, simultaneous conditions, flapping, unavailable/stale data, removed resources, and transaction-order tests.
- **Security considerations:** Event causes contain bounded normalized data, not raw Docker or exception payloads.
- **Estimated difficulty:** Hard.

### CORE-11 - Build the overview and event pages

- **Objective:** Deliver the first usable read-only preview: one overview of host/Docker health plus central and contextual event history.
- **Why it is needed:** This solves the core problem of moving between tools to understand current VPS health.
- **Files or components likely affected:** Dashboard page, event page, shared health/summary components, CPU/RAM/filesystem/container detail integrations.
- **Dependencies:** `CORE-03`, `CORE-05`, `CORE-07`, `CORE-09`, `CORE-10`.
- **Implementation outline:** Add all confirmed summary values, concurrent badges/causes, warning count, latest successful collection time, manual refresh, event filters, and contextual event sections.
- **Acceptance criteria:** The overview contains every FR-3 item; manual and minute refresh work; simultaneous health states remain visible; event pages retain and link evidence; English/French and Android layouts work.
- **Tests required:** Dashboard aggregation, multi-status, stale/current timestamps, manual refresh, event filters/links, responsive and localization tests.
- **Security considerations:** Manual refresh is rate-limited later; event text must be safely rendered and must not contain secrets.
- **Estimated difficulty:** Medium.

### CORE-12 - Stream bounded Docker logs from the backend

- **Objective:** Provide the latest 100 lines and cancellable stdout/stderr SSE without storing container logs.
- **Why it is needed:** Live logs are approved, but duplicating them would violate privacy and storage requirements.
- **Files or components likely affected:** Docker SDK layer, SSE API, stream middleware and tests.
- **Dependencies:** `CORE-08`, `FND-04`.
- **Implementation outline:** Validate container IDs; request recent Docker logs with timestamps/stream separation; frame SSE safely; cancel on disconnect; bound line/frame sizes, time, and concurrent streams.
- **Acceptance criteria:** Viewer startup returns up to 100 recent lines then follows new output; stdout/stderr remain distinguishable; disconnect stops Docker work; no log is written to SQLite or operational logs.
- **Tests required:** SSE framing, multiline/binary-like content, large lines, reconnect, disconnect, backpressure, unavailable logging driver, and timeout tests.
- **Security considerations:** Raw logs may contain secrets; never persist, index, audit, or operationally log their contents; enforce private access and concurrency bounds.
- **Estimated difficulty:** Hard.

### CORE-13 - Build the bounded live-log viewer

- **Objective:** Present searchable live logs with all approved controls and a 5 MiB browser-memory ceiling.
- **Why it is needed:** A raw unbounded stream would become unusable and could exhaust Android browser memory.
- **Files or components likely affected:** `web/src/features/logs/`, Web Worker/search utilities, translations.
- **Dependencies:** `CORE-12`, `FND-05`.
- **Implementation outline:** Add ring buffer, dropped-content count, plain/case-sensitive/regex modes, safe cancellable regex work, highlighting, stdout/stderr filters, pause/follow/clear/copy, and Africa/Tunis timestamps.
- **Acceptance criteria:** Initial and new lines are searchable; invalid regex is safe; buffer never exceeds 5 MiB; oldest content is discarded visibly; clearing affects only the browser; follow behavior matches scrolling.
- **Tests required:** Buffer boundary, noisy stream, Unicode, regex cancellation, invalid regex, filters, pause/resume, scroll-follow, reconnect, copy, and Android memory behavior.
- **Security considerations:** Render logs as text only; do not place log content in URLs, analytics, error reports, persistent browser storage, or clipboard without an explicit copy action.
- **Estimated difficulty:** Hard.

### CORE-14 - Implement the administrative audit repository and view

- **Objective:** Persist and display bounded 14-day administrative action records independently from monitoring history.
- **Why it is needed:** Configuration reloads, exports, backups, restores, Docker actions, and deletion must be accountable by source IP and result.
- **Files or components likely affected:** `internal/audit/`, `audit.db` repository/API, audit page and translations.
- **Dependencies:** `FND-03`, `FND-02`, `FND-05`.
- **Implementation outline:** Define event writer interface; connect config results; add paged/date queries and UI; retain normalized action/target/result/request IP; add rolling retention. Deletion is deferred to `SEC-03`.
- **Acceptance criteria:** Success/failure records are independently stored and viewable; config reloads are captured; audit data is excluded from history backups; records expire after 14 days.
- **Tests required:** Insert/query/paging/date range/retention, unavailable audit DB, safe field bounds, UI empty/error/paging, and translation tests.
- **Security considerations:** Never store secrets, CSRF tokens, raw logs, exported data, or unrestricted Docker errors; derive the requesting IP from the direct TCP peer and ignore forwarded-client headers.
- **Estimated difficulty:** Medium.

### CORE-15 - Implement streaming CSV and JSON exports

- **Objective:** Export every approved dataset and time range without loading the entire result into memory.
- **Why it is needed:** Users need portable evidence while the monitor remains within resource limits.
- **Files or components likely affected:** `internal/export/`, export API/page, audit integration.
- **Dependencies:** `CORE-01`, `CORE-10`, `CORE-14`.
- **Implementation outline:** Validate datasets/ranges; stream raw rows; define stable columns/schema; add preset/custom UI; cancel on disconnect; limit concurrency/duration; audit metadata and result.
- **Acceptance criteria:** CSV and JSON cover all confirmed datasets, including audit records but excluding container logs; exports preserve minute samples and valid gaps; one client cannot run unbounded concurrent exports.
- **Tests required:** Format/schema, escaping/Unicode, empty/custom ranges, 14-day boundary, cancellation, large streaming dataset, audit result, and memory tests.
- **Security considerations:** Treat exports as sensitive downloads; set safe content headers; prevent formula injection in CSV; never cache publicly or log exported content.
- **Estimated difficulty:** Hard.

### CORE-16 - Implement scheduled and manual backup creation

- **Objective:** Create verified history/settings backups every two days and on confirmed manual request.
- **Why it is needed:** Local logical recovery is an approved MVP requirement.
- **Files or components likely affected:** `internal/backup/`, scheduler, backup API/page, manifest format, audit integration.
- **Dependencies:** `FND-02`, `FND-03`, `CORE-14`.
- **Implementation outline:** Use SQLite online backup; copy active/last-valid settings; exclude audit/logs; checksum and integrity-check temporary output; atomically publish; list manifests independently of history DB.
- **Acceptance criteria:** Scheduled/manual archives are consistent and verifiable; incomplete archives never appear valid; audit records describe result; backups contain no `audit.db` or logs.
- **Tests required:** Online backup under writes, checksum/integrity failure, interrupted creation, manifest compatibility, scheduler timing, exclusions, and audit tests.
- **Security considerations:** Restrict archive ownership/modes; never include credentials or signing material; avoid following symlinks outside approved paths.
- **Estimated difficulty:** Hard.

### CORE-17 - Implement restore and corruption recovery

- **Objective:** Restore a selected verified backup only after confirmation and a successful safety copy, including when history storage is corrupt.
- **Why it is needed:** A backup without a rehearsed safe restore path does not satisfy recovery requirements.
- **Files or components likely affected:** `internal/backup/restore`, maintenance/recovery API, recovery page, audit integration, corruption fixtures.
- **Dependencies:** `CORE-16`, `FND-04`.
- **Implementation outline:** Detect defined corruption/unopenable states; enter maintenance mode; list manifests without history DB; create safety copy; validate archive/schema/checksum; atomically replace; smoke-check; automatically return to pre-restore files on failure.
- **Acceptance criteria:** Restoration is never automatic; no restore proceeds without safety copy; corrupted history still permits recovery UI; failed restore preserves/returns to prior files; success/failure is audited when audit DB is available.
- **Tests required:** Corrupt/truncated DB, invalid checksum, disk-full safety copy, incompatible schema, interrupted atomic swap, rollback, successful restore, and UI confirmation tests.
- **Security considerations:** Prevent path traversal and arbitrary archive selection; accept only trusted local manifests; keep backup/safety files outside web-accessible paths.
- **Estimated difficulty:** Hard.

### CORE-18 - Enforce the combined 5 GB storage ceiling

- **Objective:** Keep monitoring databases and retained backup/safety data within the approved 5 GB budget without silently deleting unexpired evidence.
- **Why it is needed:** The VPS has limited free disk and no swap; storage exhaustion could damage both the monitor and hosted workloads.
- **Files or components likely affected:** retention/backup/storage-budget services, internal status, dashboard warning.
- **Dependencies:** `CORE-01`, `CORE-14`, `CORE-16`, `CORE-17`.
- **Implementation outline:** Measure included paths safely; expire data by policy; preflight backups/exports/restores; reject work that cannot fit; expose Critical storage state and required user action.
- **Acceptance criteria:** Included data never intentionally exceeds 5 GB; 14-day cleanup runs in bounded batches; new work is refused clearly rather than deleting unexpired data; symlinks cannot escape counted roots.
- **Tests required:** Boundary sizes, low disk, concurrent backup/cleanup, safety copies, symlink fixtures, refusal behavior, and recovery after space is freed.
- **Security considerations:** Avoid recursive traversal outside explicit directories; resist symlink and integer-overflow attacks; never delete paths based on unvalidated names.
- **Estimated difficulty:** Hard.

## Phase 4 - Security and error handling

### SEC-01 - Protect state-changing and expensive API requests

- **Objective:** Enforce same-origin requests, strict Host/Origin checks, CSRF tokens, single-use confirmation intents, rate limits, and request/stream bounds.
- **Why it is needed:** Tailnet access replaces login but does not prevent browser request forgery, accidents, or abusive approved devices.
- **Files or components likely affected:** API middleware, confirmation-intent service, frontend API client/dialog flow, security configuration.
- **Dependencies:** `FND-04`, `FND-05`, `CORE-14`.
- **Implementation outline:** Define trusted host/origin; reject permissive CORS; issue short-lived action-bound intents; require JSON/custom header; rate-limit refresh/export/log/recovery/control paths; add safe security headers.
- **Acceptance criteria:** Cross-origin and replayed actions fail; intents cannot change target/action; legitimate same-origin flows work; limits return stable bilingual errors; audit attribution uses only the direct TCP peer.
- **Tests required:** Host/Origin/CSRF matrix, token replay/expiry/substitution, malformed content types, rate-limit, body/stream/time limits, and proof that forwarded-client headers are ignored.
- **Security considerations:** Store signing material outside Git with restrictive modes; compare tokens safely; do not treat confirmation as identity authentication.
- **Estimated difficulty:** Hard.

### SEC-02 - Add confirmed Docker start, stop, and restart

- **Objective:** Expose only the three approved Docker actions with confirmation, understandable results, and mandatory auditing.
- **Why it is needed:** Docker administration is required, but direct socket access creates the project's most dangerous attack surface.
- **Files or components likely affected:** Docker SDK action service/API, container detail page, confirmation dialogs, audit integration.
- **Dependencies:** `CORE-08`, `CORE-09`, `CORE-14`, `SEC-01`.
- **Implementation outline:** Validate immutable container IDs; map only start/stop/restart; require action-bound intent; apply deadlines; refresh state; show bounded friendly/raw errors; reject action if audit DB is unavailable; audit requested result.
- **Acceptance criteria:** Each action requires a fresh confirmation; create/delete/exec/arbitrary operations have no endpoint; successes and failures are shown and audited; audit failure prevents action execution.
- **Tests required:** Allowed-action matrix, ID validation, token binding/replay, Docker timeout/error, audit-unavailable fail-closed behavior, concurrent requests, and UI confirmations.
- **Security considerations:** Docker access is root-equivalent even with a small API; never accept an Engine path, CLI string, or arbitrary operation from the browser.
- **Estimated difficulty:** Hard.

### SEC-03 - Add confirmed audit-record deletion

- **Objective:** Delete selected, date-range, or all existing audit entries while always leaving a new deletion record.
- **Why it is needed:** The approved privacy workflow permits deletion but requires evidence that deletion occurred.
- **Files or components likely affected:** Audit repository/API/page, confirmation intents and dialogs.
- **Dependencies:** `CORE-14`, `SEC-01`.
- **Implementation outline:** Validate scope; preview count; require bound confirmation; transactionally delete and insert a new deletion record containing time, IP, scope, and count.
- **Acceptance criteria:** All three scopes work; cancellation changes nothing; deleting all leaves exactly the new deletion record; failures do not report false success.
- **Tests required:** Selected/range/all/empty scopes, concurrent inserts, rollback, replay, count accuracy, and UI confirmation tests.
- **Security considerations:** Require explicit intent and parameterized SQL; bound selection sizes; do not allow arbitrary query predicates.
- **Estimated difficulty:** Medium.

### SEC-04 - Complete stale, partial-failure, and degraded-mode behavior

- **Objective:** Make every collector/storage/Docker/settings failure honest and keep safe portions of the app usable.
- **Why it is needed:** A monitoring tool is dangerous if it silently shows old or invented values during failure.
- **Files or components likely affected:** collectors, health evaluator, API errors, dashboard/detail/recovery views, translations.
- **Dependencies:** `CORE-11`, `CORE-17`, `CORE-18`, `SEC-02`.
- **Implementation outline:** Apply two-minute staleness; retain timestamped last-known values; preserve chart gaps; support partial runs; show invalid-config fallback; disable unsafe actions when audit/history requirements fail; localize stable error codes.
- **Acceptance criteria:** Every specified expected failure has an explicit UI state; healthy subsystems continue; no missing point is interpolated; corruption routes to recovery; invalid configuration visibly uses the old settings.
- **Tests required:** Failure injection for every collector, Docker, SQLite busy/corrupt, audit unavailable, settings invalid, disk ceiling, SSE disconnect, and recovery states.
- **Security considerations:** Error details must aid the owner without revealing stack traces, secrets, environment data, or raw Docker configuration.
- **Estimated difficulty:** Hard.

### SEC-05 - Add Linux service and filesystem hardening

- **Objective:** Define a dedicated service identity, exact file modes, systemd sandboxing, runtime directories, and direct Docker socket access.
- **Why it is needed:** Production must limit ordinary filesystem/process exposure even though Docker access remains root-equivalent.
- **Files or components likely affected:** `deploy/systemd/`, install scripts or instructions, state/config/backup path handling.
- **Dependencies:** `FND-04`, `CORE-18`, `SEC-02`.
- **Implementation outline:** Define service user/group; provision `/etc/pim`, `/var/lib/pim`, `/var/backups/pim`, `/run/pim`; grant Docker group access; apply compatible systemd hardening; configure restart/startup and journald.
- **Acceptance criteria:** Service starts after boot and restarts after crash; only service user and sudo administrators access state; Go HTTP socket is private; hardening does not break required Docker/storage operations.
- **Tests required:** `systemd-analyze security`, permission matrix, boot/crash restart, denied-path checks, Docker access, state persistence, and journal output.
- **Security considerations:** Document clearly that Docker group membership defeats a complete privilege boundary; do not grant additional Linux capabilities unnecessarily.
- **Estimated difficulty:** Hard.

### SEC-06 - Enforce tailnet-only HTTP exposure

- **Objective:** Ensure that only approved Headscale/Tailscale devices can reach the Go HTTP listener.
- **Why it is needed:** Network-only authorization is the product's primary access-control boundary.
- **Files or components likely affected:** Go listener configuration, `deploy/firewall/`, Headscale policy instructions, systemd configuration.
- **Dependencies:** `SET-01`, `SEC-01`, `SEC-05`.
- **Implementation outline:** Discover and bind only the `tailscale0` IPv4 address; fail closed if unavailable; allow the configured port only on the Tailscale interface and for approved Headscale sources; block public/unrelated interfaces.
- **Acceptance criteria:** Supported tailnet browsers reach the HTTP dashboard; the accepted browser "Not secure" indicator is documented; public IP/unrelated interfaces cannot connect; unapproved tailnet devices are denied by policy.
- **Tests required:** Positive Chrome/Firefox/Opera/Android access, negative public/non-tailnet/unauthorized-device tests, listener-address audit, firewall/policy checks, and forwarded-header spoof tests.
- **Security considerations:** HTTP is safe only within the encrypted tailnet; preserve recovery access before firewall changes and stop deployment immediately if any public listener is detected.
- **Estimated difficulty:** Hard.

## Phase 5 - Testing

### TST-01 - Complete backend unit-test coverage

- **Objective:** Close unit-test gaps across calculations, transitions, configuration, retention, backup, restore, export, and validation.
- **Why it is needed:** Core logic must be trustworthy without requiring a live VPS for every change.
- **Files or components likely affected:** Go `_test.go` files and deterministic fixtures.
- **Dependencies:** `SEC-04`.
- **Implementation outline:** Build a requirement-to-test matrix; add boundary/property/table tests; use fake clocks/providers; run race detector on concurrency-sensitive packages.
- **Acceptance criteria:** Every pure core rule has success, boundary, and failure coverage; tests are deterministic; race detector passes.
- **Tests required:** This task is the test suite itself plus repeated clean runs and coverage review; do not use coverage percentage as the only quality measure.
- **Security considerations:** Fixtures must contain synthetic data only; add negative tests for unsafe IDs, paths, SQL inputs, headers, and log content.
- **Estimated difficulty:** Medium.

### TST-02 - Complete Linux, SQLite, and Docker integration tests

- **Objective:** Verify real component boundaries in a disposable Linux environment.
- **Why it is needed:** `/proc`, mounts, WAL, systemd, and Docker behavior cannot be proven fully with mocks.
- **Files or components likely affected:** `tests/integration/`, Linux test harness, harmless fixture containers/databases.
- **Dependencies:** `TST-01`, `SEC-05`.
- **Implementation outline:** Provision disposable fixtures; test migrations/WAL/retention/corruption; Docker states/stats/logs/actions; tailnet-interface binding; permissions; shutdown and recovery.
- **Acceptance criteria:** Integration suite is repeatable and never touches production; all approved container states and storage failures are represented; cleanup leaves no fixture state.
- **Tests required:** Run full integration suite from a clean disposable environment at least twice, including race/cancellation-sensitive scenarios.
- **Security considerations:** Never mount host root or production Docker socket into tests; fixture actions must be confined to clearly named disposable resources.
- **Estimated difficulty:** Hard.

### TST-03 - Complete frontend, localization, and accessibility tests

- **Objective:** Verify every important UI state and complete English/French coverage.
- **Why it is needed:** A bilingual monitoring UI must remain understandable during errors, not only during healthy demos.
- **Files or components likely affected:** frontend tests, i18n completeness checker, accessibility fixtures.
- **Dependencies:** `SEC-04`, `SEC-03`.
- **Implementation outline:** Cover loading/empty/stale/Unknown/Warning/Critical, dialogs, charts/gaps, log controls, recovery, language switching, keyboard/touch use, icon+text+color statuses.
- **Acceptance criteria:** No missing or fallback translation key ships; core workflows are keyboard accessible; Android layouts avoid clipped controls and unreadable tables/charts.
- **Tests required:** Vitest/RTL suite, automated accessibility checks, translation-key parity, representative viewport snapshots where useful.
- **Security considerations:** Assert untrusted names/logs/errors render as text and cannot inject HTML or unsafe links.
- **Estimated difficulty:** Medium.

### TST-04 - Run end-to-end browser acceptance

- **Objective:** Exercise complete owner workflows through the real HTTP/API/storage stack.
- **Why it is needed:** Component tests do not prove that collection, history, controls, audit, backup, and recovery work together.
- **Files or components likely affected:** `tests/e2e/`, browser fixtures, acceptance checklist.
- **Dependencies:** `TST-02`, `TST-03`, `SEC-06`.
- **Implementation outline:** Automate primary workflows in Chromium/Firefox; test Android Chrome/Firefox layouts; smoke Opera GX; include positive and denial paths; rehearse backup/restore.
- **Acceptance criteria:** Every applicable `PROJECT_CONTEXT.md` acceptance criterion has passing evidence or a documented blocker; supported browsers complete the critical paths.
- **Tests required:** Full Playwright suite, real Opera GX smoke test, Android device/emulator checks, tailnet access/denial checks, recovery rehearsal.
- **Security considerations:** Use disposable accounts/data and non-production fixture containers; do not capture secret-bearing logs in screenshots/artifacts.
- **Estimated difficulty:** Hard.

### TST-05 - Verify performance, storage, and resilience budgets

- **Objective:** Prove the monitor remains lightweight and bounded with 14 days of representative/stress data.
- **Why it is needed:** Monitoring must not become a significant workload on the 11 GiB/no-swap VPS.
- **Files or components likely affected:** performance harness, generated fixtures, benchmark report.
- **Dependencies:** `TST-02`, `TST-04`.
- **Implementation outline:** Generate realistic/stress mount/container samples; measure collection, chart queries, export, log buffer, cleanup, backup, restore; exercise disk-full and restart behavior.
- **Acceptance criteria:** Normal average CPU is below 5%, memory below 512 MiB, data/backups below 5 GB; browser log buffer stays at 5 MiB; permitted spikes are measured and documented.
- **Tests required:** Repeatable load/soak tests, noisy-log test, 14-day query benchmarks, low-disk, crash/restart, concurrent reads/writes, and cleanup timing.
- **Security considerations:** Use synthetic logs/metrics; resource tests must have hard time, memory, disk, and process limits to avoid harming the host.
- **Estimated difficulty:** Hard.

## Phase 6 - Documentation

### DOC-01 - Write developer setup and contribution workflow

- **Objective:** Make a fresh development environment reproducible for the owner.
- **Why it is needed:** Future maintenance should not depend on remembering hidden setup steps.
- **Files or components likely affected:** `README.md`, development/setup documentation.
- **Dependencies:** `TST-01`, `TST-02`.
- **Implementation outline:** Document prerequisites, repository structure, safe local modes, commands, test layers, formatting, and one-task workflow.
- **Acceptance criteria:** A clean machine can build and run safe development mode from the documentation; production Docker access is clearly separated.
- **Tests required:** Follow the guide from a clean checkout/environment and record corrections.
- **Security considerations:** Use placeholders for domains/tokens; warn against production socket use and secret commits.
- **Estimated difficulty:** Easy.

### DOC-02 - Write configuration and operations guide

- **Objective:** Document settings, service lifecycle, normal status, storage, exports, backups, and routine troubleshooting.
- **Why it is needed:** The dashboard should save time rather than create undocumented operational work.
- **Files or components likely affected:** `docs/operations.md`, configuration reference, example TOML.
- **Dependencies:** `TST-05`, `SEC-05`.
- **Implementation outline:** Explain every setting/default/bound; startup/restart; journald; data locations; manual refresh/export/backup; invalid-config fallback; capacity behavior.
- **Acceptance criteria:** Every administrator-editable setting is documented in English; examples validate; common failures link to safe checks.
- **Tests required:** Validate example configuration; follow operational procedures in staging.
- **Security considerations:** Never include real IPs, domain credentials, private keys, container logs, or secrets in examples.
- **Estimated difficulty:** Easy.

### DOC-03 - Write security, network, and recovery runbooks

- **Objective:** Document the real trust boundaries, accepted risks, access verification, incident response, backup restore, and rollback.
- **Why it is needed:** Direct Docker access and no application login demand unusually clear operational guidance.
- **Files or components likely affected:** `docs/security-boundaries.md`, `docs/recovery.md`, deployment documentation.
- **Dependencies:** `SEC-06`, `TST-04`, `TST-05`.
- **Implementation outline:** Explain tailnet authorization, DNS token handling, Docker root-equivalent risk, permissions, exposure checks, corruption recovery, safety copies, application rollback, and same-VPS backup limitation.
- **Acceptance criteria:** Runbooks match rehearsed commands and outcomes; public-exposure verification is explicit; destructive steps include exact targets and validation; limitations are honest.
- **Tests required:** Perform a tabletop walkthrough and one staging recovery/rollback using only the runbook.
- **Security considerations:** Redact environment-specific secrets and avoid publishing exploitable infrastructure details if documentation becomes public.
- **Estimated difficulty:** Medium.

### DOC-04 - Complete requirement traceability and release notes

- **Objective:** Map every MVP requirement and acceptance criterion to implementation and test evidence, then prepare the first changelog entry.
- **Why it is needed:** The project is broad enough that features could otherwise be declared complete without proof.
- **Files or components likely affected:** `docs/acceptance-matrix.md`, `CHANGELOG.md`, `TASKS.md` statuses.
- **Dependencies:** `DOC-01`, `DOC-02`, `DOC-03`, `TST-04`.
- **Implementation outline:** Build requirement/task/test links; mark passed/blocked; document accepted limitations and postponed features; write concise release notes.
- **Acceptance criteria:** Every FR/NFR/acceptance item has evidence or an explicit approved exception; no postponed feature is described as implemented.
- **Tests required:** Manual cross-check against both approved source documents and test reports.
- **Security considerations:** Evidence must not embed secrets, raw production logs, private DNS tokens, or sensitive screenshots.
- **Estimated difficulty:** Medium.

## Phase 7 - Release

### REL-01 - Produce a reproducible release artifact

- **Objective:** Build one versioned Linux amd64 Go executable containing the production frontend, plus checksum and manifest.
- **Why it is needed:** Production should not require Node.js, Python, Cloudflare, or development servers.
- **Files or components likely affected:** release/build configuration, embedded assets, version metadata, artifact checksums.
- **Dependencies:** `DOC-04`, all passing test tasks.
- **Implementation outline:** Clean-install dependencies; type-check/test/build frontend; embed assets; compile pinned Go dependencies; run artifact smoke tests; generate version/checksum manifest.
- **Acceptance criteria:** Artifact starts in a clean supported Linux environment; serves matching UI/API versions; checksum verifies; build inputs are locked and documented.
- **Tests required:** Clean reproducible build, checksum verification, embedded-asset/API smoke tests, dependency/license/vulnerability review.
- **Security considerations:** Build from reviewed locked dependencies; exclude source secrets and development files; protect release provenance/checksums.
- **Estimated difficulty:** Medium.

### REL-02 - Rehearse installation, upgrade, restore, and rollback

- **Objective:** Prove the entire release lifecycle on a disposable Linux staging host before touching the VPS.
- **Why it is needed:** Schema, systemd, listener, firewall, and recovery mistakes are safer to discover away from production.
- **Files or components likely affected:** deployment scripts/instructions, release checklist, rehearsal report.
- **Dependencies:** `REL-01`, `SEC-06`.
- **Implementation outline:** Install into versioned paths; start services; upgrade with pre-deploy backup; verify health; simulate failure; roll back binary/schema; restore data; verify tailnet-only access.
- **Acceptance criteria:** Fresh install, one upgrade, one crash recovery, one backup restore, and one rollback succeed from documentation; expected data-loss boundaries are recorded.
- **Tests required:** Full staging rehearsal, service/permission checks, listener-address audit, public-exposure negative test, and acceptance smoke suite.
- **Security considerations:** Use staging-only policies; verify firewall targets before applying and preserve recovery access.
- **Estimated difficulty:** Hard.

### REL-03 - Deploy and accept the first MVP release

- **Objective:** Install the verified release on the monitored VPS and collect final acceptance evidence.
- **Why it is needed:** The project is complete only when the intended owner can safely use it on the real tailnet.
- **Files or components likely affected:** VPS release paths/configuration, systemd/firewall/Headscale policy, release record; repository changes only for discovered documentation fixes.
- **Dependencies:** `REL-02` and explicit user authorization for production deployment.
- **Implementation outline:** Preflight disk/resources/backups; install versioned artifact; provision protected credentials/settings; apply reviewed network policy; start services; run acceptance checks; monitor first collection/backup cycle; retain rollback release.
- **Acceptance criteria:** All approved acceptance criteria pass on the VPS; tailnet browsers work without warnings; public access fails; resource/storage budgets hold; rollback remains ready; user approves the release.
- **Tests required:** Production-safe acceptance suite, supported-browser checks, listener/firewall audit, Docker read/control confirmation, collection/history, backup verification, restart persistence, and post-deploy health review.
- **Security considerations:** This task changes production and requires a separate explicit start; never expose secrets in output; preserve SSH/tailnet recovery access; stop and roll back on unexpected public exposure or data-integrity failure.
- **Estimated difficulty:** Hard.

## Next task

SET-01 through SET-04, FND-01 through FND-07, and CORE-01 are complete. The next and only task eligible to start is **CORE-02 - Implement CPU collection and API**.

Wait for the exact authorization: `START TASK CORE-02`.
