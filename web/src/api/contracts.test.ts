import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { describe, expect, it } from "vitest";

import {
  ContractValidationError,
  parseChartSeries,
  parseMonitoringResponse,
} from "./contracts";

const fixtures = [
  "snapshot-complete.json",
  "snapshot-partial.json",
  "snapshot-stale.json",
  "snapshot-unavailable.json",
];

describe("monitoring contract examples", () => {
  for (const fixture of fixtures) {
    it(`parses ${fixture}`, () => {
      const parsed = parseMonitoringResponse(readFixture(fixture));

      expect(parsed.apiVersion).toBe("v1");
      expect(parsed.error).toBeNull();
    });
  }

  it("rejects an available metric without a value", () => {
    const fixture = readFixture("snapshot-complete.json");
    fixture.data.cpu.overall.value = null;

    expect(() => parseMonitoringResponse(fixture)).toThrow(
      ContractValidationError,
    );
  });

  it("rejects stale data without last-known evidence", () => {
    const fixture = readFixture("snapshot-stale.json");
    fixture.data.freshness.lastSuccessfulAt = null;

    expect(() => parseMonitoringResponse(fixture)).toThrow(
      ContractValidationError,
    );
  });

  it("rejects an incompatible API version", () => {
    const fixture = readFixture("snapshot-complete.json");
    fixture.apiVersion = "v2";

    expect(() => parseMonitoringResponse(fixture)).toThrow(
      ContractValidationError,
    );
  });
});

describe("chart contract example", () => {
  it("keeps collected-unavailable points distinct from gaps", () => {
    const chart = parseChartSeries(readFixture("chart-with-gap.json"));

    expect(chart.points[1]).toMatchObject({
      state: "observed",
      measurement: { availability: "unavailable" },
    });
    expect(chart.points[2]).toEqual({
      timestamp: "2026-08-04T11:57:00Z",
      state: "gap",
      measurement: null,
    });
  });

  it("rejects a gap containing a fabricated measurement", () => {
    const fixture = readFixture("chart-with-gap.json");
    fixture.points[2].measurement = {
      availability: "available",
      value: 0,
      unit: "percent",
      reasonCode: null,
    };

    expect(() => parseChartSeries(fixture)).toThrow(ContractValidationError);
  });
});

function readFixture(name: string): Record<string, any> {
  const path = resolve(
    process.cwd(),
    "..",
    "tests",
    "fixtures",
    "contracts",
    "v1",
    name,
  );
  return JSON.parse(readFileSync(path, "utf8")) as Record<string, any>;
}
