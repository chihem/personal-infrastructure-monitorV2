# Legacy Prototype Recovery

Status: **Preserved and retired from the active development line**
Preserved: 2026-08-04
Git tag: `legacy-prototype-v0.2.0`
Commit: `05cf47fa263f760ef56e1e848c78f5602d332849`

## What was preserved

The tag contains the complete tracked v0.2.0 prototype:

- Windows Python/FastAPI collector and local SQLite history.
- React/TypeScript Vinext/Next frontend.
- Cloudflare Sites/Worker metadata and build integration.
- Read-only Hermes Light report integration.
- PowerShell start/stop scripts and the legacy automated tests.

The active project does not reuse these runtime assumptions. The approved replacement runs directly on Hermes using Go, React/Vite, systemd, local SQLite, and direct Docker access.

## Baseline verification before retirement

The legacy test command was run before removing its tracked files:

```powershell
npm test
```

Actual result on 2026-08-04:

- Vinext production build: passed.
- Node rendered-dashboard test: 1 passed, 0 failed.
- Python tests: 11 passed, 0 failed.
- Python emitted one Starlette deprecation warning about `httpx`; it did not fail the suite.

## Inspect without changing the active tree

```powershell
git show --stat legacy-prototype-v0.2.0
git ls-tree -r --name-only legacy-prototype-v0.2.0
```

## Restore into a separate worktree

Use a separate directory so the approved replacement is not overwritten:

```powershell
git worktree add ..\Personal-Infrastructure-Monitor-Legacy legacy-prototype-v0.2.0
```

This creates a detached worktree at the preserved tag. Do not use a destructive reset of the active project.

## Local data boundary

Ignored machine-local artifacts were not added to the tag and were not deleted during SET-02. This includes `config.local.json`, `data/`, `.venv/`, logs, generated build output, and deployment archives. They may contain private machine state and require separate approval before removal or archival.

The tag is local unless explicitly pushed later. Its target commit is also present on `origin/main` at the time of preservation.
