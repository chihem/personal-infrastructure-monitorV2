# Infrastructure Monitor — Project Context

Status: **Approved by user**
Approved: 2026-08-03
Last updated: 2026-08-04
Architecture: **Approved for task-by-task implementation**
Implementation: **Only explicitly started tasks are authorized**

## Project overview

Infrastructure Monitor is a private, self-hosted web dashboard for monitoring and administering one Linux VPS. The application runs on the same VPS that it monitors and is reachable only through the user's Headscale-managed Tailscale network. HTTP inside the encrypted tailnet is explicitly accepted; the dashboard is not a public website.

The MVP focuses on host resource monitoring and Docker. It collects measurements once per minute, retains history for 14 days, presents responsive English/French dashboards, and allows confirmed Docker start, stop, and restart operations.

The selected backend language is Go and the selected frontend language is TypeScript. Frameworks, libraries, database technology, deployment topology, and other architecture choices are intentionally undecided until this document is approved.

### Scope assessment

The monitored domain has been reduced to one VPS and two primary feature areas, but the requested feature depth makes this a **medium-to-large MVP**, not a small dashboard. The largest scope contributors are:

- Detailed CPU, RAM, filesystem, and per-container history.
- Every mount, including virtual filesystems.
- Docker administration with confirmation and auditing.
- Searchable live container logs.
- Backup and in-dashboard recovery workflows.
- CSV and JSON exports.
- English/French localization.
- Responsive behavior and three-browser testing.

No confirmed feature has been removed. If a smaller first delivery becomes necessary, scope reduction must be approved separately.

## Problem and goals

### Problem

The user currently has to move between different VPS and Docker tools to understand infrastructure health. This costs time and makes it harder to see important conditions in one place.

### Primary goal

Provide one private web interface that shows the current and recent health of a single VPS and its Docker containers.

### Supporting goals

- Make current resource pressure and capacity easy to understand.
- Preserve 14 days of measurements, state changes, warnings, and critical events.
- Make Docker container state, health, resource consumption, ports, and logs visible.
- Permit deliberate Docker start, stop, and restart actions.
- Make failures and stale data explicit instead of presenting misleading values.
- Support recovery through local backups.
- Remain lightweight enough not to materially interfere with the monitored VPS.

## Target users

### MVP user

- The VPS owner only.

### Possible later users

- A small number of trusted security analysts.

Multi-user accounts, user roles, permissions, and attribution by person are not MVP requirements.

## User stories

1. As the VPS owner, I want to see the VPS's overall health and active problems from one screen.
2. As the VPS owner, I want to inspect current and historical overall CPU usage.
3. As the VPS owner, I want to inspect each logical vCPU and Linux load averages on a dedicated CPU page.
4. As the VPS owner, I want to understand total, used, available, free, cached, and buffered memory without confusing free memory with usable memory.
5. As the VPS owner, I want to see memory pressure, swap state, historical statistics, and the containers using the most RAM.
6. As the VPS owner, I want to see every mounted filesystem, including virtual mounts, and open a detail page for each one.
7. As the VPS owner, I want to identify filesystems nearing full capacity and inspect their history and I/O activity when measurable.
8. As the VPS owner, I want all existing and newly created Docker containers to appear automatically.
9. As the VPS owner, I want a detail page for each container showing its state, health, uptime, restarts, resources, ports, and history.
10. As the VPS owner, I want to search and follow recent Docker logs without the monitor storing another copy.
11. As the VPS owner, I want to start, stop, or restart a container after confirming the action.
12. As the VPS owner, I want failed Docker actions to be understandable while retaining optional technical detail.
13. As the VPS owner, I want missing measurements to be visibly stale and historical gaps to remain honest.
14. As the VPS owner, I want cleared warnings and critical conditions retained as events for 14 days.
15. As the VPS owner, I want to export selected retained data as CSV or JSON.
16. As the VPS owner, I want automatic and manual backups and a confirmed recovery workflow.
17. As the VPS owner, I want to switch the interface between English and French at any time.

## Functional requirements

### FR-1: Deployment and access

