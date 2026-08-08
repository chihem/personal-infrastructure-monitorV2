import { describe, expect, it } from "vitest";

import type { MemoryHistoryPoint, Metric } from "../../api/contracts";
import {
  buildMemoryHistoryPath,
  formatTunisInput,
  memoryHealth,
  summarizeMemoryHistory,
  tunisLocalInputToUTC,
} from "./model";

describe("memory page model", () => {
  it("calculates weighted history statistics without filling missing evidence", () => {
    const points: MemoryHistoryPoint[] = [
      observed("2026-08-07T10:00:00Z", 20, 30, 40, 1),
      {
        start: "2026-08-07T10:01:00Z",
        end: "2026-08-07T10:02:00Z",
        state: "gap",
        observedSamples: 0,
        availableSamples: 0,
        minimum: null,
        average: null,
        maximum: null,
      },
      observed("2026-08-07T10:02:00Z", 50, 60, 70, 3),
      {
        start: "2026-08-07T10:03:00Z",
        end: "2026-08-07T10:04:00Z",
        state: "unavailable",
        observedSamples: 1,
        availableSamples: 0,
        minimum: null,
        average: null,
        maximum: null,
      },
    ];

    expect(summarizeMemoryHistory(points)).toEqual({
      minimum: 20,
      average: 52.5,
      maximum: 70,
      observedBuckets: 2,
      unavailableBuckets: 1,
      gapBuckets: 1,
      availableSamples: 4,
    });
  });

  it("builds allowlisted preset and custom memory history queries", () => {
    expect(buildMemoryHistoryPath({ preset: "last_5m" }, "usage")).toBe(
      "/api/v1/memory/history?metric=usage&range=last_5m",
    );
    expect(
      buildMemoryHistoryPath(
        {
          preset: "custom",
          start: "2026-08-07T10:00:00.000Z",
          end: "2026-08-07T11:00:00.000Z",
        },
        "pressure_full_total",
      ),
    ).toContain("metric=pressure_full_total&range=custom");
  });

  it("converts Africa/Tunis local inputs independently of browser timezone", () => {
    expect(tunisLocalInputToUTC("2026-08-07T12:30")).toBe(
      "2026-08-07T11:30:00.000Z",
    );
    expect(formatTunisInput(new Date("2026-08-07T11:30:00Z"))).toBe(
      "2026-08-07T12:30",
    );
  });

  it("classifies RAM thresholds only for fresh available percentages", () => {
    expect(memoryHealth(metric(84.9), "fresh")).toBe("healthy");
    expect(memoryHealth(metric(85), "fresh")).toBe("warning");
    expect(memoryHealth(metric(95), "fresh")).toBe("critical");
    expect(memoryHealth(metric(95), "stale")).toBe("unknown");
    expect(
      memoryHealth(
        {
          availability: "unavailable",
          value: null,
          unit: "percent",
          reasonCode: "not_collected",
        },
        "fresh",
      ),
    ).toBe("unknown");
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
): MemoryHistoryPoint {
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
