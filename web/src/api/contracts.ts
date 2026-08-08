export const API_VERSION = "v1" as const;

export type HealthState = "healthy" | "warning" | "critical" | "unknown";
export type Availability = "available" | "unavailable";
export type FreshnessState = "fresh" | "stale" | "unavailable";
export type Unit =
  | "none"
  | "percent"
  | "bytes"
  | "bytes_per_second"
  | "count"
  | "seconds"
  | "load"
  | "microseconds";
export type UnavailabilityReason =
  | "not_collected"
  | "not_supported"
  | "not_configured"
  | "collector_error"
  | "permission_denied"
  | "dependency_unavailable";

export type Metric<T extends number = number> =
  | {
      availability: "available";
      value: T;
      unit: Unit;
      reasonCode: null;
    }
  | {
      availability: "unavailable";
      value: null;
      unit: Unit;
      reasonCode: UnavailabilityReason;
    };

export type Freshness =
  | {
      state: "fresh";
      observedAt: string;
      lastSuccessfulAt: string;
    }
  | {
      state: "stale";
      observedAt: string | null;
      lastSuccessfulAt: string;
    }
  | {
      state: "unavailable";
      observedAt: null;
      lastSuccessfulAt: null;
    };

export type ResourceKind =
  | "host"
  | "cpu"
  | "memory"
  | "filesystem"
  | "block_device"
  | "docker"
  | "container"
  | "history_database"
  | "audit_database"
  | "configuration"
  | "backup"
  | "export";

export type CauseCode =
  | "cpu_warning"
  | "cpu_critical"
  | "memory_warning"
  | "memory_critical"
  | "filesystem_warning"
  | "filesystem_critical"
  | "container_unhealthy"
  | "docker_unavailable"
  | "history_unavailable"
  | "audit_unavailable"
  | "configuration_invalid"
  | "collection_stale"
  | "host_measurements_unavailable"
  | "memory_pressure_unavailable"
  | "storage_limit_reached";

const healthStates = new Set<HealthState>([
  "healthy",
  "warning",
  "critical",
  "unknown",
]);
const units = new Set<Unit>([
  "none",
  "percent",
  "bytes",
  "bytes_per_second",
  "count",
  "seconds",
  "load",
  "microseconds",
]);
const unavailabilityReasons = new Set<UnavailabilityReason>([
  "not_collected",
  "not_supported",
  "not_configured",
  "collector_error",
  "permission_denied",
  "dependency_unavailable",
]);
const causeCodes = new Set<CauseCode>([
  "cpu_warning",
  "cpu_critical",
  "memory_warning",
  "memory_critical",
  "filesystem_warning",
  "filesystem_critical",
  "container_unhealthy",
  "docker_unavailable",
  "history_unavailable",
  "audit_unavailable",
  "configuration_invalid",
  "collection_stale",
  "host_measurements_unavailable",
  "memory_pressure_unavailable",
  "storage_limit_reached",
]);

export interface ResourceRef {
  kind: ResourceKind;
  id: string;
  displayName: string;
}

export type ErrorCode =
  | "validation_failed"
  | "not_found"
  | "unavailable"
  | "conflict"
  | "confirmation_required"
  | "confirmation_expired"
  | "rate_limited"
  | "internal_error"
  | "docker_unavailable"
  | "docker_action_failed"
  | "history_unavailable"
  | "audit_unavailable"
  | "settings_invalid"
  | "storage_limit_reached"
  | "backup_failed"
  | "restore_failed"
  | "export_failed";

export interface FieldError {
  field: string;
  code: string;
  messageKey: string;
  fallbackMessage: string;
}

export interface APIError {
  code: ErrorCode;
  messageKey: string;
  fallbackMessage: string;
  technicalDetail: string | null;
  fieldErrors: FieldError[];
}

export type APIResponse<T> =
  | {
      apiVersion: typeof API_VERSION;
      requestId: string;
      generatedAt: string;
      data: T;
      error: null;
    }
  | {
      apiVersion: typeof API_VERSION;
      requestId: string;
      generatedAt: string;
      data: null;
      error: APIError;
    };

export interface PageInfo {
  limit: number;
  hasMore: boolean;
  nextCursor: string | null;
}

export interface Page<T> {
  items: T[];
  page: PageInfo;
}

export type RangePreset =
  | "last_1m"
  | "last_5m"
  | "last_15m"
  | "last_30m"
  | "last_1h"
  | "last_6h"
  | "last_24h"
  | "last_7d"
  | "last_14d"
  | "custom";

export type CustomRangeSelection = {
  preset: "custom";
  start: string;
  end: string;
};
export type RangeSelection =
  | { preset: Exclude<RangePreset, "custom">; start: null; end: null }
  | CustomRangeSelection;

export interface ResolvedRange {
  preset: RangePreset;
  start: string;
  end: string;
}

export interface HealthCause {
  code: CauseCode;
  state: Exclude<HealthState, "healthy">;
  resource: ResourceRef;
  startedAt: string;
  messageKey: string;
}

export interface HealthSummary {
  states: HealthState[];
  activeWarningCount: number;
  causes: HealthCause[];
}

export interface CPUCore {
  index: number;
  usage: Metric;
}

export interface CPUSnapshot {
  resource: ResourceRef;
  freshness: Freshness;
  overall: Metric;
  cores: CPUCore[];
  load: {
    oneMinute: Metric;
    fiveMinutes: Metric;
    fifteenMinutes: Metric;
  };
  logicalCpuCount: number;
}

