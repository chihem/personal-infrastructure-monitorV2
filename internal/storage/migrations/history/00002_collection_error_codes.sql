-- +goose Up
-- Compatibility: additive collection diagnostics. Version-1 binaries refuse
-- newer schemas, so rollback requires the documented pre-migration backup.

ALTER TABLE collection_runs ADD COLUMN host_error_code TEXT
    CHECK (host_error_code IS NULL OR length(host_error_code) BETWEEN 1 AND 64);

ALTER TABLE collection_runs ADD COLUMN docker_error_code TEXT
    CHECK (docker_error_code IS NULL OR length(docker_error_code) BETWEEN 1 AND 64);

UPDATE schema_compatibility
SET current_version = 2,
    minimum_reader_version = 2,
    backward_compatible_from_version = 2,
    applied_at_unix = unixepoch()
WHERE singleton_id = 1 AND schema_kind = 'history';
