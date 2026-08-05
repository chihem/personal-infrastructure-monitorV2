# Storage schema and migration compatibility

The application uses two independent SQLite databases under `/var/lib/pim/`:

- `history.db` stores monitoring resources, minute samples, state changes, collection runs, and incidents.
- `audit.db` stores bounded administrative-action metadata and is deliberately excluded from normal history backups.

Both databases use WAL mode, foreign keys, a five-second busy timeout, synchronous `NORMAL`, automatic WAL checkpoints, and a four-connection pool. Timestamps in product tables are UTC Unix seconds. Byte counts are integers and percentages are numeric values.

Database files and SQLite sidecar files use owner-only permissions on Linux. Database paths must end in `.db`, and symbolic links are rejected. Runtime paths must remain outside embedded/static web assets; production defaults are `/var/lib/pim/history.db` and `/var/lib/pim/audit.db`.

## Migration policy

Migrations are embedded into the Go binary and applied transactionally by Goose. `schema_migrations` records applied versions. `schema_compatibility` records the database kind, current schema, minimum reader, and the oldest compatible schema version. A binary refuses a database newer than its embedded migrations, and read-only mode refuses incomplete schemas.

Each new migration must update this table and this document. Destructive automatic down-migrations are not provided; deployment creates a verified pre-migration backup before a schema-changing release.

| Database | Version | Migration | Compatibility with previous release | Rollback requirement |
|---|---:|---|---|---|
| history | 1 | Initial collection/resource/sample/incident schema | No earlier database-backed release exists | Previous scaffold ignores the new database; no destructive down migration |
| audit | 1 | Initial administrative-audit schema | No earlier database-backed release exists | Previous scaffold ignores the new database; no destructive down migration |

The Goose-owned migration timestamp is migration metadata. All application data timestamps, including `schema_compatibility.applied_at_unix`, use UTC Unix seconds.

## Sensitive-data boundary

Neither schema has columns for container logs, request headers, confirmation tokens, environment variables, Docker configuration, exported content, or secrets. `audit_entries.parameters_summary` is reserved for a small allowlisted JSON summary and must be validated by the later audit repository before insertion.
