-- +goose Up
-- Compatibility: first history schema; no earlier database-backed release exists.

CREATE TABLE schema_compatibility (
    singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
    schema_kind TEXT NOT NULL CHECK (schema_kind = 'history'),
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
) VALUES (1, 'history', 1, 1, 1, unixepoch());

CREATE TABLE collection_runs (
    id INTEGER PRIMARY KEY,
    started_at_unix INTEGER NOT NULL CHECK (started_at_unix > 0),
    finished_at_unix INTEGER CHECK (finished_at_unix IS NULL OR finished_at_unix >= started_at_unix),
    trigger_kind TEXT NOT NULL CHECK (trigger_kind IN ('scheduled', 'manual')),
    result TEXT NOT NULL CHECK (result IN ('succeeded', 'partial', 'failed')),
    host_result TEXT NOT NULL CHECK (host_result IN ('succeeded', 'partial', 'failed', 'not_attempted')),
    docker_result TEXT NOT NULL CHECK (docker_result IN ('succeeded', 'partial', 'failed', 'not_attempted')),
    error_code TEXT CHECK (error_code IS NULL OR length(error_code) BETWEEN 1 AND 64)
) STRICT;

CREATE INDEX collection_runs_started_at_idx ON collection_runs (started_at_unix);

CREATE TABLE host_samples (
    collection_run_id INTEGER PRIMARY KEY REFERENCES collection_runs(id) ON DELETE CASCADE,
    observed_at_unix INTEGER NOT NULL CHECK (observed_at_unix > 0),
    availability TEXT NOT NULL CHECK (availability IN ('available', 'unavailable')),
    unavailable_reason TEXT,
    overall_cpu_percent REAL CHECK (overall_cpu_percent BETWEEN 0 AND 100),
    load_1 REAL CHECK (load_1 >= 0),
    load_5 REAL CHECK (load_5 >= 0),
    load_15 REAL CHECK (load_15 >= 0),
    memory_total_bytes INTEGER CHECK (memory_total_bytes >= 0),
    memory_used_bytes INTEGER CHECK (memory_used_bytes >= 0),
    memory_available_bytes INTEGER CHECK (memory_available_bytes >= 0),
    memory_free_bytes INTEGER CHECK (memory_free_bytes >= 0),
    memory_cached_bytes INTEGER CHECK (memory_cached_bytes >= 0),
    memory_buffered_bytes INTEGER CHECK (memory_buffered_bytes >= 0),
    memory_usage_percent REAL CHECK (memory_usage_percent BETWEEN 0 AND 100),
    swap_total_bytes INTEGER CHECK (swap_total_bytes >= 0),
    swap_used_bytes INTEGER CHECK (swap_used_bytes >= 0),
    psi_availability TEXT NOT NULL CHECK (psi_availability IN ('available', 'unavailable')),
    psi_unavailable_reason TEXT,
    memory_psi_some_avg10 REAL CHECK (memory_psi_some_avg10 >= 0),
    memory_psi_full_avg10 REAL CHECK (memory_psi_full_avg10 >= 0),
    memory_psi_some_total_us INTEGER CHECK (memory_psi_some_total_us >= 0),
    memory_psi_full_total_us INTEGER CHECK (memory_psi_full_total_us >= 0),
    CHECK (
        (availability = 'available' AND unavailable_reason IS NULL) OR
        (availability = 'unavailable' AND unavailable_reason IS NOT NULL)
    ),
    CHECK (
        (psi_availability = 'available' AND psi_unavailable_reason IS NULL) OR
        (psi_availability = 'unavailable' AND psi_unavailable_reason IS NOT NULL)
    )
) STRICT;

CREATE INDEX host_samples_observed_at_idx ON host_samples (observed_at_unix);

