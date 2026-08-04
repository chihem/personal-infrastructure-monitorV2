# Implementation Prerequisites

Status: **Complete for SET-01**
Verified: 2026-08-04
Target: Hermes VPS
Verification method: user-provided output from the read-only Light-agent inspection

## Purpose

This record locks the minimum facts needed before replacing the legacy prototype. It contains no complete Tailscale address, credential, private key, environment value, container name, application log, or partner data.

Codex did not connect to Hermes. Light performed the approved read-only checks and reported that it made no changes.

## Confirmed decisions

### Browser transport and reachability

- The dashboard is private infrastructure, not a public website.
- The Go service will serve HTTP directly on the IPv4 address assigned to `tailscale0`.
- Tailscale provides encrypted transport between tailnet devices.
- The user accepts that browsers may display "Not secure" for the HTTP page.
- No public domain, DNS API credential, browser-trusted certificate, TLS private key, Caddy installation, or reverse proxy is required.
- The application must fail closed if it cannot discover or bind the `tailscale0` address. It must never fall back to a wildcard or public listener.
- Headscale policy and the existing host firewall must allow the configured application port only from approved tailnet devices.

### Missing memory-pressure information

- Hermes currently exposes readable Linux PSI memory-pressure information.
- If PSI becomes unavailable, the RAM page will show **Memory pressure: Unknown** and add an **Unknown** health badge.
- Ordinary RAM monitoring continues when PSI alone is unavailable.

### Build tool lines

- Backend build line: Go 1.26, using the latest security/patch release available when the dependency lock is created. The verified current release on 2026-08-04 is Go 1.26.5.
- Frontend build line: Node.js 24 LTS, using the latest 24.x LTS patch available when the dependency lock is created.
- Node.js is a build-time dependency only. The production release will embed the frontend and will not require a Node.js runtime.
- Go is a build-time dependency only when a prebuilt Linux amd64 release artifact is deployed.

Official release references:

- [Go release history](https://go.dev/doc/devel/release)
- [Node.js release schedule](https://nodejs.org/en/about/previous-releases)

## Verified Hermes facts

| Check | Result | Evidence | Status |
|---|---|---|---|
| Operating system | Ubuntu 25.04 | `/etc/os-release` reported `PRETTY_NAME="Ubuntu 25.04"` | Confirmed |
| Kernel CPU architecture | `x86_64` | `uname -m` | Confirmed |
| Debian package architecture | `amd64` | `dpkg --print-architecture` | Confirmed |
| Memory PSI | Exists and is readable | `/proc/pressure/memory` | Confirmed |
| Tailscale version | 1.98.10 | `tailscale version` | Confirmed |
| Tailscale interface | `tailscale0` exists and is UP | `ip link show tailscale0` | Confirmed |
| Tailscale IPv4 | Address exists; full value deliberately not recorded | `100.64.0.REDACTED` | Confirmed |
| Docker socket | `/var/run/docker.sock` exists and is a Unix socket | Light inspection | Confirmed |
| Docker socket access | `root:docker`, mode `660` | `stat` | Confirmed |
| Current-user Docker query | Docker server query succeeded | `docker version` returned server data | Confirmed |
| Docker client | 29.2.1 | `docker version` | Confirmed |
| Docker server | 29.2.1 | `docker version` | Confirmed |
| systemd | 257 (`257.4-1ubuntu3.2`) | `/usr/lib/systemd/systemd --version` | Confirmed |
| Go on Hermes | Not installed | `go` command not found | Unavailable; not required at runtime |
| Node.js on Hermes | v22.23.1 | `node --version` | Confirmed; supported but not selected for the new build line |
| npm on Hermes | 10.9.8 | `npm --version` | Confirmed |
| Caddy on Hermes | Not installed | `caddy` command not found | Unavailable; no longer required |

## Security implications

- Direct Docker socket access is root-equivalent. This risk remains explicitly accepted and must be tested and documented throughout implementation.
- HTTP is acceptable only because the listener is restricted to the encrypted tailnet. A wildcard or public-interface bind is a release-blocking security failure.
- The complete Tailscale address remains outside tracked documentation. The application discovers it from `tailscale0` at startup.
- No installation, configuration, firewall, Docker, service, or package change was made during SET-01.
- Ubuntu 25.04 is end-of-life. The user previously accepted this risk and decided that the operating system will not be upgraded.

## Deferred implementation details

These do not block SET-01 and belong to their named roadmap tasks:

- The configurable non-privileged HTTP port and validation bounds: `FND-02`.
- Tailscale-interface discovery and fail-closed listener behavior: `FND-04`.
- Exact Headscale policy and firewall rules, with recovery-access protection: `SEC-06`.
- Go/Node installation or local build-environment setup: `SET-03`.
- Docker access for the future dedicated `pim` service account: `SEC-05`.

## SET-01 acceptance review

- PSI behavior is unambiguous: **Pass**.
- Browser transport and public-exposure boundary are unambiguous: **Pass**.
- Tailnet interface/address availability is verified without recording the complete address: **Pass**.
- Docker socket availability and current access are verified: **Pass**.
- Target operating system and CPU architecture are verified: **Pass**.
- Supported Go and Node.js build lines are recorded from official sources: **Pass**.
- No prerequisite remains Blocked: **Pass**.
- No real secret was created or recorded: **Pass**.
