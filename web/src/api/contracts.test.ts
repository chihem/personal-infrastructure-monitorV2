import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { describe, expect, it } from "vitest";

import {
  ContractValidationError,
  parseChartSeries,
  parseMonitoringResponse,
  parseOperationalStatusResponse,
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

describe("operational status contract", () => {
  const response: Record<string, any> = {
    apiVersion: "v1",
    requestId: "request-001",
    generatedAt: "2026-08-05T12:00:00Z",
    data: {
      state: "degraded",
      uptimeSeconds: 15,
      maintenance: false,
      configurationState: "valid",
      historyDatabase: { state: "available", sizeBytes: 4096 },
      auditDatabase: { state: "unavailable", sizeBytes: null },
      collection: {
        state: "not_started",
        inProgress: false,
        lastRun: null,
        lastSuccessfulAt: null,
      },
      backupState: "not_implemented",
      dockerConnectivity: "not_checked",
    },
    error: null,
  };

  it("parses honest foundation placeholders", () => {
    expect(
      parseOperationalStatusResponse(structuredClone(response)).data,
    ).toMatchObject({
      state: "degraded",
      backupState: "not_implemented",
      dockerConnectivity: "not_checked",
    });
  });

  it("rejects a not-started collector with fabricated run state", () => {
    const invalid = structuredClone(response);
    invalid.data.collection.inProgress = true;

    expect(() => parseOperationalStatusResponse(invalid)).toThrow(
      ContractValidationError,
    );
  });

  it("rejects an unavailable database with a size", () => {
    const invalid = structuredClone(response);
    invalid.data.auditDatabase.sizeBytes = 1024;

    expect(() => parseOperationalStatusResponse(invalid)).toThrow(
      ContractValidationError,
    );
  });

  it("rejects a ready state when a required dependency is unavailable", () => {
    const invalid = structuredClone(response);
    invalid.data.historyDatabase = { state: "unavailable", sizeBytes: null };

    expect(() => parseOperationalStatusResponse(invalid)).toThrow(
      ContractValidationError,
    );
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