- Monitor exactly one VPS.
- Run on the same VPS being monitored.
- Be reachable only from devices connected to the Headscale-managed Tailscale network.
- Not be reachable through the VPS public IP or unrelated network interfaces.
- Serve the dashboard over HTTP on a configurable non-privileged port bound only to the VPS Tailscale interface/address.
- Accept that browsers may label the HTTP page as "Not secure" even though transport between tailnet devices is encrypted by Tailscale.
- Do not require an application login, user account, password, or control PIN.
- Treat a device with permitted tailnet access as authorized.

### FR-2: Collection, freshness, and retention

- Collect host and Docker measurements once per minute.
- Refresh an open dashboard automatically once per minute.
- Provide a manual refresh control.
- Retain monitoring history for 14 days using rolling cleanup.
- Mark data stale after two minutes without a successful update.
- When data is stale, show the last known value, collection timestamp, stale label, and a prominent collection-failure warning.
- Display missing collection periods as gaps; do not interpolate or estimate values.
- Retain container state and health changes for 14 days.

### FR-3: Main dashboard

The main dashboard must show:

- All currently applicable health badges rather than collapsing simultaneous states into one badge.
- A Healthy badge when no problem is active.
- Active warning count.
- Current overall CPU percentage.
- Current RAM usage.
- Root filesystem usage.
- Counts of running, stopped, and unhealthy Docker containers.
- Time of the most recent successful collection.
- Navigation to CPU, RAM, disk, Docker, and event details.

Health states are Healthy, Warning, Critical, and Unknown. Simultaneous Warning, Critical, and Unknown badges may be visible, with each badge exposing its causes.

### FR-4: Thresholds and health conditions

- Store warning thresholds in a settings file.
- Initially configure CPU warning at 85%, RAM warning at 85%, and filesystem warning at 90%.
- Trigger those warnings after one qualifying one-minute measurement.
- Mark overall health Critical when any of the following is detected:
  - CPU is at least 95%.
  - RAM is at least 95%.
  - Any filesystem is at least 95%.
  - Any Docker container is unhealthy.
  - The history database cannot be accessed.
  - The monitor cannot communicate with Docker.
- Trigger CPU and RAM Critical states after one qualifying one-minute measurement.
- Clear warning and Critical states after the next normal measurement.
- Mark unavailable current measurements Unknown or stale as appropriate while retaining last-known evidence.

### FR-5: Settings reload

- Detect settings-file changes automatically.
- Apply valid changes without restarting the application.
- Reject invalid configuration.
- Continue using the last valid configuration after an invalid edit.
- Show a dashboard warning explaining that the file is invalid and the previous configuration is active.
- Clear that warning after a valid configuration loads successfully.
- Record configuration reload successes and failures in the administrative audit log.

### FR-6: CPU monitoring

- Show compact overall CPU usage on the main dashboard.
- Link the CPU summary to a dedicated CPU page.
- On the CPU page, show:
  - Overall CPU usage.
  - Usage for each of the currently expected six logical vCPUs.
  - Linux load averages for 1, 5, and 15 minutes.
  - Historical charts covering retained data.
- Apply the confirmed CPU warning and Critical thresholds.

### FR-7: RAM monitoring

- Show a compact RAM summary on the main dashboard.
- Link the summary to a dedicated RAM page.
- On the RAM page, show:
  - Total RAM.
  - Used RAM.
  - Available RAM.
  - Free RAM.
  - Cached memory.
  - Buffered memory.
  - Usage percentage and 85% warning line.
  - Historical charts covering retained data.
  - Minimum, average, and peak usage for the selected period.
  - Swap status, including the current no-swap state.
  - Linux memory-pressure information.
  - Docker containers ranked by RAM usage, linking to container details.

### FR-8: Filesystem monitoring

- Monitor every mounted filesystem, including `/proc`, `/sys`, `tmpfs`, and other virtual or temporary mounts.
- Show a compact root-filesystem summary on the main dashboard.
- Provide a disk overview page listing every active mount.
- Provide a dedicated detail page for each filesystem.
- On each detail page, show:
  - Mount path and device name.
  - Filesystem type.
  - Total, used, and free capacity.
  - Usage percentage and 90% warning line.
  - Historical usage chart covering retained data.
  - Minimum, average, and peak usage.
  - Read and write activity when the mount maps to a measurable block device.
  - Read-only or read-write mount status.
