import { useQueryClient } from "@tanstack/react-query";
import {
  ChevronDown,
  Clock3,
  Cpu,
  Gauge,
  RefreshCw,
  TriangleAlert,
} from "lucide-react";
import { useMemo, useState, type CSSProperties, type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import type {
  CPUMetric,
  CPUSnapshot,
  Metric,
  RangePreset,
} from "../../api/contracts";
import { StatePanel } from "../../components/StatePanel";
import { StatusBadge } from "../../components/StatusBadge";
import { formatDashboardTime } from "../../i18n/format";
import { normalizeLanguage } from "../../i18n";
import { cpuQueryKeys, useCPUHistory, useCurrentCPU } from "./api";
import { CPUHistoryChart, formatMeasurement } from "./CPUHistoryChart";
import {
  CPU_CRITICAL_PERCENT,
  CPU_RANGE_PRESETS,
  CPU_WARNING_PERCENT,
  cpuHealth,
  formatTunisInput,
  summarizeCPUHistory,
  tunisLocalInputToUTC,
  type CPUHistorySelection,
  type CPUSelectedMetric,
} from "./model";
import styles from "./CPUPage.module.css";

const fixedMetrics: ReadonlyArray<{
  value: CPUMetric;
  labelKey:
    | "cpu.metrics.overall"
    | "cpu.metrics.load1"
    | "cpu.metrics.load5"
    | "cpu.metrics.load15";
}> = [
  { value: "overall", labelKey: "cpu.metrics.overall" },
  { value: "load_1", labelKey: "cpu.metrics.load1" },
  { value: "load_5", labelKey: "cpu.metrics.load5" },
  { value: "load_15", labelKey: "cpu.metrics.load15" },
];

export function CPUPage() {
  const { t, i18n } = useTranslation();
  const language = normalizeLanguage(i18n.language) ?? "en";
  const queryClient = useQueryClient();
  const current = useCurrentCPU();
  const [selectedMetric, setSelectedMetric] = useState<CPUSelectedMetric>({
    metric: "overall",
    coreIndex: null,
  });
  const [rangePreset, setRangePreset] = useState<RangePreset>("last_1h");
  const defaultCustom = useMemo(() => createDefaultCustomRange(), []);
  const [customStart, setCustomStart] = useState(defaultCustom.start);
  const [customEnd, setCustomEnd] = useState(defaultCustom.end);
  const [customRange, setCustomRange] = useState<CPUHistorySelection | null>(
    null,
  );
  const [customError, setCustomError] = useState<string | null>(null);
  const activeRange = useMemo<CPUHistorySelection | null>(
    () => (rangePreset === "custom" ? customRange : { preset: rangePreset }),
    [customRange, rangePreset],
  );
  const history = useCPUHistory(activeRange, selectedMetric);

  const refreshing = current.isFetching || history.isFetching;
  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: cpuQueryKeys.all });
  };

  const applyCustomRange = (event: FormEvent) => {
    event.preventDefault();
    try {
      const start = tunisLocalInputToUTC(customStart);
      const end = tunisLocalInputToUTC(customEnd);
      const startTime = new Date(start).getTime();
      const endTime = new Date(end).getTime();
      const now = Date.now();
      const retentionWindow = 14 * 24 * 60 * 60 * 1000;
      if (endTime <= startTime) {
        throw new TypeError("end must be later than start");
      }
      if (endTime > now) {
        throw new TypeError("end cannot be in the future");
      }
      if (
        startTime < now - retentionWindow ||
        endTime - startTime > retentionWindow
      ) {
        throw new TypeError("range cannot exceed retention");
      }
      setCustomError(null);
      setCustomRange({ preset: "custom", start, end });
    } catch {
      setCustomError(t("cpu.history.customInvalid"));
      setCustomRange(null);
    }
  };

  return (
    <div className={styles.page}>
      <header className={styles.pageHeader}>
        <div>
          <p className={styles.eyebrow}>{t("cpu.eyebrow")}</p>
          <h1>{t("cpu.title")}</h1>
          <p className={styles.introduction}>{t("cpu.introduction")}</p>
        </div>
        <button
          type="button"
          className={styles.refreshButton}
          onClick={() => void refresh()}
          disabled={refreshing}
        >
          <RefreshCw
            aria-hidden="true"
            className={refreshing ? styles.spinning : undefined}
            size={17}
          />
          {refreshing ? t("cpu.refreshing") : t("cpu.refresh")}
        </button>
      </header>

      {current.isPending ? <StatePanel variant="loading" /> : null}
      {current.isError ? (
        <StatePanel
          variant="error"
          title={t("cpu.currentErrorTitle")}
          message={t("cpu.currentErrorMessage")}
        />
      ) : null}
      {current.data?.freshness.state === "stale" ? (
        <StatePanel
          variant="stale"
          title={t("cpu.staleTitle")}
          message={t("cpu.staleMessage", {
            timestamp:
              current.data.freshness.lastSuccessfulAt === null
                ? t("cpu.notAvailable")
                : formatDashboardTime(
                    current.data.freshness.lastSuccessfulAt,
                    language,
                  ),
          })}
        />
      ) : null}
      {current.data?.freshness.state === "unavailable" ? (
        <StatePanel
          variant="unavailable"
          title={t("cpu.unavailableTitle")}
          message={t("cpu.unavailableMessage")}
        />
      ) : null}

      {current.data !== undefined ? (
        <CurrentCPU snapshot={current.data} />
      ) : null}

      <section
        className={styles.historySection}
        aria-labelledby="cpu-history-title"
      >
        <div className={styles.sectionHeading}>
          <div>
            <p className={styles.sectionEyebrow}>{t("cpu.history.eyebrow")}</p>
            <h2 id="cpu-history-title">{t("cpu.history.title")}</h2>
          </div>
          <span className={styles.retentionNote}>
            <Clock3 aria-hidden="true" size={16} />
            {t("cpu.history.retention")}
          </span>
        </div>

        <div className={styles.historyControls}>
          <label>
            <span>{t("cpu.history.metric")}</span>
            <select
              value={metricSelectionValue(selectedMetric)}
              onChange={(event) =>
                setSelectedMetric(parseMetricSelection(event.target.value))
              }
            >
              {fixedMetrics.map((metric) => (
                <option key={metric.value} value={metric.value}>
                  {t(metric.labelKey)}
                </option>
              ))}
              {(current.data?.cores ?? []).map((core) => (
                <option key={core.index} value={`core:${core.index}`}>
                  {t("cpu.metrics.core", { index: core.index })}
                </option>
              ))}
            </select>
          </label>
          <label>
            <span>{t("cpu.history.range")}</span>
            <select
              value={rangePreset}
              onChange={(event) => {
                const value = event.target.value as RangePreset;
                setRangePreset(value);
                setCustomError(null);
                if (value === "custom") {
                  setCustomRange(null);
                }
              }}
            >
              {CPU_RANGE_PRESETS.map((preset) => (
                <option key={preset} value={preset}>
                  {t(`cpu.ranges.${preset}`)}
                </option>
              ))}
            </select>
          </label>
        </div>

        {rangePreset === "custom" ? (
          <form className={styles.customRangeForm} onSubmit={applyCustomRange}>
            <label>
              <span>{t("cpu.history.start")}</span>
              <input
                type="datetime-local"
                value={customStart}
                onChange={(event) => setCustomStart(event.target.value)}
                required
              />
            </label>
            <label>
              <span>{t("cpu.history.end")}</span>
              <input
                type="datetime-local"
                value={customEnd}
                onChange={(event) => setCustomEnd(event.target.value)}
                required
              />
            </label>
            <button type="submit">{t("cpu.history.apply")}</button>
            <small>{t("cpu.history.customTimezone")}</small>
            {customError === null ? null : (
              <p className={styles.fieldError} role="alert">
                {customError}
              </p>
            )}
          </form>
        ) : null}

        {rangePreset === "custom" && activeRange === null ? (
          <StatePanel
            variant="empty"
            title={t("cpu.history.customTitle")}
            message={t("cpu.history.customMessage")}
          />
        ) : null}
        {history.isPending && activeRange !== null ? (
          <StatePanel
            variant="loading"
            title={t("cpu.history.loadingTitle")}
            message={t("cpu.history.loadingMessage")}
          />
        ) : null}
        {history.isError ? (
          <StatePanel
            variant="error"
            title={t("cpu.history.errorTitle")}
            message={t("cpu.history.errorMessage")}
          />
        ) : null}
        {history.data !== undefined ? (
          <HistoryContent series={history.data} />
        ) : null}
      </section>
    </div>
  );
}

