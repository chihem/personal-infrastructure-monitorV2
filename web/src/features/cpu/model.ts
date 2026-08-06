import type {
  CPUHistoryPoint,
  CPUHistorySeries,
  CPUMetric,
  HealthState,
  Metric,
  RangePreset,
} from "../../api/contracts";
import type { APIPath } from "../../api/client";

export const CPU_WARNING_PERCENT = 85;
export const CPU_CRITICAL_PERCENT = 95;
export const DASHBOARD_TIMEZONE = "Africa/Tunis";

export const CPU_RANGE_PRESETS = [
  "last_1m",
  "last_5m",
  "last_15m",
  "last_30m",
  "last_1h",
  "last_6h",
  "last_24h",
  "last_7d",
  "last_14d",
  "custom",
] as const satisfies readonly RangePreset[];

export type CPUHistorySelection =
  | { preset: Exclude<RangePreset, "custom"> }
  | { preset: "custom"; start: string; end: string };

export interface CPUSelectedMetric {
  metric: CPUMetric;
  coreIndex: number | null;
}

export interface CPUHistorySummary {
  minimum: number;
  average: number;
  maximum: number;
  observedBuckets: number;
  unavailableBuckets: number;
  gapBuckets: number;
  availableSamples: number;
}

export function cpuHealth(
  metric: Metric<number>,
  freshness: "fresh" | "stale" | "unavailable",
): HealthState {
  if (freshness !== "fresh" || metric.availability !== "available") {
    return "unknown";
  }
  if (metric.value >= CPU_CRITICAL_PERCENT) {
    return "critical";
  }
  if (metric.value >= CPU_WARNING_PERCENT) {
    return "warning";
  }
  return "healthy";
}

export function summarizeCPUHistory(
  points: readonly CPUHistoryPoint[],
): CPUHistorySummary | null {
  const observed = points.filter(
    (point): point is Extract<CPUHistoryPoint, { state: "observed" }> =>
      point.state === "observed",
  );
  if (observed.length === 0) {
    return null;
  }
  const availableSamples = observed.reduce(
    (total, point) => total + point.availableSamples,
    0,
  );
  const weightedTotal = observed.reduce(
    (total, point) => total + point.average * point.availableSamples,
    0,
  );
  return {
    minimum: Math.min(...observed.map((point) => point.minimum)),
    average: weightedTotal / availableSamples,
    maximum: Math.max(...observed.map((point) => point.maximum)),
    observedBuckets: observed.length,
    unavailableBuckets: points.filter((point) => point.state === "unavailable")
      .length,
    gapBuckets: points.filter((point) => point.state === "gap").length,
    availableSamples,
  };
}

export function buildCPUHistoryPath(
  selection: CPUHistorySelection,
  selectedMetric: CPUSelectedMetric,
): APIPath {
  const query = new URLSearchParams({
    metric: selectedMetric.metric,
    range: selection.preset,
  });
  if (selectedMetric.metric === "core" && selectedMetric.coreIndex !== null) {
    query.set("core", String(selectedMetric.coreIndex));
  }
  if (selection.preset === "custom") {
    query.set("start", selection.start);
    query.set("end", selection.end);
  }
  return `/api/v1/cpu/history?${query.toString()}`;
}

export function tunisLocalInputToUTC(value: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(value);
  if (match === null) {
    throw new TypeError("custom date must use YYYY-MM-DDTHH:mm");
  }
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const hour = Number(match[4]);
  const minute = Number(match[5]);
  let instant = Date.UTC(year, month - 1, day, hour, minute);
  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: DASHBOARD_TIMEZONE,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  });
  for (let attempt = 0; attempt < 2; attempt += 1) {
    const actual = zonedParts(formatter, new Date(instant));
    const difference =
      Date.UTC(year, month - 1, day, hour, minute) -
      Date.UTC(
        actual.year,
        actual.month - 1,
        actual.day,
        actual.hour,
        actual.minute,
      );
    instant += difference;
  }
  const finalParts = zonedParts(formatter, new Date(instant));
  if (
    finalParts.year !== year ||
    finalParts.month !== month ||
    finalParts.day !== day ||
    finalParts.hour !== hour ||
    finalParts.minute !== minute
  ) {
    throw new TypeError("custom date is not valid in Africa/Tunis");
  }
  return new Date(instant).toISOString();
}

export function formatTunisInput(value: Date): string {
  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: DASHBOARD_TIMEZONE,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  });
  const parts = zonedParts(formatter, value);
  return `${parts.year}-${pad(parts.month)}-${pad(parts.day)}T${pad(parts.hour)}:${pad(parts.minute)}`;
}

export function historySeriesLabel(series: CPUHistorySeries): string {
  return series.metric === "core" && series.coreIndex !== null
    ? `vCPU ${series.coreIndex}`
    : series.metric;
}

function zonedParts(formatter: Intl.DateTimeFormat, value: Date) {
  const parts = Object.fromEntries(
    formatter
      .formatToParts(value)
      .filter((part) => part.type !== "literal")
      .map((part) => [part.type, Number(part.value)]),
  );
  return {
    year: parts.year ?? 0,
    month: parts.month ?? 0,
    day: parts.day ?? 0,
    hour: parts.hour ?? 0,
    minute: parts.minute ?? 0,
  };
}

function pad(value: number): string {
  return String(value).padStart(2, "0");
}