- Show unsupported measurements for virtual filesystems as unavailable, not erroneous.
- Automatically include newly mounted filesystems.
- Remove unmounted filesystems from the active list while retaining their history for the remainder of the 14-day period.
- Mark retained history for removed mounts as no longer mounted.

### FR-9: Docker overview and discovery

- Automatically discover and display all Docker containers, including stopped containers.
- Do not require manual registration or impose a configured container-count maximum.
- Provide a Docker overview page and a dedicated detail page per container.
- Remove deleted containers from the active list.
- Retain deleted-container history for the remainder of the 14-day period and mark it deleted.

### FR-10: Docker container details

For every container, show:

- Container name.
- Current state: running, stopped, paused, restarting, or other Docker-reported state.
- Docker health-check result when defined.
- An explicit not-configured/unavailable health value when no health check exists.
- Container uptime.
- Restart count.
- Current and historical per-container CPU usage.
- Current and historical per-container RAM usage.
- Published ports.
- Historical state and health changes.

Only Docker's unhealthy result creates the container-health warning. Stopped, paused, and restarting states remain visible but do not independently create a warning.

### FR-11: Docker controls

- Allow start, stop, and restart from the container detail page.
- Require a normal confirmation dialog before every Docker control action.
- Do not require a second secret, PIN, password, or session unlock.
- On failure, show a beginner-friendly summary with expandable raw Docker details.
- Record both successful and failed requests in the administrative audit log.

### FR-12: Container logs

- Read container logs directly from Docker; do not store a separate copy.
- Load the latest 100 lines when the viewer opens.
- Continue with live updates while the viewer remains active.
- Display logs exactly as Docker provides them; do not redact secrets or personal data.
- Provide default case-insensitive plain-text search.
- Provide an optional case-sensitive search mode.
- Provide an explicitly enabled regular-expression mode.
- Highlight matches and safely report invalid regular expressions.
- Apply search to the initially loaded lines and subsequent live lines.
- Provide controls to:
  - Filter standard output and standard error.
  - Pause and resume live updates.
  - Follow the newest line automatically.
  - Stop following when the user scrolls upward.
  - Clear only the browser's current view.
  - Copy selected text.
- Display log timestamps in Africa/Tunis when timestamps are available.
- Make log availability depend on Docker's configured logging driver and retention.

### FR-13: Historical charts

- Provide the following ranges where history applies:
  - Last minute.
  - Last 5 minutes.
  - Last 15 minutes.
  - Last 30 minutes.
  - Last hour.
  - Last 6 hours.
  - Last 24 hours.
  - Last 7 days.
  - Last 14 days.
  - Custom start and end within retained history.
- Keep one-minute sampling for every range; short ranges may contain very few points.
- Provide history for overall CPU, overall RAM, every filesystem, each container's CPU and RAM, and container state/health changes.

### FR-14: Warning and Critical event history

- Retain warning and Critical event records for 14 days after they clear.
- Record event start, end, severity, cause, and affected resource or subsystem.
- Provide a central event-history page.
- Also show relevant events on CPU, RAM, filesystem, and container detail pages.

### FR-15: Data export

- Export CSV and JSON.
- Support preset periods of one hour, 24 hours, 7 days, and 14 days.
- Support a custom start and end within retained history.
- Export:
  - CPU history.
  - RAM history.
  - Filesystem history.
  - Per-container CPU and RAM history.
  - Container state and health changes.
  - Warning and Critical events.
  - Administrative action records.
- Do not include container application logs in history exports.

### FR-16: Administrative audit log

- Retain administrative records for 14 days.
- Record Docker control requests, backup creation and restoration attempts, data exports, configuration reload results, and audit-log deletion.
- When applicable, record timestamp, requesting device IP, action details, and result.
- Permit deletion of selected audit entries, entries in a date range, or all existing entries.
- Require a normal confirmation dialog for every deletion, including deleting all entries.
- After deletion, create a new audit record with the time, requesting IP, deletion scope, and count removed.
- When all earlier entries are deleted, retain the new deletion record.

### FR-17: Backup and restoration