function CurrentCPU({ snapshot }: { snapshot: CPUSnapshot }) {
  const { t, i18n } = useTranslation();
  const language = normalizeLanguage(i18n.language) ?? "en";
  const health = cpuHealth(snapshot.overall, snapshot.freshness.state);
  return (
    <>
      <section
        className={styles.currentGrid}
        aria-labelledby="current-cpu-title"
      >
        <article className={`${styles.card} ${styles.overallCard}`}>
          <div className={styles.cardHeading}>
            <span className={styles.cardIcon}>
              <Gauge aria-hidden="true" size={19} />
            </span>
            <div>
              <p>{t("cpu.current")}</p>
              <h2 id="current-cpu-title">{t("cpu.overall")}</h2>
            </div>
            <StatusBadge state={health} />
          </div>
          <div className={styles.overallReading}>
            <CPUUsageDial metric={snapshot.overall} />
            <div className={styles.thresholdLegend}>
              <span data-threshold="warning">
                <i />
                {t("cpu.thresholds.warning", {
                  value: CPU_WARNING_PERCENT,
                })}
              </span>
              <span data-threshold="critical">
                <i />
                {t("cpu.thresholds.critical", {
                  value: CPU_CRITICAL_PERCENT,
                })}
              </span>
              <small>
                {snapshot.freshness.observedAt === null
                  ? t("cpu.notCollected")
                  : t("cpu.observedAt", {
                      timestamp: formatDashboardTime(
                        snapshot.freshness.observedAt,
                        language,
                      ),
                    })}
              </small>
            </div>
          </div>
        </article>

        <article className={styles.card}>
          <div className={styles.cardHeading}>
            <span className={styles.cardIcon}>
              <Cpu aria-hidden="true" size={19} />
            </span>
            <div>
              <p>{t("cpu.load.subtitle")}</p>
              <h2>{t("cpu.load.title")}</h2>
            </div>
          </div>
          <div className={styles.loadGrid}>
            <MetricValue
              label={t("cpu.load.one")}
              metric={snapshot.load.oneMinute}
            />
            <MetricValue
              label={t("cpu.load.five")}
              metric={snapshot.load.fiveMinutes}
            />
            <MetricValue
              label={t("cpu.load.fifteen")}
              metric={snapshot.load.fifteenMinutes}
            />
          </div>
          <p className={styles.loadHelp}>{t("cpu.load.help")}</p>
        </article>
      </section>

      <details className={styles.coresDisclosure}>
        <summary>
          <span>
            <strong>{t("cpu.cores.title")}</strong>
            <small>
              {t("cpu.cores.count", { count: snapshot.logicalCpuCount })}
            </small>
          </span>
          <ChevronDown aria-hidden="true" size={19} />
        </summary>
        {snapshot.cores.length === 0 ? (
          <StatePanel
            variant="unavailable"
            title={t("cpu.cores.unavailableTitle")}
            message={t("cpu.cores.unavailableMessage")}
          />
        ) : (
          <div className={styles.coreGrid}>
            {snapshot.cores.map((core) => (
              <article className={styles.coreCard} key={core.index}>
                <div>
                  <span>{t("cpu.cores.logical")}</span>
                  <strong>vCPU {core.index}</strong>
                </div>
                <MetricText metric={core.usage} />
                <MetricBar metric={core.usage} />
              </article>
            ))}
          </div>
        )}
      </details>
    </>
  );
}