export interface PressureWindow {
  average10Seconds: Metric;
  average60Seconds: Metric;
  average300Seconds: Metric;
  total: Metric;
}

export interface MemorySnapshot {
  resource: ResourceRef;
  freshness: Freshness;
  total: Metric;
  used: Metric;
  available: Metric;
  free: Metric;
  cached: Metric;
  buffered: Metric;
  usage: Metric;
  swap: {
    configured: boolean | null;
    total: Metric;
    used: Metric;
    free: Metric;
  };
  pressure: {
    some: PressureWindow;
    full: PressureWindow;
  };
}

export interface FilesystemSnapshot {
  resource: ResourceRef;
  freshness: Freshness;
  mountPath: string;
  deviceName: string;
  filesystemType: string;
  mounted: boolean;
  readOnly: boolean;
  total: Metric;
  used: Metric;
  free: Metric;
  usage: Metric;
  io: {
    readRate: Metric;
    writeRate: Metric;
  };
}

export type ContainerState =
  "running" | "stopped" | "paused" | "restarting" | "other";
export type ContainerHealth =
  "healthy" | "unhealthy" | "starting" | "not_configured" | "unavailable";

export interface PublishedPort {
  protocol: "tcp" | "udp" | "sctp";
  containerPort: number;
  hostIp: string;
  hostPort: number;
}

export interface ContainerSnapshot {
  resource: ResourceRef;
  freshness: Freshness;
  name: string;
  state: ContainerState;
  health: ContainerHealth;
  deleted: boolean;
  uptime: Metric;
  restartCount: Metric;
  cpuUsage: Metric;
  memoryUsage: Metric;
  ports: PublishedPort[];
}

export interface DockerSnapshot {
  resource: ResourceRef;
  freshness: Freshness;
  communication:
    | { state: "available"; reasonCode: null }
    | { state: "unavailable"; reasonCode: UnavailabilityReason };
  containers: ContainerSnapshot[];
}

export type ConfigurationStatus =
  | {
      resource: ResourceRef;
      state: "valid";
      loadedAt: string;
      errorCode: null;
      errorMessageKey: null;
    }
  | {
      resource: ResourceRef;
      state: "using_previous" | "unavailable";
      loadedAt: string | null;
      errorCode: ErrorCode;
      errorMessageKey: string;
    };

export interface MonitoringSnapshot {
  freshness: Freshness;
  health: HealthSummary;
  cpu: CPUSnapshot;
  memory: MemorySnapshot;
  filesystems: FilesystemSnapshot[];
  docker: DockerSnapshot;
  configuration: ConfigurationStatus;
}

export type MonitoringResponse = APIResponse<MonitoringSnapshot>;

export type OperationalState = "ok" | "degraded" | "maintenance" | "not_ready";
export type DependencyState =
  | "available"
  | "unavailable"
  | "not_started"
  | "not_implemented"
  | "not_checked";
export type CollectionStatus =
  "succeeded" | "partial" | "failed" | "not_attempted";

export interface DatabaseOperationalStatus {
  state: "available" | "unavailable";
  sizeBytes: number | null;
}

export interface CollectionRunOperationalStatus {
  startedAt: string;
  finishedAt: string;
  durationMs: number;
  status: Exclude<CollectionStatus, "not_attempted">;
}

export interface CollectionOperationalStatus {
  state: "available" | "unavailable" | "not_started";
  inProgress: boolean;
  lastRun: CollectionRunOperationalStatus | null;
  lastSuccessfulAt: string | null;
}

export interface OperationalStatusSnapshot {
  state: OperationalState;
  uptimeSeconds: number;
  maintenance: boolean;
  configurationState: "valid" | "using_previous" | "unavailable";
  historyDatabase: DatabaseOperationalStatus;
  auditDatabase: DatabaseOperationalStatus;
  collection: CollectionOperationalStatus;
  backupState: "available" | "unavailable" | "not_implemented";
  dockerConnectivity: "available" | "unavailable" | "not_checked";
}

export interface LivenessStatus {
  alive: true;
}

export interface ReadinessStatus {
  ready: boolean;
  maintenance: boolean;
  configurationState: "valid" | "using_previous" | "unavailable";
  historyDatabaseAvailable: boolean;
}

export type OperationalStatusResponse = APIResponse<OperationalStatusSnapshot>;

export type CPUCurrentResponse = APIResponse<CPUSnapshot>;
export type CPUMetric = "overall" | "core" | "load_1" | "load_5" | "load_15";
export type CPUHistoryPoint =
  | {
      start: string;
      end: string;
      state: "observed";
      observedSamples: number;
      availableSamples: number;
      minimum: number;
      average: number;
      maximum: number;
    }
  | {
      start: string;
      end: string;
      state: "unavailable";
      observedSamples: number;
      availableSamples: 0;
      minimum: null;
      average: null;
      maximum: null;
    }
  | {
      start: string;
      end: string;
      state: "gap";
      observedSamples: 0;
      availableSamples: 0;
      minimum: null;
      average: null;
      maximum: null;
    };

export interface CPUHistorySeries {
  resource: ResourceRef;
  metric: CPUMetric;
  coreIndex: number | null;
  unit: "percent" | "load";
  range: ResolvedRange;
  bucketDurationSeconds: number;
  points: CPUHistoryPoint[];
}

export type CPUHistoryResponse = APIResponse<CPUHistorySeries>;