- Back up settings and monitoring history every two days.
- Do not require administrative audit records in the backup.
- Store backups on the same VPS.
- Retain backups for 14 days, normally producing about seven scheduled backup sets.
- Provide a confirmed Create Backup Now action.
- Apply the same retention and storage rules to manual backups.
- Provide backup restoration from the dashboard.
- Require explicit confirmation before restoration.
- When corruption is detected, present a recovery screen that asks whether to restore; never restore automatically.
- Keep the recovery screen usable when the normal history database cannot be opened.
- Before restoration, create a safety copy of the current database and settings, even if they appear corrupted.
- Proceed with restoration only after the safety copy succeeds.

### FR-18: Localization and time

- Provide complete English and French interfaces.
- Permit switching language at any time.
- On first visit, follow the browser language when it is English or French; otherwise use English.
- Display dashboard, chart, log, event, and audit times in Africa/Tunis.

### FR-19: Visual and responsive behavior

- Use a dark theme only.
- Work on desktop, laptop, and mobile browser layouts.
- Represent Healthy, Warning, Critical, and Unknown using color, a distinct icon, and a written label together.
- Do not rely on color alone.

### FR-20: Service lifecycle

- Start automatically after VPS boot.
- Restart automatically after an unexpected application failure.
- Preserve existing history across ordinary restarts and failures.

## Non-functional requirements

### NFR-1: Platform

- Host platform: Ubuntu 25.04 on Linux.
- Current VPS resources:
  - 6 logical vCPUs, Intel Haswell-class virtual CPU.
  - 11 GiB RAM, approximately 8 GiB available at discovery time.
  - No swap configured.
  - 100 GB block device.
  - Root filesystem approximately 96 GB usable, 67 GB used, 30 GB free, and 70% utilized at discovery time.

### NFR-2: Resource efficiency

- Under normal operation, target less than 5% average CPU and less than 512 MiB RAM.
- Permit brief unapproved spikes during startup, crash recovery, retention cleanup, and scheduled/manual backup work.
- Require user confirmation for other optional operations expected to need substantially more resources.
- Limit the monitoring database, audit records, and retained backups together to 5 GB.
- Enforce cleanup and retention before exceeding that limit.

### NFR-3: Availability

- Best-effort personal service; no formal monthly uptime percentage.
- Automatic startup and crash restart remain mandatory.

### NFR-4: Responsiveness

- No strict page-load-time target.
- The interface must remain usable under normal VPS conditions without violating the normal resource target.

### NFR-5: Browser compatibility

- Support the latest stable Google Chrome, Mozilla Firefox, and Opera GX.
- Older versions and other named browsers are not required.

### NFR-6: Quality and testing

- Automated core tests for calculations, thresholds, settings, retention, and cleanup.
- Integration tests for data storage and Docker communication.
- Browser tests for important dashboard workflows.
- Compatibility checks against the supported browsers.

### NFR-7: Data integrity

- Never invent missing measurements.
- Preserve last-known values and collection timestamps during collection failures.
- Reject invalid configuration without losing the last valid configuration.
- Do not restore a backup until a safety copy succeeds.

### NFR-8: Maintainability

- Use Go for backend code and TypeScript for frontend code.
- Keep configuration editable without code changes.
- Keep frameworks and architecture undecided until this context is approved.

## MVP features

1. Tailnet-only HTTP access to one self-monitored VPS, relying on Tailscale transport encryption and network policy.
2. Responsive dark dashboard in English and French.
3. One-minute collection, refresh, 14-day history, stale handling, and honest chart gaps.
4. Main overview with concurrent health badges and warning counts.
5. Detailed CPU monitoring and CPU page.
6. Detailed RAM, swap, pressure, and RAM page.
7. Every mounted filesystem, disk overview, and per-filesystem pages.
8. Automatic discovery and historical monitoring of all Docker containers.
9. Per-container state, health, uptime, restarts, resources, ports, and charts.
10. Confirmed Docker start, stop, and restart actions.
11. Searchable and filterable live Docker log viewer without duplicated storage.
12. Warning, Critical, and administrative audit histories.
13. CSV and JSON export.
14. Automatic/manual same-VPS backup and confirmed dashboard restoration.
15. Automatic application startup and crash recovery.
16. Core, integration, and browser testing.

## Features postponed for later