CREATE TABLE cpu_core_samples (
    collection_run_id INTEGER NOT NULL REFERENCES collection_runs(id) ON DELETE CASCADE,
    logical_index INTEGER NOT NULL CHECK (logical_index >= 0),
    observed_at_unix INTEGER NOT NULL CHECK (observed_at_unix > 0),
    availability TEXT NOT NULL CHECK (availability IN ('available', 'unavailable')),
    unavailable_reason TEXT,
    usage_percent REAL CHECK (usage_percent BETWEEN 0 AND 100),
    PRIMARY KEY (collection_run_id, logical_index),
    CHECK (
        (availability = 'available' AND unavailable_reason IS NULL AND usage_percent IS NOT NULL) OR
        (availability = 'unavailable' AND unavailable_reason IS NOT NULL AND usage_percent IS NULL)
    )
) STRICT;

CREATE INDEX cpu_core_samples_time_idx ON cpu_core_samples (logical_index, observed_at_unix);

CREATE TABLE filesystems (
    id TEXT PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 128),
    mount_path TEXT NOT NULL CHECK (length(mount_path) BETWEEN 1 AND 4096),
    device_name TEXT NOT NULL CHECK (length(device_name) BETWEEN 1 AND 4096),
    filesystem_type TEXT NOT NULL CHECK (length(filesystem_type) BETWEEN 1 AND 128),
    first_seen_at_unix INTEGER NOT NULL CHECK (first_seen_at_unix > 0),
    last_seen_at_unix INTEGER NOT NULL CHECK (last_seen_at_unix >= first_seen_at_unix),
    removed_at_unix INTEGER CHECK (removed_at_unix IS NULL OR removed_at_unix >= first_seen_at_unix)
) STRICT;

CREATE INDEX filesystems_active_idx ON filesystems (removed_at_unix, mount_path);

CREATE TABLE filesystem_samples (
    collection_run_id INTEGER NOT NULL REFERENCES collection_runs(id) ON DELETE CASCADE,
    filesystem_id TEXT NOT NULL REFERENCES filesystems(id) ON DELETE RESTRICT,
    observed_at_unix INTEGER NOT NULL CHECK (observed_at_unix > 0),
    availability TEXT NOT NULL CHECK (availability IN ('available', 'unavailable')),
    unavailable_reason TEXT,
    total_bytes INTEGER CHECK (total_bytes >= 0),
    used_bytes INTEGER CHECK (used_bytes >= 0),
    free_bytes INTEGER CHECK (free_bytes >= 0),
    usage_percent REAL CHECK (usage_percent BETWEEN 0 AND 100),
    mount_mode TEXT CHECK (mount_mode IN ('read_only', 'read_write')),
    PRIMARY KEY (collection_run_id, filesystem_id),
    CHECK (
        (availability = 'available' AND unavailable_reason IS NULL AND total_bytes IS NOT NULL) OR
        (availability = 'unavailable' AND unavailable_reason IS NOT NULL)
    )
) STRICT;

CREATE INDEX filesystem_samples_time_idx ON filesystem_samples (filesystem_id, observed_at_unix);

CREATE TABLE block_devices (
    id TEXT PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 128),
    device_name TEXT NOT NULL CHECK (length(device_name) BETWEEN 1 AND 4096),
    first_seen_at_unix INTEGER NOT NULL CHECK (first_seen_at_unix > 0),
    last_seen_at_unix INTEGER NOT NULL CHECK (last_seen_at_unix >= first_seen_at_unix),
    removed_at_unix INTEGER CHECK (removed_at_unix IS NULL OR removed_at_unix >= first_seen_at_unix)
) STRICT;

CREATE TABLE block_device_io_samples (
    collection_run_id INTEGER NOT NULL REFERENCES collection_runs(id) ON DELETE CASCADE,
    block_device_id TEXT NOT NULL REFERENCES block_devices(id) ON DELETE RESTRICT,
    observed_at_unix INTEGER NOT NULL CHECK (observed_at_unix > 0),
    availability TEXT NOT NULL CHECK (availability IN ('available', 'unavailable')),
    unavailable_reason TEXT,
    read_bytes_total INTEGER CHECK (read_bytes_total >= 0),
    write_bytes_total INTEGER CHECK (write_bytes_total >= 0),
    read_bytes_per_second REAL CHECK (read_bytes_per_second >= 0),
    write_bytes_per_second REAL CHECK (write_bytes_per_second >= 0),
    PRIMARY KEY (collection_run_id, block_device_id),
    CHECK (
        (availability = 'available' AND unavailable_reason IS NULL) OR
        (availability = 'unavailable' AND unavailable_reason IS NOT NULL)
    )
) STRICT;