export type MemoryCurrentResponse = APIResponse<MemorySnapshot>;
export type MemoryMetric =
  | "total"
  | "used"
  | "available"
  | "free"
  | "cached"
  | "buffered"
  | "usage"
  | "swap_total"
  | "swap_used"
  | "pressure_some_avg10"
  | "pressure_full_avg10"
  | "pressure_some_total"
  | "pressure_full_total";

export type MemoryHistoryPoint = CPUHistoryPoint;

export interface MemoryHistorySeries {
  resource: ResourceRef;
  metric: MemoryMetric;
  unit: "bytes" | "percent" | "microseconds";
  range: ResolvedRange;
  bucketDurationSeconds: number;
  points: MemoryHistoryPoint[];
}

export type MemoryHistoryResponse = APIResponse<MemoryHistorySeries>;

export type ChartPoint =
  | { timestamp: string; state: "observed"; measurement: Metric }
  | { timestamp: string; state: "gap"; measurement: null };

export interface ChartSeries {
  resource: ResourceRef;
  metric: string;
  range: ResolvedRange;
  points: ChartPoint[];
}

export interface MetricStatistics {
  minimum: Metric;
  average: Metric;
  maximum: Metric;
}

export interface Incident {
  id: string;
  severity: Exclude<HealthState, "healthy">;
  causeCode: CauseCode;
  resource: ResourceRef;
  startedAt: string;
  endedAt: string | null;
  active: boolean;
}

export interface ContainerStateEvent {
  id: string;
  container: ResourceRef;
  timestamp: string;
  state: ContainerState;
  health: ContainerHealth;
}

export interface DockerLogLine {
  sequence: number;
  stream: "stdout" | "stderr";
  timestamp: string | null;
  content: string;
}

export type DockerLogEvent =
  | { type: "line"; line: DockerLogLine; error: null }
  | { type: "error"; line: null; error: APIError }
  | { type: "end"; line: null; error: null };

export type ActionKind =
  | "docker.start"
  | "docker.stop"
  | "docker.restart"
  | "backup.create"
  | "backup.restore"
  | "export.create"
  | "audit.delete"
  | "configuration.reload";
export type ActionStatus = "pending" | "succeeded" | "failed" | "rejected";

export type ConfirmationRequest =
  | {
      action: "docker.start" | "docker.stop" | "docker.restart";
      target: ResourceRef & { kind: "container" };
    }
  | {
      action: "backup.create" | "backup.restore";
      target: ResourceRef & { kind: "backup" };
    }
  | {
      action: "audit.delete";
      target: ResourceRef & { kind: "audit_database" };
    };

export type ConfirmationIntent = ConfirmationRequest & {
  id: string;
  expiresAt: string;
};

export type ExecuteActionRequest = ConfirmationRequest & {
  confirmationId: string;
};

export type ActionResult = ConfirmationRequest & {
  status: ActionStatus;
  requestedAt: string;
  completedAt: string | null;
  error: APIError | null;
};

export type ExportFormat = "csv" | "json";
export type ExportDataset =
  | "cpu"
  | "memory"
  | "filesystems"
  | "container_usage"
  | "container_events"
  | "incidents"
  | "audit";

export interface ExportRequest {
  format: ExportFormat;
  datasets: ExportDataset[];
  range:
    | {
        preset: "last_1h" | "last_24h" | "last_7d" | "last_14d";
        start: null;
        end: null;
      }
    | CustomRangeSelection;
}

export interface ExportJob {
  id: string;
  status: ActionStatus;
  format: ExportFormat;
  datasets: ExportDataset[];
  range: ResolvedRange;
  requestedAt: string;
  completedAt: string | null;
  downloadUrl: string | null;
  error: APIError | null;
}

export interface AuditEntry {
  id: string;
  requestedAt: string;
  completedAt: string | null;
  sourceIp: string;
  action: ActionKind;
  target: ResourceRef | null;
  parameters: Record<string, string>;
  outcome: "succeeded" | "failed" | "rejected";
  errorCode: ErrorCode | null;
  errorDetail: string | null;
}

export type AuditDeleteRequest =
  | { scope: "selected"; ids: string[]; range: null }
  | { scope: "range"; ids: []; range: CustomRangeSelection }
  | { scope: "all"; ids: []; range: null };

export interface BackupRecord {
  resource: ResourceRef;
  kind: "scheduled" | "manual" | "safety";
  status: "pending" | "available" | "invalid" | "failed";
  createdAt: string;
  sizeBytes: number;
  formatVersion: number;
  checksum: string;
  errorCode: ErrorCode | null;
}

export interface RecoveryStatus {
  mode: "normal" | "restore_recommended" | "maintenance";
  historyAvailability:
    | { state: "available"; reasonCode: null }
    | { state: "unavailable"; reasonCode: UnavailabilityReason };
  recommendedBackup: ResourceRef | null;
  reasonCode: ErrorCode | null;
}

export class ContractValidationError extends Error {
  readonly problems: string[];

  constructor(problems: string[]) {
    super("The API response does not match contract v1.");
    this.name = "ContractValidationError";
    this.problems = problems;
  }
}

export function parseMonitoringResponse(value: unknown): MonitoringResponse {
  const problems = validateMonitoringResponse(value);
  if (problems.length > 0) {
    throw new ContractValidationError(problems);
  }
  return value as MonitoringResponse;
}