- System service status and uptime monitoring.
- Port availability checks.
- SSL certificate-expiration monitoring for other services.
- Tailscale/Headscale connectivity monitoring as a monitored signal.
- Recent security-event ingestion and analysis.
- Multiple VPS support.
- Additional security-analyst users.
- Application accounts, roles, and person-level action attribution.
- External notifications through email, Telegram, Discord, or other services.
- Light theme.

## Out-of-scope items

- Public internet access to the dashboard.
- SSH-tunnel-only access as the normal access path.
- Automatic operating-system upgrade or remediation.
- Upgrading Ubuntu 25.04.
- Multi-VPS orchestration.
- Full observability platform features such as distributed tracing.
- Storing a duplicate copy of Docker application logs.
- Exporting container application logs through monitoring-history exports.
- Application-level encryption of the database or backups.
- Off-VPS disaster-recovery backups.
- Automatic backup restoration without user approval.
- Automatic estimation or interpolation of missing measurements.
- Notification delivery outside the dashboard.
- Supporting browser versions older than the latest stable releases.
- Architecture selection or implementation before context approval.

## Constraints

- The host remains on Ubuntu 25.04.
- The application and monitored workloads share the same VPS resources.
- Only about 30 GB of root-filesystem space was free during discovery.
- Combined application data and backups are limited to 5 GB.
- No swap is currently configured.
- Collection and normal refresh occur once per minute, even for one- and five-minute charts.
- Monitoring and Docker administration depend on local Docker availability and permissions.
- Dashboard access depends on the Headscale-managed tailnet being available.
- Backups live on the same VPS and cannot protect against total VPS loss.
- The MVP has no deadline and no stated monetary budget limit.

## Security and privacy considerations

### Confirmed safeguards

- Tailnet-only reachability.
- Tailscale-encrypted transport between approved tailnet devices.
- Application binding and host-firewall rules that reject public and unrelated interfaces.
- Confirmation before every Docker action, manual backup, restore, and audit deletion.
- Fourteen-day administrative auditing with requesting IP where applicable.
- Restricted local file access: the monitoring service, the user's account, `root`, and other `sudo` administrators only.
- Invalid configuration cannot replace the last valid configuration.
- A safety copy is required before restoration.
- The interface communicates status using text and icons as well as color.

### Explicitly accepted risks

- Ubuntu 25.04 reached end of life and does not receive normal security maintenance. The user explicitly decided it will not be upgraded.
- There is no application login, user account, action PIN, or second factor.
- HTTP is used inside the encrypted tailnet. Browsers may display "Not secure," and accidental exposure on a public or unrelated interface would send application traffic without TLS.
- Any permitted or compromised tailnet device that reaches the dashboard can attempt Docker control actions.
- Docker administration is high privilege; compromise of the monitor may affect every container accessible through its Docker connection.
- Container logs are displayed without redaction and may expose secrets, tokens, personal data, or internal network details.
- The database and backups have no application-level encryption.
- Same-VPS backups do not protect against disk failure, VPS deletion, or full-server compromise.
- Audit entries may be deleted from the dashboard after a normal confirmation, although the deletion itself is audited.
- Immediate one-sample thresholds may create short-lived warning and Critical state changes.
- Monitoring every virtual filesystem may create noise and unusual or meaningless capacity values.

### Security matters reserved for architecture

- Exact tailnet interface binding and firewall enforcement.
- Least-privilege Docker access and the security boundary around Docker control.
- Protection of state-changing web requests.
- File ownership and exact Linux permission modes.
- Backup file format and safe restore mechanics.
- Dependency selection, update policy, and supply-chain controls.

## Acceptance criteria

The MVP is acceptable when all of the following are demonstrably true:

