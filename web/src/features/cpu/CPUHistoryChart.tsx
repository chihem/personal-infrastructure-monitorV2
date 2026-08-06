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

import type { CPUHistorySeries } from "../../api/contracts";
import { formatDashboardTime } from "../../i18n/format";
import { normalizeLanguage } from "../../i18n";
import {
  CPU_CRITICAL_PERCENT,
  CPU_WARNING_PERCENT,
  summarizeCPUHistory,
} from "./model";
import styles from "./CPUPage.module.css";

use([
  LineChart,
  GridComponent,
  MarkLineComponent,
  TooltipComponent,
  CanvasRenderer,
]);

interface CPUHistoryChartProps {
  series: CPUHistorySeries;
}

export function CPUHistoryChart({ series }: CPUHistoryChartProps) {
  const { t, i18n } = useTranslation();
  const chartElement = useRef<HTMLDivElement>(null);
  const chart = useRef<ECharts | null>(null);
  const language = normalizeLanguage(i18n.language) ?? "en";
  const summary = useMemo(
    () => summarizeCPUHistory(series.points),
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
    const percent = series.unit === "percent";
    return {
      animation: false,
      backgroundColor: "transparent",
      grid: { top: 24, right: 24, bottom: 50, left: 52 },
      tooltip: {
        trigger: "axis",
        renderMode: "richText",
        valueFormatter: (value: unknown) =>
          typeof value === "number"
            ? formatMeasurement(value, series.unit, language)
            : t("cpu.history.noMeasurement"),
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
        ...(percent ? { max: 100 } : {}),
        axisLabel: {
          color: "#8f9caf",
          formatter: percent ? "{value}%" : "{value}",
        },
        splitLine: { lineStyle: { color: "rgba(71, 85, 105, 0.28)" } },
      },
      series: [
        historyLine(
          t("cpu.statistics.minimum"),
          values("minimum"),
          "#64748b",
          1,
        ),
        {
          ...historyLine(
            t("cpu.statistics.average"),
            values("average"),
            "#38bdf8",
            2.5,
          ),
          areaStyle: { color: "rgba(56, 189, 248, 0.10)" },
          markLine: percent
            ? {
                silent: true,
                symbol: "none",
                label: { color: "#dce7f3", formatter: "{b}" },
                data: [
                  {
                    name: t("cpu.thresholds.warningShort"),
                    yAxis: CPU_WARNING_PERCENT,
                    lineStyle: { color: "#fbbf24", type: "dashed" },
                  },
                  {
                    name: t("cpu.thresholds.criticalShort"),
                    yAxis: CPU_CRITICAL_PERCENT,
                    lineStyle: { color: "#fb7185", type: "dashed" },
                  },
                ],
              }
            : undefined,
        },
        historyLine(t("cpu.statistics.peak"), values("maximum"), "#7dd3fc", 1),
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
        data-testid="cpu-history-chart"
      />
      <figcaption className={styles.chartCaption}>
        {summary === null
          ? t("cpu.history.noObservedSummary")
          : t("cpu.history.accessibleSummary", {
              minimum: formatMeasurement(
                summary.minimum,
                series.unit,
                language,
              ),
              average: formatMeasurement(
                summary.average,
                series.unit,
                language,
              ),
              maximum: formatMeasurement(
                summary.maximum,
                series.unit,
                language,
              ),
              gaps: summary.gapBuckets,
              unavailable: summary.unavailableBuckets,
            })}
      </figcaption>
      <details className={styles.dataTableDisclosure}>
        <summary>{t("cpu.history.showTable")}</summary>
        <div className={styles.tableScroller}>
          <table className={styles.dataTable}>
            <thead>
              <tr>
                <th scope="col">{t("cpu.history.time")}</th>
                <th scope="col">{t("cpu.history.state")}</th>
                <th scope="col">{t("cpu.statistics.minimum")}</th>
                <th scope="col">{t("cpu.statistics.average")}</th>
                <th scope="col">{t("cpu.statistics.peak")}</th>
              </tr>
            </thead>
            <tbody>
              {series.points.map((point) => (
                <tr key={`${point.start}-${point.end}`}>
                  <td>{formatDashboardTime(point.start, language)}</td>
                  <td>{t(`cpu.history.states.${point.state}`)}</td>
                  {point.state === "observed" ? (
                    <>
                      <td>
                        {formatMeasurement(
                          point.minimum,
                          series.unit,
                          language,
                        )}
                      </td>
                      <td>
                        {formatMeasurement(
                          point.average,
                          series.unit,
                          language,
                        )}
                      </td>
                      <td>
                        {formatMeasurement(
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

export function formatMeasurement(
  value: number,
  unit: "percent" | "load",
  language: "en" | "fr",
): string {
  const formatted = new Intl.NumberFormat(
    language === "fr" ? "fr-FR" : "en-GB",
    {
      minimumFractionDigits: unit === "percent" ? 1 : 2,
      maximumFractionDigits: unit === "percent" ? 1 : 2,
    },
  ).format(value);
  return unit === "percent" ? `${formatted}%` : formatted;
}