export function parseCPUCurrentResponse(value: unknown): CPUCurrentResponse {
  const problems: string[] = [];
  const data = validateResponseEnvelope(value, problems);
  if (data !== undefined) {
    validateCPUSnapshot(data, "data", problems);
  }
  if (problems.length > 0) {
    throw new ContractValidationError(problems);
  }
  return value as CPUCurrentResponse;
}

export function parseCPUHistoryResponse(value: unknown): CPUHistoryResponse {
  const problems: string[] = [];
  const data = validateResponseEnvelope(value, problems);
  if (data !== undefined) {
    validateCPUHistory(data, "data", problems);
  }
  if (problems.length > 0) {
    throw new ContractValidationError(problems);
  }
  return value as CPUHistoryResponse;
}

export function parseMemoryCurrentResponse(
  value: unknown,
): MemoryCurrentResponse {
  const problems: string[] = [];
  const data = validateResponseEnvelope(value, problems);
  if (data !== undefined) {
    validateMemorySnapshot(data, "data", problems);
  }
  if (problems.length > 0) {
    throw new ContractValidationError(problems);
  }
  return value as MemoryCurrentResponse;
}

export function parseMemoryHistoryResponse(
  value: unknown,
): MemoryHistoryResponse {
  const problems: string[] = [];
  const data = validateResponseEnvelope(value, problems);
  if (data !== undefined) {
    validateMemoryHistory(data, "data", problems);
  }
  if (problems.length > 0) {
    throw new ContractValidationError(problems);
  }
  return value as MemoryHistoryResponse;
}

export function parseOperationalStatusResponse(
  value: unknown,
): OperationalStatusResponse {
  const problems: string[] = [];
  const data = validateResponseEnvelope(value, problems);
  if (data !== undefined) {
    validateOperationalStatus(data, "data", problems);
  }
  if (problems.length > 0) {
    throw new ContractValidationError(problems);
  }
  return value as OperationalStatusResponse;
}

export function validateMonitoringResponse(value: unknown): string[] {
  const problems: string[] = [];
  if (!isRecord(value)) {
    return ["response must be an object"];
  }
  if (value.apiVersion !== API_VERSION) {
    problems.push("apiVersion must be v1");
  }
  validateNonEmptyString(value.requestId, "requestId", problems);
  validateUTC(value.generatedAt, "generatedAt", problems);

  const hasData = value.data !== null && value.data !== undefined;
  const hasError = value.error !== null && value.error !== undefined;
  if (hasData === hasError) {
    problems.push("exactly one of data or error must be present");
    return problems;
  }
  if (!hasData) {
    validateAPIError(value.error, "error", problems);
    return problems;
  }
  if (!isRecord(value.data)) {
    problems.push("data must be an object");
    return problems;
  }

  validateFreshness(value.data.freshness, "data.freshness", problems);
  validateHealth(value.data.health, "data.health", problems);
  validateSnapshotResource(value.data.cpu, "data.cpu", "cpu", problems);
  validateSnapshotResource(
    value.data.memory,
    "data.memory",
    "memory",
    problems,
  );
  validateSnapshotResource(
    value.data.docker,
    "data.docker",
    "docker",
    problems,
  );
  validateConfiguration(
    value.data.configuration,
    "data.configuration",
    problems,
  );

  if (!Array.isArray(value.data.filesystems)) {
    problems.push("data.filesystems must be an array");
  } else {
    value.data.filesystems.forEach((filesystem, index) => {
      validateSnapshotResource(
        filesystem,
        `data.filesystems[${index}]`,
        "filesystem",
        problems,
      );
    });
  }

  walkMetrics(value.data, "data", problems);
  return problems;
}

export function parseChartSeries(value: unknown): ChartSeries {
  const problems: string[] = [];
  if (!isRecord(value)) {
    throw new ContractValidationError(["chart must be an object"]);
  }
  validateResource(value.resource, "resource", undefined, problems);
  validateNonEmptyString(value.metric, "metric", problems);
  if (!isRecord(value.range)) {
    problems.push("range must be an object");
  } else {
    validateUTC(value.range.start, "range.start", problems);
    validateUTC(value.range.end, "range.end", problems);
  }
  if (!Array.isArray(value.points)) {
    problems.push("points must be an array");
  } else {
    value.points.forEach((point, index) => {
      const path = `points[${index}]`;
      if (!isRecord(point)) {
        problems.push(`${path} must be an object`);
        return;
      }
      validateUTC(point.timestamp, `${path}.timestamp`, problems);
      if (point.state === "gap") {
        if (point.measurement !== null) {
          problems.push(`${path}.gap cannot contain a measurement`);
        }
      } else if (point.state === "observed") {
        validateMetric(point.measurement, `${path}.measurement`, problems);
      } else {
        problems.push(`${path}.state is invalid`);
      }
    });
  }
  if (problems.length > 0) {
    throw new ContractValidationError(problems);
  }
  return value as unknown as ChartSeries;
}

function validateSnapshotResource(
  value: unknown,
  path: string,
  kind: ResourceKind,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateResource(value.resource, `${path}.resource`, kind, problems);
  validateFreshness(value.freshness, `${path}.freshness`, problems);
}