1. A supported browser on an approved tailnet-connected device opens the dashboard over its private HTTP address; the accepted browser "Not secure" indicator is not treated as a failure.
2. Requests through the VPS public IP and unrelated interfaces cannot reach the dashboard.
3. The application starts after VPS boot and restarts after an unexpected crash without losing retained history.
4. New host and Docker measurements are normally recorded once per minute and visible after the next one-minute refresh or a manual refresh.
5. Data older than 14 days is cleaned up, and combined data plus backups remain within 5 GB.
6. After two minutes without collection, last-known values are visibly stale with timestamps and a collection warning.
7. Collection outages appear as gaps in charts, with no generated values.
8. The main dashboard shows all confirmed summary values, current badges, active warning count, and latest successful collection time.
9. CPU, RAM, filesystem, and Docker pages expose every confirmed current and historical field.
10. All current mounts, including virtual mounts, appear; unsupported physical-I/O fields show unavailable.
11. Newly mounted filesystems and newly created containers appear without manual registration.
12. Removed mounts and deleted containers leave the active lists while their marked history remains until retention expiry.
13. Warning and Critical thresholds trigger and clear after one qualifying one-minute measurement as confirmed.
14. Simultaneous Critical, Warning, and Unknown conditions can all remain visible with their causes.
15. Invalid settings leave the last valid settings active and create a visible configuration warning and audit entry.
16. All Docker containers appear, including stopped containers; only Docker-unhealthy health creates the container health warning.
17. Start, stop, and restart require confirmation; success and failure are shown and audited.
18. The log viewer loads 100 recent lines, streams new lines, supports the confirmed search modes and controls, and stores no duplicate logs.
19. Warning and Critical events remain searchable/viewable centrally and on relevant detail pages for 14 days.
20. CSV and JSON exports contain the selected datasets and requested valid time range.
21. Scheduled backups run every two days, manual backup works after confirmation, and backups expire after 14 days.
22. Restore is never automatic; a safety copy succeeds before a confirmed restore proceeds.
23. Audit deletion supports selected, ranged, and all-record modes after confirmation and leaves a new deletion audit record.
24. English and French content can be switched at any time; the initial language follows the confirmed browser-language rule.
25. The responsive dark interface remains usable on the required desktop and mobile layouts in the latest stable Chrome, Firefox, and Opera GX.
26. Automated core, integration, and important browser-workflow tests pass.
27. During representative normal operation, average monitor CPU stays below 5% and memory stays below 512 MiB, excluding permitted brief maintenance spikes.

## Confirmed decisions

- Product name: Infrastructure Monitor.
- One self-monitored VPS; Ubuntu 25.04 remains unchanged.
- Current host: 6 logical vCPUs, 11 GiB RAM, no swap, 100 GB disk, approximately 30 GB free at discovery.
- Go backend and TypeScript frontend.
- Personal MVP user only.
- Tailnet-only HTTP over Tailscale transport encryption, no application login or control PIN.
- CPU/RAM/filesystem and Docker are the MVP monitoring domains.
- One-minute sampling and refresh; 14-day rolling history.
- 2-minute stale threshold and visible gaps.
- Warning thresholds: CPU 85%, RAM 85%, filesystem 90%.
- Critical thresholds/conditions: CPU 95%, RAM 95%, filesystem 95%, unhealthy container, inaccessible database, or unavailable Docker communication.
- One-sample trigger and one-normal-sample clearing.
- All mounts and all containers are automatically included.
- Deleted-container and removed-mount history lasts until normal expiry.
- Dashboard-only warnings; no external notifications.
- Docker start/stop/restart, live logs, history, audit, export, backup, and recovery behavior as specified above.
- English and French; Africa/Tunis display time; dark theme.
- Latest stable Chrome, Firefox, and Opera GX.
- Best-effort availability, lightweight normal resource target, and 5 GB data/backup ceiling.

## Remaining assumptions

- Docker Engine is installed and exposes the state, statistics, logs, health information, and control operations needed by the application.
- The VPS kernel exposes the Linux measurements needed for the selected CPU, memory, pressure, filesystem, and block-I/O fields.
- The number of logical vCPUs remains six for initial acceptance testing.
- The user's supported browsers can reach the private HTTP listener through the Headscale-managed tailnet.
- The Headscale/Tailscale access policy permits only the user's intended device or devices.
- The 5 GB ceiling is sufficient for the actual number of mounts, containers, retained events, exports in progress, and backups.
- Docker's own logging configuration retains at least the recent lines the viewer requests.

These are not architecture decisions. They must be verified before implementation or converted into explicit failure behavior.

## Open questions

There are no unresolved product-requirement questions blocking implementation. Implementation-specific verification and deferred decisions are tracked in `TECHNICAL_PLAN.md`, `TASKS.md`, and `docs/implementation-prerequisites.md`.

## Approval gate

The project context and technical architecture are approved. Implementation remains task-gated: only the task explicitly started by the user may be changed.
