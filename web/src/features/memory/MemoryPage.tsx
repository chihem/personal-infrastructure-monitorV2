import { useQueryClient } from "@tanstack/react-query";
import {
  Boxes,
  Clock3,
  Database,
  Gauge,
  MemoryStick,
  RefreshCw,
  TriangleAlert,
  Waves,
} from "lucide-react";
import { useMemo, useState, type CSSProperties, type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import type {
  MemoryHistorySeries,
  MemoryMetric,
  MemorySnapshot,
  Metric,
  PressureWindow,
  RangePreset,
} from "../../api/contracts";
import { StatePanel } from "../../components/StatePanel";
import { StatusBadge } from "../../components/StatusBadge";
import { normalizeLanguage } from "../../i18n";
import { formatDashboardTime } from "../../i18n/format";
import { memoryQueryKeys, useCurrentMemory, useMemoryHistory } from "./api";
import {
  formatMemoryMeasurement,
  MemoryHistoryChart,
} from "./MemoryHistoryChart";
import {
  MEMORY_CRITICAL_PERCENT,
  MEMORY_METRICS,
  MEMORY_RANGE_PRESETS,
  MEMORY_WARNING_PERCENT,
  formatTunisInput,
  memoryHealth,
  summarizeMemoryHistory,
  tunisLocalInputToUTC,
  type MemoryHistorySelection,
} from "./model";
import styles from "./MemoryPage.module.css";

export function MemoryPage() {
  const { t, i18n } = useTranslation();
  const language = normalizeLanguage(i18n.language) ?? "en";
  const queryClient = useQueryClient();
  const current = useCurrentMemory();
  const [selectedMetric, setSelectedMetric] = useState<MemoryMetric>("usage");
  const [rangePreset, setRangePreset] = useState<RangePreset>("last_1h");
  const defaultCustom = useMemo(() => createDefaultCustomRange(), []);
  const [customStart, setCustomStart] = useState(defaultCustom.start);
  const [customEnd, setCustomEnd] = useState(defaultCustom.end);
  const [customRange, setCustomRange] = useState<MemoryHistorySelection | null>(
    null,
  );
  const [customError, setCustomError] = useState<string | null>(null);
  const activeRange = useMemo<MemoryHistorySelection | null>(
    () => (rangePreset === "custom" ? customRange : { preset: rangePreset }),
    [customRange, rangePreset],
  );
  const history = useMemoryHistory(activeRange, selectedMetric);
  const refreshing = current.isFetching || history.isFetching;

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: memoryQueryKeys.all });
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
      setCustomError(t("memory.history.customInvalid"));
      setCustomRange(null);
    }
  };

  return (
    <div className={styles.page}>
      <header className={styles.pageHeader}>
        <div>
          <p className={styles.eyebrow}>{t("memory.eyebrow")}</p>
          <h1>{t("memory.title")}</h1>
          <p className={styles.introduction}>{t("memory.introduction")}</p>
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
          {refreshing ? t("memory.refreshing") : t("memory.refresh")}
        </button>
      </header>

      {current.isPending ? <StatePanel variant="loading" /> : null}
      {current.isError ? (
        <StatePanel
          variant="error"
          title={t("memory.currentErrorTitle")}
          message={t("memory.currentErrorMessage")}
        />
      ) : null}
      {current.data?.freshness.state === "stale" ? (
        <StatePanel
          variant="stale"
          title={t("memory.staleTitle")}
          message={t("memory.staleMessage", {
            timestamp:
              current.data.freshness.lastSuccessfulAt === null
                ? t("memory.notAvailable")
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
          title={t("memory.unavailableTitle")}
          message={t("memory.unavailableMessage")}
        />
      ) : null}

      {current.data === undefined ? null : (
        <CurrentMemory snapshot={current.data} />
      )}

      <MemoryHistorySection
        activeRange={activeRange}
        applyCustomRange={applyCustomRange}
        customEnd={customEnd}
        customError={customError}
        customStart={customStart}
        history={history}
        rangePreset={rangePreset}
        selectedMetric={selectedMetric}
        setCustomEnd={setCustomEnd}
        setCustomStart={setCustomStart}
        setRangePreset={(value) => {
          setRangePreset(value);
          setCustomError(null);
          if (value === "custom") {
            setCustomRange(null);
          }
        }}
        setSelectedMetric={setSelectedMetric}
      />

      <ContainerMemoryRanking state="unavailable" />
    </div>
  );
}

function CurrentMemory({ snapshot }: { snapshot: MemorySnapshot }) {
  const { t, i18n } = useTranslation();
  const language = normalizeLanguage(i18n.language) ?? "en";
  const health = memoryHealth(snapshot.usage, snapshot.freshness.state);
  return (
    <>
      <section
        className={styles.currentGrid}
        aria-labelledby="current-memory-title"
      >
        <article className={`${styles.card} ${styles.overallCard}`}>
          <div className={styles.cardHeading}>
            <span className={styles.cardIcon}>
              <Gauge aria-hidden="true" size={19} />
            </span>
            <div>
              <p>{t("memory.current")}</p>
              <h2 id="current-memory-title">{t("memory.overall")}</h2>
            </div>
            <StatusBadge state={health} />
          </div>
          <div className={styles.overallReading}>
            <MemoryUsageDial metric={snapshot.usage} />
            <div className={styles.thresholdLegend}>
              <span data-threshold="warning">
                <i />
                {t("memory.thresholds.warning", {
                  value: MEMORY_WARNING_PERCENT,
                })}
              </span>
              <span data-threshold="critical">
                <i />
                {t("memory.thresholds.critical", {
                  value: MEMORY_CRITICAL_PERCENT,
                })}
              </span>
              <small>
                {snapshot.freshness.observedAt === null
                  ? t("memory.notCollected")
                  : t("memory.observedAt", {
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
              <MemoryStick aria-hidden="true" size={19} />
            </span>
            <div>
              <p>{t("memory.breakdown.subtitle")}</p>
              <h2>{t("memory.breakdown.title")}</h2>
            </div>
          </div>
          <div className={styles.metricGrid}>
            <MetricCard
              label={t("memory.metrics.total")}
              metric={snapshot.total}
            />
            <MetricCard
              label={t("memory.metrics.used")}
              metric={snapshot.used}
            />
            <MetricCard
              label={t("memory.metrics.available")}
              metric={snapshot.available}
            />
            <MetricCard
              label={t("memory.metrics.free")}
              metric={snapshot.free}
            />
            <MetricCard
              label={t("memory.metrics.cached")}
              metric={snapshot.cached}
            />
            <MetricCard
              label={t("memory.metrics.buffered")}
              metric={snapshot.buffered}
            />
          </div>
          <p className={styles.helpText}>{t("memory.breakdown.help")}</p>
        </article>
      </section>

      <section className={styles.secondaryGrid}>
        <SwapCard snapshot={snapshot} />
        <PressureCard pressure={snapshot.pressure} />
      </section>
    </>
  );
}

function SwapCard({ snapshot }: { snapshot: MemorySnapshot }) {
  const { t } = useTranslation();
  return (
    <article className={styles.card} aria-labelledby="swap-title">
      <div className={styles.cardHeading}>
        <span className={styles.cardIcon}>
          <Database aria-hidden="true" size={19} />
        </span>
        <div>
          <p>{t("memory.swap.subtitle")}</p>
          <h2 id="swap-title">{t("memory.swap.title")}</h2>
        </div>
      </div>
      {snapshot.swap.configured === false ? (
        <StatePanel
          variant="empty"
          title={t("memory.swap.notConfiguredTitle")}
          message={t("memory.swap.notConfiguredMessage")}
        />
      ) : snapshot.swap.configured === null ? (
        <StatePanel
          variant="unavailable"
          title={t("memory.swap.unknownTitle")}
          message={t("memory.swap.unknownMessage")}
        />
      ) : (
        <div className={styles.metricGrid}>
          <MetricCard
            label={t("memory.metrics.swapTotal")}
            metric={snapshot.swap.total}
          />
          <MetricCard
            label={t("memory.metrics.swapUsed")}
            metric={snapshot.swap.used}
          />
          <MetricCard
            label={t("memory.metrics.swapFree")}
            metric={snapshot.swap.free}
          />
        </div>
      )}
    </article>
  );
}

function PressureCard({ pressure }: { pressure: MemorySnapshot["pressure"] }) {
  const { t } = useTranslation();
  const unavailable = [
    ...pressureMetrics(pressure.some),
    ...pressureMetrics(pressure.full),
  ].every((metric) => metric.availability === "unavailable");
  return (
    <article className={styles.card} aria-labelledby="pressure-title">
      <div className={styles.cardHeading}>
        <span className={styles.cardIcon}>
          <Waves aria-hidden="true" size={19} />
        </span>
        <div>
          <p>{t("memory.pressure.subtitle")}</p>
          <h2 id="pressure-title">{t("memory.pressure.title")}</h2>
        </div>
      </div>
      {unavailable ? (
        <StatePanel
          variant="unavailable"
          title={t("memory.pressure.unavailableTitle")}
          message={t("memory.pressure.unavailableMessage")}
        />
      ) : (
        <div className={styles.pressureGrid}>
          <PressureGroup
            label={t("memory.pressure.some")}
            window={pressure.some}
          />
          <PressureGroup
            label={t("memory.pressure.full")}
            window={pressure.full}
          />
        </div>
      )}
      <p className={styles.helpText}>{t("memory.pressure.help")}</p>
    </article>
  );
}

function PressureGroup({
  label,
  window,
}: {
  label: string;
  window: PressureWindow;
}) {
  const { t } = useTranslation();
  return (
    <div className={styles.pressureGroup}>
      <h3>{label}</h3>
      <MetricCard
        label={t("memory.pressure.avg10")}
        metric={window.average10Seconds}
      />
      <MetricCard
        label={t("memory.pressure.avg60")}
        metric={window.average60Seconds}
      />
      <MetricCard
        label={t("memory.pressure.avg300")}
        metric={window.average300Seconds}
      />
      <MetricCard label={t("memory.pressure.total")} metric={window.total} />
    </div>
  );
}

function pressureMetrics(window: PressureWindow) {
  return [
    window.average10Seconds,
    window.average60Seconds,
    window.average300Seconds,
    window.total,
  ];
}

interface HistorySectionProps {
  activeRange: MemoryHistorySelection | null;
  applyCustomRange: (event: FormEvent) => void;
  customEnd: string;
  customError: string | null;
  customStart: string;
  history: ReturnType<typeof useMemoryHistory>;
  rangePreset: RangePreset;
  selectedMetric: MemoryMetric;
  setCustomEnd: (value: string) => void;
  setCustomStart: (value: string) => void;
  setRangePreset: (value: RangePreset) => void;
  setSelectedMetric: (value: MemoryMetric) => void;
}

function MemoryHistorySection(props: HistorySectionProps) {
  const { t } = useTranslation();
  return (
    <section
      className={styles.historySection}
      aria-labelledby="memory-history-title"
    >
      <div className={styles.sectionHeading}>
        <div>
          <p className={styles.sectionEyebrow}>{t("memory.history.eyebrow")}</p>
          <h2 id="memory-history-title">{t("memory.history.title")}</h2>
        </div>
        <span className={styles.retentionNote}>
          <Clock3 aria-hidden="true" size={16} />
          {t("memory.history.retention")}
        </span>
      </div>
      <div className={styles.historyControls}>
        <label>
          <span>{t("memory.history.metric")}</span>
          <select
            value={props.selectedMetric}
            onChange={(event) =>
              props.setSelectedMetric(event.target.value as MemoryMetric)
            }
          >
            {MEMORY_METRICS.map((metric) => (
              <option key={metric} value={metric}>
                {t(`memory.metrics.${metric}`)}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>{t("memory.history.range")}</span>
          <select
            value={props.rangePreset}
            onChange={(event) =>
              props.setRangePreset(event.target.value as RangePreset)
            }
          >
            {MEMORY_RANGE_PRESETS.map((preset) => (
              <option key={preset} value={preset}>
                {t(`memory.ranges.${preset}`)}
              </option>
            ))}
          </select>
        </label>
      </div>

      {props.rangePreset === "custom" ? (
        <form
          className={styles.customRangeForm}
          onSubmit={props.applyCustomRange}
        >
          <label>
            <span>{t("memory.history.start")}</span>
            <input
              type="datetime-local"
              value={props.customStart}
              onChange={(event) => props.setCustomStart(event.target.value)}
              required
            />
          </label>
          <label>
            <span>{t("memory.history.end")}</span>
            <input
              type="datetime-local"
              value={props.customEnd}
              onChange={(event) => props.setCustomEnd(event.target.value)}
              required
            />
          </label>
          <button type="submit">{t("memory.history.apply")}</button>
          <small>{t("memory.history.customTimezone")}</small>
          {props.customError === null ? null : (
            <p className={styles.fieldError} role="alert">
              {props.customError}
            </p>
          )}
        </form>
      ) : null}

      {props.rangePreset === "custom" && props.activeRange === null ? (
        <StatePanel
          variant="empty"
          title={t("memory.history.customTitle")}
          message={t("memory.history.customMessage")}
        />
      ) : null}
      {props.history.isPending && props.activeRange !== null ? (
        <StatePanel
          variant="loading"
          title={t("memory.history.loadingTitle")}
          message={t("memory.history.loadingMessage")}
        />
      ) : null}
      {props.history.isError ? (
        <StatePanel
          variant="error"
          title={t("memory.history.errorTitle")}
          message={t("memory.history.errorMessage")}
        />
      ) : null}
      {props.history.data === undefined ? null : (
        <HistoryContent series={props.history.data} />
      )}
    </section>
  );
}

function HistoryContent({ series }: { series: MemoryHistorySeries }) {
  const { t, i18n } = useTranslation();
  const language = normalizeLanguage(i18n.language) ?? "en";
  const summary = summarizeMemoryHistory(series.points);
  return (
    <div className={styles.historyContent}>
      <div className={styles.historyMeta}>
        <span>
          {t("memory.history.period", {
            start: formatDashboardTime(series.range.start, language),
            end: formatDashboardTime(series.range.end, language),
          })}
        </span>
        <span>
          {t("memory.history.bucket", {
            seconds: series.bucketDurationSeconds,
          })}
        </span>
      </div>
      {summary === null ? (
        <StatePanel
          variant="unavailable"
          title={t("memory.history.noObservedTitle")}
          message={t("memory.history.noObservedMessage", {
            gaps: series.points.filter((point) => point.state === "gap").length,
          })}
        />
      ) : (
        <div className={styles.statisticsGrid}>
          <Statistic
            label={t("memory.statistics.minimum")}
            value={formatMemoryMeasurement(
              summary.minimum,
              series.unit,
              language,
            )}
          />
          <Statistic
            label={t("memory.statistics.average")}
            value={formatMemoryMeasurement(
              summary.average,
              series.unit,
              language,
            )}
          />
          <Statistic
            label={t("memory.statistics.peak")}
            value={formatMemoryMeasurement(
              summary.maximum,
              series.unit,
              language,
            )}
          />
          <Statistic
            label={t("memory.statistics.gaps")}
            value={String(summary.gapBuckets)}
            warning={summary.gapBuckets > 0}
          />
        </div>
      )}
      <MemoryHistoryChart series={series} />
    </div>
  );
}

function MemoryUsageDial({ metric }: { metric: Metric<number> }) {
  const { t } = useTranslation();
  const value = metric.availability === "available" ? metric.value : 0;
  const style = {
    "--memory-value": `${Math.min(100, Math.max(0, value)) * 3.6}deg`,
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
          : t("memory.notAvailable")}
      </span>
      <small>{t("memory.usage")}</small>
    </div>
  );
}

function MetricCard({
  label,
  metric,
}: {
  label: string;
  metric: Metric<number>;
}) {
  const { t, i18n } = useTranslation();
  const language = normalizeLanguage(i18n.language) ?? "en";
  return (
    <div className={styles.metricCard}>
      <span>{label}</span>
      {metric.availability === "available" ? (
        <strong>
          {formatMemoryMeasurement(
            metric.value,
            metric.unit as "bytes" | "percent" | "microseconds",
            language,
          )}
        </strong>
      ) : (
        <strong title={t(`memory.reasons.${metric.reasonCode}`)}>
          {t("memory.notAvailable")}
        </strong>
      )}
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

export function ContainerMemoryRanking({
  state,
}: {
  state: "unavailable" | "empty";
}) {
  const { t } = useTranslation();
  return (
    <section className={styles.rankingSection} aria-labelledby="ranking-title">
      <div className={styles.sectionHeading}>
        <div>
          <p className={styles.sectionEyebrow}>{t("memory.ranking.eyebrow")}</p>
          <h2 id="ranking-title">{t("memory.ranking.title")}</h2>
        </div>
        <Boxes aria-hidden="true" size={20} />
      </div>
      <StatePanel
        variant={state === "empty" ? "empty" : "unavailable"}
        title={t(`memory.ranking.${state}Title`)}
        message={t(`memory.ranking.${state}Message`)}
      />
    </section>
  );
}

function createDefaultCustomRange() {
  const end = new Date();
  end.setSeconds(0, 0);
  const start = new Date(end.getTime() - 60 * 60 * 1000);
  return { start: formatTunisInput(start), end: formatTunisInput(end) };
}