function validateCPUSnapshot(value: unknown, path: string, problems: string[]) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateResource(value.resource, `${path}.resource`, "cpu", problems);
  validateFreshness(value.freshness, `${path}.freshness`, problems);
  validateExpectedMetric(value.overall, `${path}.overall`, "percent", problems);
  validateNonNegativeInteger(
    value.logicalCpuCount,
    `${path}.logicalCpuCount`,
    problems,
  );
  if (!Array.isArray(value.cores)) {
    problems.push(`${path}.cores must be an array`);
  } else {
    const indexes = new Set<number>();
    value.cores.forEach((core, index) => {
      const corePath = `${path}.cores[${index}]`;
      if (!isRecord(core)) {
        problems.push(`${corePath} must be an object`);
        return;
      }
      validateNonNegativeInteger(core.index, `${corePath}.index`, problems);
      if (typeof core.index === "number") {
        if (indexes.has(core.index)) {
          problems.push(`${corePath}.index is duplicated`);
        }
        indexes.add(core.index);
      }
      validateExpectedMetric(
        core.usage,
        `${corePath}.usage`,
        "percent",
        problems,
      );
    });
    if (
      typeof value.logicalCpuCount === "number" &&
      value.cores.length > value.logicalCpuCount
    ) {
      problems.push(`${path}.cores exceeds logicalCpuCount`);
    }
  }
  if (!isRecord(value.load)) {
    problems.push(`${path}.load must be an object`);
  } else {
    validateExpectedMetric(
      value.load.oneMinute,
      `${path}.load.oneMinute`,
      "load",
      problems,
    );
    validateExpectedMetric(
      value.load.fiveMinutes,
      `${path}.load.fiveMinutes`,
      "load",
      problems,
    );
    validateExpectedMetric(
      value.load.fifteenMinutes,
      `${path}.load.fifteenMinutes`,
      "load",
      problems,
    );
  }
}

function validateCPUHistory(value: unknown, path: string, problems: string[]) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateResource(value.resource, `${path}.resource`, "cpu", problems);
  const metrics = new Set<CPUMetric>([
    "overall",
    "core",
    "load_1",
    "load_5",
    "load_15",
  ]);
  if (
    typeof value.metric !== "string" ||
    !metrics.has(value.metric as CPUMetric)
  ) {
    problems.push(`${path}.metric is invalid`);
  }
  if (value.metric === "core") {
    validateNonNegativeInteger(value.coreIndex, `${path}.coreIndex`, problems);
  } else if (value.coreIndex !== null) {
    problems.push(`${path}.coreIndex is valid only for the core metric`);
  }
  const expectedUnit =
    value.metric === "overall" || value.metric === "core" ? "percent" : "load";
  if (value.unit !== expectedUnit) {
    problems.push(`${path}.unit does not match metric`);
  }
  if (!isRecord(value.range)) {
    problems.push(`${path}.range must be an object`);
  } else {
    validateUTC(value.range.start, `${path}.range.start`, problems);
    validateUTC(value.range.end, `${path}.range.end`, problems);
  }
  validateNonNegativeInteger(
    value.bucketDurationSeconds,
    `${path}.bucketDurationSeconds`,
    problems,
  );
  if (
    typeof value.bucketDurationSeconds === "number" &&
    value.bucketDurationSeconds < 60
  ) {
    problems.push(`${path}.bucketDurationSeconds must be at least 60`);
  }
  if (!Array.isArray(value.points)) {
    problems.push(`${path}.points must be an array`);
    return;
  }
  if (value.points.length > 600) {
    problems.push(`${path}.points cannot exceed 600 entries`);
  }
  value.points.forEach((point, index) => {
    validateCPUHistoryPoint(
      point,
      `${path}.points[${index}]`,
      value.unit,
      problems,
    );
  });
}

function validateMemorySnapshot(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateResource(value.resource, `${path}.resource`, "memory", problems);
  validateFreshness(value.freshness, `${path}.freshness`, problems);
  for (const field of [
    "total",
    "used",
    "available",
    "free",
    "cached",
    "buffered",
  ]) {
    validateExpectedMetric(value[field], `${path}.${field}`, "bytes", problems);
  }
  validateExpectedMetric(value.usage, `${path}.usage`, "percent", problems);
  if (!isRecord(value.swap)) {
    problems.push(`${path}.swap must be an object`);
  } else {
    if (
      value.swap.configured !== null &&
      typeof value.swap.configured !== "boolean"
    ) {
      problems.push(`${path}.swap.configured must be boolean or null`);
    }
    for (const field of ["total", "used", "free"]) {
      validateExpectedMetric(
        value.swap[field],
        `${path}.swap.${field}`,
        "bytes",
        problems,
      );
    }
  }
  if (!isRecord(value.pressure)) {
    problems.push(`${path}.pressure must be an object`);
  } else {
    validatePressureWindow(
      value.pressure.some,
      `${path}.pressure.some`,
      problems,
    );
    validatePressureWindow(
      value.pressure.full,
      `${path}.pressure.full`,
      problems,
    );
  }
}

function validatePressureWindow(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateExpectedMetric(
    value.average10Seconds,
    `${path}.average10Seconds`,
    "percent",
    problems,
  );
  validateExpectedMetric(
    value.average60Seconds,
    `${path}.average60Seconds`,
    "percent",
    problems,
  );
  validateExpectedMetric(
    value.average300Seconds,
    `${path}.average300Seconds`,
    "percent",
    problems,
  );
  validateExpectedMetric(
    value.total,
    `${path}.total`,
    "microseconds",
    problems,
  );
}

