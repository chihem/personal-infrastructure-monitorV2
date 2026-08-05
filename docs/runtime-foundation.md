# Runtime foundation

FND-04 provides the executable base for the Infrastructure Monitor. It does not collect host or Docker measurements yet.

## Startup sequence

1. Load and validate `/etc/pim/settings.toml`, with the protected last-valid/default recovery behavior from FND-02.
2. Resolve the active IPv4 address from the interface named exactly `tailscale0`.
3. Open a `tcp4` listener on that exact address and the configured non-privileged port.
4. Open `history.db`; startup stops if this mandatory store cannot be opened.
5. Open `audit.db`; if it is unavailable, read-only service startup continues and health reports a degraded state. Future state-changing actions must reject requests while audit storage is unavailable.
6. Start the settings watcher and serve the embedded UI plus `/api/v1` routes.

Startup never falls back to `0.0.0.0`, `::`, another interface, or localhost. The production listener also verifies that its actual socket address matches the discovered Tailscale IPv4 address and configured port.

## Shutdown sequence

`cmd/pim` handles `SIGINT` and `SIGTERM`. Cancellation stops the settings watcher, asks the HTTP server to drain active requests for up to 15 seconds, forcibly closes the server only if that deadline expires, and then closes the audit and history databases.

## Current HTTP surface

| Path | Purpose |
| --- | --- |
| `/` and extensionless UI routes | Embedded FND-04 placeholder page |
| `/app.css` | Embedded placeholder style |
| `/api/v1/health` | Temporary foundation health response |
| Other `/api/` paths | JSON not-found response |

The placeholder health response reports whether maintenance mode is active, whether both databases opened, and the configuration manager's current state. FND-07 will expand operational liveness/readiness behavior.

Shared HTTP safeguards currently include:

- the validated read-header, read, write, and idle timeouts;
- a 16 KiB request-header limit;
- a 1 MiB request-body limit;
- same-origin embedded assets with no permissive CORS behavior;
- Content Security Policy, frame denial, nosniff, and no-referrer headers;
- removal of `Forwarded`, `X-Forwarded-*`, and `X-Real-IP` before application handlers run.

Host/Origin validation, CSRF protection, action confirmation intents, and rate limits belong to SEC-01 before state-changing or expensive endpoints are released.

## Configuration behavior

Threshold and future collector settings remain available through the atomic live configuration snapshot. The listener port and HTTP timeout fields are startup settings in this foundation: changing them updates the validated settings snapshot but does not rebind or mutate an active HTTP server. Restart the service to apply those server fields. A later task must make this distinction visible before administrator-facing configuration editing is released.

## Local and production verification

Run the focused platform-independent suite from the repository root:

```text
go test ./internal/app ./internal/api ./internal/web -count=1
```

Run the repository quality suite:

```text
npm run check
```

The integration tests inject a loopback listener only inside the test process; production `Run` always uses strict `tailscale0` discovery. Final deployment verification must inspect the real Linux listening socket and prove that the port is reachable from an approved tailnet device but not through the VPS public interface. That VPS-level check belongs to SEC-06.
