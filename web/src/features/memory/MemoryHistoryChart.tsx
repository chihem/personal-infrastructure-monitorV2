import { LineChart } from "echarts/charts";
import {
  GridComponent,
  MarkLineComponent,
  TooltipComponent,
} from "echarts/components";
import { init, use, type ECharts, type EChartsCoreOption } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";

import type { MemoryHistorySeries } from "../../api/contracts";
import { normalizeLanguage } from "../../i18n";
import { formatDashboardTime } from "../../i18n/format";
import {
  MEMORY_CRITICAL_PERCENT,
  MEMORY_WARNING_PERCENT,
  summarizeMemoryHistory,
} from "./model";
import styles from "./MemoryPage.module.css";

use([
  LineChart,
  GridComponent,
  MarkLineComponent,
  TooltipComponent,
  CanvasRenderer,
]);

export function MemoryHistoryChart({
  series,
}: {
  series: MemoryHistorySeries;
}) {
  const { t, i18n } = useTranslation();
  const chartElement = useRef<HTMLDivElement>(null);
  const chart = useRef<ECharts | null>(null);
  const language = normalizeLanguage(i18n.language) ?? "en";
  const summary = useMemo(
    () => summarizeMemoryHistory(series.points),
    [series.points],
  );
  const option = useMemo<EChartsCoreOption>(() => {
    const labels = series.points.map((point) =>
      formatDashboardTime(point.start, language),
    );
    const values = (field: "minimum" | "average" | "maximum") =>
      series.points.map((point) =>
        point.state === "observed" ? point[field] : null,
      );
    const usage = series.metric === "usage";
    return {
      animation: false,
      backgroundColor: "transparent",
      grid: { top: 24, right: 24, bottom: 50, left: 62 },
      tooltip: {
        trigger: "axis",
        renderMode: "richText",
        valueFormatter: (value: unknown) =>
          typeof value === "number"
            ? formatMemoryMeasurement(value, series.unit, language)
            : t("memory.history.noMeasurement"),
      },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: labels,
        axisLabel: { color: "#8f9caf", hideOverlap: true },
        axisLine: { lineStyle: { color: "#344356" } },
      },
      yAxis: {
        type: "value",
        min: 0,
        ...(series.unit === "percent" ? { max: 100 } : {}),
        axisLabel: {
          color: "#8f9caf",
          formatter: (value: number) =>
            formatAxisMeasurement(value, series.unit, language),
        },
        splitLine: { lineStyle: { color: "rgba(71, 85, 105, 0.28)" } },
      },
      series: [
        historyLine(
          t("memory.statistics.minimum"),
          values("minimum"),
          "#64748b",
          1,
        ),
        {
          ...historyLine(
            t("memory.statistics.average"),
            values("average"),
            "#a78bfa",
            2.5,
          ),
          areaStyle: { color: "rgba(167, 139, 250, 0.11)" },
          markLine: usage
            ? {
                silent: true,
                symbol: "none",
                label: { color: "#dce7f3", formatter: "{b}" },
                data: [
                  {
                    name: t("memory.thresholds.warningShort"),
                    yAxis: MEMORY_WARNING_PERCENT,
                    lineStyle: { color: "#fbbf24", type: "dashed" },
                  },
                  {
                    name: t("memory.thresholds.criticalShort"),
                    yAxis: MEMORY_CRITICAL_PERCENT,
                    lineStyle: { color: "#fb7185", type: "dashed" },
                  },
                ],
              }
            : undefined,
        },
        historyLine(
          t("memory.statistics.peak"),
          values("maximum"),
          "#c4b5fd",
          1,
        ),
      ],
    };
  }, [language, series, t]);

  useEffect(() => {
    if (chartElement.current === null) {
      return;
    }
    chart.current = init(chartElement.current, undefined, {
      renderer: "canvas",
    });
    const activeChart = chart.current;
    const resize = () => activeChart.resize();
    const observer =
      typeof ResizeObserver === "undefined" ? null : new ResizeObserver(resize);
    observer?.observe(chartElement.current);
    window.addEventListener("resize", resize);
    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", resize);
      activeChart.dispose();
      chart.current = null;
    };
  }, []);

  useEffect(() => {
    chart.current?.setOption(option, true);
  }, [option]);

  return (
    <figure className={styles.chartFigure}>
      <div
        ref={chartElement}
        className={styles.chartCanvas}
        aria-hidden="true"
        data-testid="memory-history-chart"
      />
      <figcaption className={styles.chartCaption}>
        {summary === null
          ? t("memory.history.noObservedSummary")
          : t("memory.history.accessibleSummary", {
              minimum: formatMemoryMeasurement(
                summary.minimum,
                series.unit,
                language,
              ),
              average: formatMemoryMeasurement(
                summary.average,
                series.unit,
                language,
              ),
              maximum: formatMemoryMeasurement(
                summary.maximum,
                series.unit,
                language,
              ),
              gaps: summary.gapBuckets,
              unavailable: summary.unavailableBuckets,
            })}
      </figcaption>
      <details className={styles.dataTableDisclosure}>
        <summary>{t("memory.history.showTable")}</summary>
        <div className={styles.tableScroller}>
          <table className={styles.dataTable}>
            <thead>
              <tr>
                <th scope="col">{t("memory.history.time")}</th>
                <th scope="col">{t("memory.history.state")}</th>
                <th scope="col">{t("memory.statistics.minimum")}</th>
                <th scope="col">{t("memory.statistics.average")}</th>
                <th scope="col">{t("memory.statistics.peak")}</th>
              </tr>
            </thead>
            <tbody>
              {series.points.map((point) => (
                <tr key={`${point.start}-${point.end}`}>
                  <td>{formatDashboardTime(point.start, language)}</td>
                  <td>{t(`memory.history.states.${point.state}`)}</td>
                  {point.state === "observed" ? (
                    <>
                      <td>
                        {formatMemoryMeasurement(
                          point.minimum,
                          series.unit,
                          language,
                        )}
                      </td>
                      <td>
                        {formatMemoryMeasurement(
                          point.average,
                          series.unit,
                          language,
                        )}
                      </td>
                      <td>
                        {formatMemoryMeasurement(
                          point.maximum,
                          series.unit,
                          language,
                        )}
                      </td>
                    </>
                  ) : (
                    <td colSpan={3}>—</td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </figure>
  );
}

function historyLine(
  name: string,
  data: Array<number | null>,
  color: string,
  width: number,
) {
  return {
    name,
    type: "line" as const,
    data,
    showSymbol: data.length <= 30,
    symbolSize: 6,
    connectNulls: false,
    lineStyle: { color, width },
    itemStyle: { color },
    emphasis: { focus: "series" as const },
  };
}

export function formatMemoryMeasurement(
  value: number,
  unit: "bytes" | "percent" | "microseconds",
  language: "en" | "fr",
): string {
  const locale = language === "fr" ? "fr-FR" : "en-GB";
  if (unit === "percent") {
    return `${new Intl.NumberFormat(locale, {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    }).format(value)}%`;
  }
  if (unit === "microseconds") {
    if (value >= 1_000_000) {
      return `${formatCompact(value / 1_000_000, locale)} s`;
    }
    if (value >= 1_000) {
      return `${formatCompact(value / 1_000, locale)} ms`;
    }
    return `${formatCompact(value, locale)} µs`;
  }
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let scaled = value;
  let index = 0;
  while (scaled >= 1024 && index < units.length - 1) {
    scaled /= 1024;
    index += 1;
  }
  return `${formatCompact(scaled, locale)} ${units[index]}`;
}

function formatAxisMeasurement(
  value: number,
  unit: "bytes" | "percent" | "microseconds",
  language: "en" | "fr",
) {
  if (unit === "percent") {
    return `${value}%`;
  }
  return formatMemoryMeasurement(value, unit, language);
}

function formatCompact(value: number, locale: string) {
  return new Intl.NumberFormat(locale, {
    minimumFractionDigits: 0,
    maximumFractionDigits: value < 10 ? 1 : 0,
  }).format(value);
}