function HistoryContent({
  series,
}: {
  series: Parameters<typeof CPUHistoryChart>[0]["series"];
}) {
  const { t, i18n } = useTranslation();
  const language = normalizeLanguage(i18n.language) ?? "en";
  const summary = summarizeCPUHistory(series.points);
  return (
    <div className={styles.historyContent}>
      <div className={styles.historyMeta}>
        <span>
          {t("cpu.history.period", {
            start: formatDashboardTime(series.range.start, language),
            end: formatDashboardTime(series.range.end, language),
          })}
        </span>
        <span>
          {t("cpu.history.bucket", {
            seconds: series.bucketDurationSeconds,
          })}
        </span>
      </div>
      {summary === null ? (
        <StatePanel
          variant="unavailable"
          title={t("cpu.history.noObservedTitle")}
          message={t("cpu.history.noObservedMessage", {
            gaps: series.points.filter((point) => point.state === "gap").length,
          })}
        />
      ) : (
        <div className={styles.statisticsGrid}>
          <Statistic
            label={t("cpu.statistics.minimum")}
            value={formatMeasurement(summary.minimum, series.unit, language)}
          />
          <Statistic
            label={t("cpu.statistics.average")}
            value={formatMeasurement(summary.average, series.unit, language)}
          />
          <Statistic
            label={t("cpu.statistics.peak")}
            value={formatMeasurement(summary.maximum, series.unit, language)}
          />
          <Statistic
            label={t("cpu.statistics.gaps")}
            value={String(summary.gapBuckets)}
            warning={summary.gapBuckets > 0}
          />
        </div>
      )}
      <CPUHistoryChart series={series} />
    </div>
  );
}