CREATE INDEX block_device_io_samples_time_idx ON block_device_io_samples (block_device_id, observed_at_unix);

CREATE TABLE containers (
    docker_id TEXT PRIMARY KEY CHECK (length(docker_id) BETWEEN 12 AND 128),
    current_name TEXT NOT NULL CHECK (length(current_name) BETWEEN 1 AND 256),
    first_seen_at_unix INTEGER NOT NULL CHECK (first_seen_at_unix > 0),
    last_seen_at_unix INTEGER NOT NULL CHECK (last_seen_at_unix >= first_seen_at_unix),
    deleted_at_unix INTEGER CHECK (deleted_at_unix IS NULL OR deleted_at_unix >= first_seen_at_unix)
) STRICT;

CREATE INDEX containers_active_idx ON containers (deleted_at_unix, current_name);

CREATE TABLE container_samples (
    collection_run_id INTEGER NOT NULL REFERENCES collection_runs(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL REFERENCES containers(docker_id) ON DELETE RESTRICT,
    observed_at_unix INTEGER NOT NULL CHECK (observed_at_unix > 0),
    availability TEXT NOT NULL CHECK (availability IN ('available', 'unavailable')),
    unavailable_reason TEXT,
    state TEXT,
    health TEXT,
    cpu_percent REAL CHECK (cpu_percent >= 0),
    memory_used_bytes INTEGER CHECK (memory_used_bytes >= 0),
    memory_limit_bytes INTEGER CHECK (memory_limit_bytes >= 0),
    memory_percent REAL CHECK (memory_percent >= 0),
    uptime_seconds INTEGER CHECK (uptime_seconds >= 0),
    restart_count INTEGER CHECK (restart_count >= 0),
    published_ports_json TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(published_ports_json)),
    PRIMARY KEY (collection_run_id, container_id),
    CHECK (
        (availability = 'available' AND unavailable_reason IS NULL) OR
        (availability = 'unavailable' AND unavailable_reason IS NOT NULL)
    )
) STRICT;

CREATE INDEX container_samples_time_idx ON container_samples (container_id, observed_at_unix);

CREATE TABLE container_state_events (
    id INTEGER PRIMARY KEY,
    container_id TEXT NOT NULL REFERENCES containers(docker_id) ON DELETE RESTRICT,
    observed_at_unix INTEGER NOT NULL CHECK (observed_at_unix > 0),
    previous_state TEXT,
    current_state TEXT NOT NULL CHECK (length(current_state) BETWEEN 1 AND 64),
    previous_health TEXT,
    current_health TEXT
) STRICT;

CREATE INDEX container_state_events_time_idx ON container_state_events (container_id, observed_at_unix);

CREATE TABLE incidents (
    id INTEGER PRIMARY KEY,
    health_state TEXT NOT NULL CHECK (health_state IN ('warning', 'critical', 'unknown')),
    cause_code TEXT NOT NULL CHECK (length(cause_code) BETWEEN 1 AND 64),
    subject_kind TEXT NOT NULL CHECK (length(subject_kind) BETWEEN 1 AND 64),
    subject_id TEXT NOT NULL CHECK (length(subject_id) BETWEEN 1 AND 128),
    subject_display_name TEXT NOT NULL CHECK (length(subject_display_name) BETWEEN 1 AND 256),
    started_at_unix INTEGER NOT NULL CHECK (started_at_unix > 0),
    updated_at_unix INTEGER NOT NULL CHECK (updated_at_unix >= started_at_unix),
    ended_at_unix INTEGER CHECK (ended_at_unix IS NULL OR ended_at_unix >= started_at_unix)
) STRICT;

CREATE INDEX incidents_started_at_idx ON incidents (started_at_unix);
CREATE INDEX incidents_subject_time_idx ON incidents (subject_kind, subject_id, started_at_unix);
CREATE INDEX incidents_active_idx ON incidents (ended_at_unix) WHERE ended_at_unix IS NULL;
