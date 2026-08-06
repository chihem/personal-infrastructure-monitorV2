import { describe, expect, it } from "vitest";

import type { CPUHistoryPoint, Metric } from "../../api/contracts";
import {
  buildCPUHistoryPath,
  cpuHealth,
  formatTunisInput,
  summarizeCPUHistory,
  tunisLocalInputToUTC,
} from "./model";

describe("CPU page model", () => {
  it("calculates weighted history statistics without filling gaps", () => {
    const points: CPUHistoryPoint[] = [
      observed("2026-08-05T10:00:00Z", 20, 30, 40, 1),
      {
        start: "2026-08-05T10:01:00Z",
        end: "2026-08-05T10:02:00Z",
        state: "gap",
        observedSamples: 0,
        availableSamples: 0,
        minimum: null,
        average: null,
        maximum: null,
      },
      observed("2026-08-05T10:02:00Z", 50, 60, 70, 3),
      {
        start: "2026-08-05T10:03:00Z",
        end: "2026-08-05T10:04:00Z",
        state: "unavailable",
        observedSamples: 1,
        availableSamples: 0,
        minimum: null,
        average: null,
        maximum: null,
      },
    ];

    expect(summarizeCPUHistory(points)).toEqual({
      minimum: 20,
      average: 52.5,
      maximum: 70,
      observedBuckets: 2,
      unavailableBuckets: 1,
      gapBuckets: 1,
      availableSamples: 4,
    });
  });

  it("builds bounded preset, per-core, and custom API queries", () => {
    expect(
      buildCPUHistoryPath(
        { preset: "last_5m" },
        { metric: "core", coreIndex: 5 },
      ),
    ).toBe("/api/v1/cpu/history?metric=core&range=last_5m&core=5");
    expect(
      buildCPUHistoryPath(
        {
          preset: "custom",
          start: "2026-08-05T10:00:00.000Z",
          end: "2026-08-05T11:00:00.000Z",
        },
        { metric: "load_5", coreIndex: null },
      ),
    ).toContain("range=custom");
  });

  it("converts Africa/Tunis local inputs without using the browser timezone", () => {
    expect(tunisLocalInputToUTC("2026-08-05T12:30")).toBe(
      "2026-08-05T11:30:00.000Z",
    );
    expect(formatTunisInput(new Date("2026-08-05T11:30:00Z"))).toBe(
      "2026-08-05T12:30",
    );
  });

  it("classifies thresholds only for fresh available percentages", () => {
    expect(cpuHealth(metric(84.9), "fresh")).toBe("healthy");
    expect(cpuHealth(metric(85), "fresh")).toBe("warning");
    expect(cpuHealth(metric(95), "fresh")).toBe("critical");
    expect(cpuHealth(metric(95), "stale")).toBe("unknown");
  });
});

function metric(value: number): Metric<number> {
  return {
    availability: "available",
    value,
    unit: "percent",
    reasonCode: null,
  };
}

function observed(
  start: string,
  minimum: number,
  average: number,
  maximum: number,
  availableSamples: number,
): CPUHistoryPoint {
  return {
    start,
    end: new Date(new Date(start).getTime() + 60_000).toISOString(),
    state: "observed",
    observedSamples: availableSamples,
    availableSamples,
    minimum,
    average,
    maximum,
  };
}