function validateMemoryHistory(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateResource(value.resource, `${path}.resource`, "memory", problems);
  const units = new Map<MemoryMetric, Unit>([
    ["total", "bytes"],
    ["used", "bytes"],
    ["available", "bytes"],
    ["free", "bytes"],
    ["cached", "bytes"],
    ["buffered", "bytes"],
    ["usage", "percent"],
    ["swap_total", "bytes"],
    ["swap_used", "bytes"],
    ["pressure_some_avg10", "percent"],
    ["pressure_full_avg10", "percent"],
    ["pressure_some_total", "microseconds"],
    ["pressure_full_total", "microseconds"],
  ]);
  if (
    typeof value.metric !== "string" ||
    !units.has(value.metric as MemoryMetric)
  ) {
    problems.push(`${path}.metric is invalid`);
  } else if (value.unit !== units.get(value.metric as MemoryMetric)) {
    problems.push(`${path}.unit does not match metric`);
  }
  if (!isRecord(value.range)) {
    problems.push(`${path}.range must be an object`);
  } else {
    validateUTC(value.range.start, `${path}.range.start`, problems);
    validateUTC(value.range.end, `${path}.range.end`, problems);
  }
  validateNonNegativeInteger(
    value.bucketDurationSeconds,
    `${path}.bucketDurationSeconds`,
    problems,
  );
  if (
    typeof value.bucketDurationSeconds === "number" &&
    value.bucketDurationSeconds < 60
  ) {
    problems.push(`${path}.bucketDurationSeconds must be at least 60`);
  }
  if (!Array.isArray(value.points)) {
    problems.push(`${path}.points must be an array`);
    return;
  }
  if (value.points.length > 600) {
    problems.push(`${path}.points cannot exceed 600 entries`);
  }
  value.points.forEach((point, index) => {
    validateCPUHistoryPoint(
      point,
      `${path}.points[${index}]`,
      value.unit,
      problems,
    );
  });
}

function validateCPUHistoryPoint(
  value: unknown,
  path: string,
  unit: unknown,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateUTC(value.start, `${path}.start`, problems);
  validateUTC(value.end, `${path}.end`, problems);
  validateNonNegativeInteger(
    value.observedSamples,
    `${path}.observedSamples`,
    problems,
  );
  validateNonNegativeInteger(
    value.availableSamples,
    `${path}.availableSamples`,
    problems,
  );
  const noValues =
    value.minimum === null && value.average === null && value.maximum === null;
  if (value.state === "observed") {
    if (
      typeof value.observedSamples !== "number" ||
      value.observedSamples < 1 ||
      typeof value.availableSamples !== "number" ||
      value.availableSamples < 1 ||
      value.availableSamples > value.observedSamples
    ) {
      problems.push(`${path}.observed sample counts are inconsistent`);
    }
    validateHistorySummary(
      value.minimum,
      value.average,
      value.maximum,
      unit,
      path,
      problems,
    );
  } else if (value.state === "unavailable") {
    if (
      typeof value.observedSamples !== "number" ||
      value.observedSamples < 1 ||
      value.availableSamples !== 0 ||
      !noValues
    ) {
      problems.push(
        `${path}.unavailable bucket contains inconsistent evidence`,
      );
    }
  } else if (value.state === "gap") {
    if (
      value.observedSamples !== 0 ||
      value.availableSamples !== 0 ||
      !noValues
    ) {
      problems.push(`${path}.gap cannot contain evidence`);
    }
  } else {
    problems.push(`${path}.state is invalid`);
  }
}

function validateHistorySummary(
  minimum: unknown,
  average: unknown,
  maximum: unknown,
  unit: unknown,
  path: string,
  problems: string[],
) {
  const values = [minimum, average, maximum];
  if (
    values.some(
      (value) =>
        typeof value !== "number" ||
        !Number.isFinite(value) ||
        value < 0 ||
        (unit === "percent" && value > 100),
    )
  ) {
    problems.push(`${path}.summary values are invalid`);
    return;
  }
  if (
    (minimum as number) > (average as number) ||
    (average as number) > (maximum as number)
  ) {
    problems.push(`${path}.summary values are not ordered`);
  }
}

function validateExpectedMetric(
  value: unknown,
  path: string,
  unit: Unit,
  problems: string[],
) {
  validateMetric(value, path, problems);
  if (isRecord(value) && value.unit !== unit) {
    problems.push(`${path}.unit must be ${unit}`);
  }
}

function validateResponseEnvelope(
  value: unknown,
  problems: string[],
): unknown | undefined {
  if (!isRecord(value)) {
    problems.push("response must be an object");
    return undefined;
  }
  if (value.apiVersion !== API_VERSION) {
    problems.push("apiVersion must be v1");
  }
  validateNonEmptyString(value.requestId, "requestId", problems);
  validateUTC(value.generatedAt, "generatedAt", problems);
  const hasData = value.data !== null && value.data !== undefined;
  const hasError = value.error !== null && value.error !== undefined;
  if (hasData === hasError) {
    problems.push("exactly one of data or error must be present");
    return undefined;
  }
  if (hasError) {
    validateAPIError(value.error, "error", problems);
    return undefined;
  }
  return value.data;
}

