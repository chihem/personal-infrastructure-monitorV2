-- +goose Up
-- Compatibility: first audit schema; no earlier database-backed release exists.

CREATE TABLE schema_compatibility (
    singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
    schema_kind TEXT NOT NULL CHECK (schema_kind = 'audit'),
    current_version INTEGER NOT NULL CHECK (current_version >= 1),
    minimum_reader_version INTEGER NOT NULL CHECK (minimum_reader_version >= 1),
    backward_compatible_from_version INTEGER NOT NULL CHECK (backward_compatible_from_version >= 1),
    applied_at_unix INTEGER NOT NULL CHECK (applied_at_unix > 0)
) STRICT;

INSERT INTO schema_compatibility (
    singleton_id,
    schema_kind,
    current_version,
    minimum_reader_version,
    backward_compatible_from_version,
    applied_at_unix
) VALUES (1, 'audit', 1, 1, 1, unixepoch());

CREATE TABLE audit_entries (
    id INTEGER PRIMARY KEY,
    action_type TEXT NOT NULL CHECK (length(action_type) BETWEEN 1 AND 64),
    requested_at_unix INTEGER NOT NULL CHECK (requested_at_unix > 0),
    source_ip TEXT CHECK (source_ip IS NULL OR length(source_ip) BETWEEN 2 AND 45),
    target_kind TEXT NOT NULL CHECK (length(target_kind) BETWEEN 1 AND 64),
    target_id TEXT CHECK (target_id IS NULL OR length(target_id) BETWEEN 1 AND 128),
    parameters_summary TEXT NOT NULL DEFAULT '{}' CHECK (
        length(parameters_summary) <= 2048 AND json_valid(parameters_summary)
    ),
    outcome TEXT NOT NULL CHECK (outcome IN ('pending', 'succeeded', 'failed', 'rejected')),
    completed_at_unix INTEGER CHECK (completed_at_unix IS NULL OR completed_at_unix >= requested_at_unix),
    error_code TEXT CHECK (error_code IS NULL OR length(error_code) BETWEEN 1 AND 64),
    error_detail TEXT CHECK (error_detail IS NULL OR length(error_detail) BETWEEN 1 AND 1024)
) STRICT;

CREATE INDEX audit_entries_requested_at_idx ON audit_entries (requested_at_unix);
CREATE INDEX audit_entries_action_time_idx ON audit_entries (action_type, requested_at_unix);
CREATE INDEX audit_entries_outcome_time_idx ON audit_entries (outcome, requested_at_unix);