function CPUUsageDial({ metric }: { metric: Metric<number> }) {
  const { t } = useTranslation();
  const value = metric.availability === "available" ? metric.value : 0;
  const style = {
    "--cpu-value": `${Math.min(100, Math.max(0, value)) * 3.6}deg`,
  } as CSSProperties;
  return (
    <div
      className={styles.usageDial}
      style={style}
      data-available={metric.availability}
    >
      <span>
        {metric.availability === "available"
          ? `${value.toFixed(1)}%`
          : t("cpu.notAvailable")}
      </span>
      <small>{t("cpu.usage")}</small>
    </div>
  );
}

function MetricValue({
  label,
  metric,
}: {
  label: string;
  metric: Metric<number>;
}) {
  return (
    <div className={styles.loadMetric}>
      <span>{label}</span>
      <MetricText metric={metric} />
    </div>
  );
}

function MetricText({ metric }: { metric: Metric<number> }) {
  const { t, i18n } = useTranslation();
  const language = normalizeLanguage(i18n.language) ?? "en";
  if (metric.availability === "unavailable") {
    return (
      <strong title={t(`cpu.reasons.${metric.reasonCode}`)}>
        {t("cpu.notAvailable")}
      </strong>
    );
  }
  return (
    <strong>
      {formatMeasurement(
        metric.value,
        metric.unit === "percent" ? "percent" : "load",
        language,
      )}
    </strong>
  );
}

function MetricBar({ metric }: { metric: Metric<number> }) {
  const value = metric.availability === "available" ? metric.value : 0;
  return (
    <div className={styles.metricBar} aria-hidden="true">
      <span style={{ width: `${Math.min(100, Math.max(0, value))}%` }} />
      <i data-threshold="warning" />
      <i data-threshold="critical" />
    </div>
  );
}

function Statistic({
  label,
  value,
  warning = false,
}: {
  label: string;
  value: string;
  warning?: boolean;
}) {
  return (
    <article className={styles.statistic} data-warning={warning}>
      {warning ? <TriangleAlert aria-hidden="true" size={16} /> : null}
      <span>{label}</span>
      <strong>{value}</strong>
    </article>
  );
}

function metricSelectionValue(selection: CPUSelectedMetric): string {
  return selection.metric === "core" && selection.coreIndex !== null
    ? `core:${selection.coreIndex}`
    : selection.metric;
}

function parseMetricSelection(value: string): CPUSelectedMetric {
  if (value.startsWith("core:")) {
    return { metric: "core", coreIndex: Number(value.slice(5)) };
  }
  return { metric: value as CPUMetric, coreIndex: null };
}

function createDefaultCustomRange() {
  const end = new Date();
  end.setSeconds(0, 0);
  const start = new Date(end.getTime() - 60 * 60 * 1000);
  return { start: formatTunisInput(start), end: formatTunisInput(end) };
}