function validateOperationalStatus(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  if (
    !new Set(["ok", "degraded", "maintenance", "not_ready"]).has(
      String(value.state),
    )
  ) {
    problems.push(`${path}.state is invalid`);
  }
  validateNonNegativeInteger(
    value.uptimeSeconds,
    `${path}.uptimeSeconds`,
    problems,
  );
  if (typeof value.maintenance !== "boolean") {
    problems.push(`${path}.maintenance must be boolean`);
  }
  if (
    !new Set(["valid", "using_previous", "unavailable"]).has(
      String(value.configurationState),
    )
  ) {
    problems.push(`${path}.configurationState is invalid`);
  }
  validateOperationalDatabase(
    value.historyDatabase,
    `${path}.historyDatabase`,
    problems,
  );
  validateOperationalDatabase(
    value.auditDatabase,
    `${path}.auditDatabase`,
    problems,
  );
  validateOperationalCollection(
    value.collection,
    `${path}.collection`,
    problems,
  );
  if (
    !new Set(["available", "unavailable", "not_implemented"]).has(
      String(value.backupState),
    )
  ) {
    problems.push(`${path}.backupState is invalid`);
  }
  if (
    !new Set(["available", "unavailable", "not_checked"]).has(
      String(value.dockerConnectivity),
    )
  ) {
    problems.push(`${path}.dockerConnectivity is invalid`);
  }

  const requiredUnavailable =
    value.configurationState === "unavailable" ||
    (isRecord(value.historyDatabase) &&
      value.historyDatabase.state === "unavailable");
  if (typeof value.maintenance === "boolean") {
    let expectedState: OperationalState = "ok";
    if (value.maintenance) {
      expectedState = "maintenance";
    } else if (requiredUnavailable) {
      expectedState = "not_ready";
    } else if (
      value.configurationState !== "valid" ||
      (isRecord(value.auditDatabase) &&
        value.auditDatabase.state !== "available") ||
      (isRecord(value.collection) && value.collection.state !== "available") ||
      value.backupState !== "available" ||
      value.dockerConnectivity !== "available"
    ) {
      expectedState = "degraded";
    }
    if (value.state !== expectedState) {
      problems.push(
        `${path}.state does not match dependency state ${expectedState}`,
      );
    }
  }
}

function validateOperationalDatabase(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  if (value.state !== "available" && value.state !== "unavailable") {
    problems.push(`${path}.state is invalid`);
  }
  if (value.sizeBytes !== null) {
    validateNonNegativeInteger(value.sizeBytes, `${path}.sizeBytes`, problems);
  }
  if (value.state === "unavailable" && value.sizeBytes !== null) {
    problems.push(`${path}.unavailable cannot report a size`);
  }
}

function validateOperationalCollection(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  if (
    !new Set(["available", "unavailable", "not_started"]).has(
      String(value.state),
    )
  ) {
    problems.push(`${path}.state is invalid`);
  }
  if (typeof value.inProgress !== "boolean") {
    problems.push(`${path}.inProgress must be boolean`);
  }
  if (value.lastRun !== null) {
    if (!isRecord(value.lastRun)) {
      problems.push(`${path}.lastRun must be an object or null`);
    } else {
      validateUTC(
        value.lastRun.startedAt,
        `${path}.lastRun.startedAt`,
        problems,
      );
      validateUTC(
        value.lastRun.finishedAt,
        `${path}.lastRun.finishedAt`,
        problems,
      );
      validateNonNegativeInteger(
        value.lastRun.durationMs,
        `${path}.lastRun.durationMs`,
        problems,
      );
      if (
        typeof value.lastRun.startedAt === "string" &&
        typeof value.lastRun.finishedAt === "string" &&
        typeof value.lastRun.durationMs === "number"
      ) {
        const expectedDuration =
          Date.parse(value.lastRun.finishedAt) -
          Date.parse(value.lastRun.startedAt);
        if (
          Number.isFinite(expectedDuration) &&
          value.lastRun.durationMs !== expectedDuration
        ) {
          problems.push(`${path}.lastRun.durationMs must match timestamps`);
        }
      }
      if (
        !new Set(["succeeded", "partial", "failed"]).has(
          String(value.lastRun.status),
        )
      ) {
        problems.push(`${path}.lastRun.status is invalid`);
      }
    }
  }
  if (value.lastSuccessfulAt !== null) {
    validateUTC(value.lastSuccessfulAt, `${path}.lastSuccessfulAt`, problems);
  }
  if (
    value.state === "not_started" &&
    (value.inProgress !== false ||
      value.lastRun !== null ||
      value.lastSuccessfulAt !== null)
  ) {
    problems.push(`${path}.not_started cannot contain run state`);
  }
}

function validateResource(
  value: unknown,
  path: string,
  expectedKind: ResourceKind | undefined,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  if (expectedKind !== undefined && value.kind !== expectedKind) {
    problems.push(`${path}.kind must be ${expectedKind}`);
  }
  validateNonEmptyString(value.id, `${path}.id`, problems);
  validateNonEmptyString(value.displayName, `${path}.displayName`, problems);
}

function validateFreshness(value: unknown, path: string, problems: string[]) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  if (value.state === "fresh") {
    validateUTC(value.observedAt, `${path}.observedAt`, problems);
    validateUTC(value.lastSuccessfulAt, `${path}.lastSuccessfulAt`, problems);
  } else if (value.state === "stale") {
    if (value.observedAt !== null) {
      validateUTC(value.observedAt, `${path}.observedAt`, problems);
    }
    validateUTC(value.lastSuccessfulAt, `${path}.lastSuccessfulAt`, problems);
  } else if (value.state === "unavailable") {
    if (value.observedAt !== null || value.lastSuccessfulAt !== null) {
      problems.push(`${path}.unavailable cannot contain timestamps`);
    }
  } else {
    problems.push(`${path}.state is invalid`);
  }
}

function validateHealth(value: unknown, path: string, problems: string[]) {
  if (!isRecord(value) || !Array.isArray(value.states)) {
    problems.push(`${path} must contain a states array`);
    return;
  }
  const states = new Set(value.states);
  if (states.size === 0 || states.size !== value.states.length) {
    problems.push(`${path}.states must be non-empty and unique`);
  }
  if (states.has("healthy") && states.size !== 1) {
    problems.push(`${path}.healthy cannot be combined with problem states`);
  }
  value.states.forEach((state) => {
    if (typeof state !== "string" || !healthStates.has(state as HealthState)) {
      problems.push(`${path}.states contains an invalid state`);
    }
  });
  if (!Array.isArray(value.causes)) {
    problems.push(`${path}.causes must be an array`);
    return;
  }
  const causeStateCounts = new Map<string, number>();
  value.causes.forEach((cause, index) => {
    const causePath = `${path}.causes[${index}]`;
    if (!isRecord(cause)) {
      problems.push(`${causePath} must be an object`);
      return;
    }
    if (
      typeof cause.code !== "string" ||
      !causeCodes.has(cause.code as CauseCode)
    ) {
      problems.push(`${causePath}.code is invalid`);
    }
    if (
      typeof cause.state !== "string" ||
      cause.state === "healthy" ||
      !states.has(cause.state)
    ) {
      problems.push(`${causePath}.state is invalid or absent from states`);
    } else {
      causeStateCounts.set(
        cause.state,
        (causeStateCounts.get(cause.state) ?? 0) + 1,
      );
    }
    validateResource(
      cause.resource,
      `${causePath}.resource`,
      undefined,
      problems,
    );
    validateUTC(cause.startedAt, `${causePath}.startedAt`, problems);
    validateNonEmptyString(
      cause.messageKey,
      `${causePath}.messageKey`,
      problems,
    );
  });
  for (const state of states) {
    if (state !== "healthy" && !causeStateCounts.has(String(state))) {
      problems.push(`${path}.${String(state)} requires a matching cause`);
    }
  }
  if (
    typeof value.activeWarningCount !== "number" ||
    value.activeWarningCount !== (causeStateCounts.get("warning") ?? 0)
  ) {
    problems.push(`${path}.activeWarningCount does not match warning causes`);
  }
}

function validateConfiguration(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateResource(
    value.resource,
    `${path}.resource`,
    "configuration",
    problems,
  );
  if (value.state === "valid") {
    validateUTC(value.loadedAt, `${path}.loadedAt`, problems);
    if (value.errorCode !== null || value.errorMessageKey !== null) {
      problems.push(`${path}.valid cannot contain an error`);
    }
  } else if (
    value.state === "using_previous" ||
    value.state === "unavailable"
  ) {
    validateNonEmptyString(value.errorCode, `${path}.errorCode`, problems);
    validateNonEmptyString(
      value.errorMessageKey,
      `${path}.errorMessageKey`,
      problems,
    );
  } else {
    problems.push(`${path}.state is invalid`);
  }
}

function walkMetrics(value: unknown, path: string, problems: string[]) {
  if (Array.isArray(value)) {
    value.forEach((entry, index) =>
      walkMetrics(entry, `${path}[${index}]`, problems),
    );
    return;
  }
  if (!isRecord(value)) {
    return;
  }
  if ("availability" in value && "value" in value && "unit" in value) {
    validateMetric(value, path, problems);
    return;
  }
  Object.entries(value).forEach(([key, entry]) => {
    walkMetrics(entry, `${path}.${key}`, problems);
  });
}

function validateMetric(value: unknown, path: string, problems: string[]) {
  if (!isRecord(value)) {
    problems.push(`${path} must be a metric object`);
    return;
  }
  if (value.availability === "available") {
    if (typeof value.value !== "number" || !Number.isFinite(value.value)) {
      problems.push(`${path}.available requires a finite numeric value`);
    }
    if (value.reasonCode !== null) {
      problems.push(`${path}.available cannot contain reasonCode`);
    }
  } else if (value.availability === "unavailable") {
    if (value.value !== null) {
      problems.push(`${path}.unavailable cannot contain a value`);
    }
    if (
      typeof value.reasonCode !== "string" ||
      !unavailabilityReasons.has(value.reasonCode as UnavailabilityReason)
    ) {
      problems.push(`${path}.reasonCode is invalid`);
    }
  } else {
    problems.push(`${path}.availability is invalid`);
  }
  if (typeof value.unit !== "string" || !units.has(value.unit as Unit)) {
    problems.push(`${path}.unit is invalid`);
  }
}

function validateAPIError(value: unknown, path: string, problems: string[]) {
  if (!isRecord(value)) {
    problems.push(`${path} must be an object`);
    return;
  }
  validateNonEmptyString(value.code, `${path}.code`, problems);
  validateNonEmptyString(value.messageKey, `${path}.messageKey`, problems);
  validateNonEmptyString(
    value.fallbackMessage,
    `${path}.fallbackMessage`,
    problems,
  );
}

function validateUTC(value: unknown, path: string, problems: string[]) {
  if (
    typeof value !== "string" ||
    !/(?:Z|[+-]00:00)$/.test(value) ||
    Number.isNaN(Date.parse(value))
  ) {
    problems.push(`${path} must be an RFC3339 UTC timestamp`);
  }
}

function validateNonEmptyString(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (typeof value !== "string" || value.trim() === "") {
    problems.push(`${path} must be a non-empty string`);
  }
}

function validateNonNegativeInteger(
  value: unknown,
  path: string,
  problems: string[],
) {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < 0) {
    problems.push(`${path} must be a non-negative safe integer`);
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
