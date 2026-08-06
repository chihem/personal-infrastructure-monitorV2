import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { describe, expect, it } from "vitest";

import {
  ContractValidationError,
  parseChartSeries,
  parseCPUCurrentResponse,
  parseCPUHistoryResponse,
  parseMonitoringResponse,
  parseOperationalStatusResponse,
} from "./contracts";

describe("CPU endpoint contracts", () => {
  const now = "2026-08-05T12:00:00Z";
  const metric = (value: number, unit: "percent" | "load") => ({
    availability: "available" as const,
    value,
    unit,
    reasonCode: null,
  });

  it("parses dynamic current CPU data", () => {
    const response: Record<string, any> = {
      apiVersion: "v1",
      requestId: "request-001",
      generatedAt: now,
      data: {
        resource: { kind: "cpu", id: "host-cpu", displayName: "Overall CPU" },
        freshness: { state: "fresh", observedAt: now, lastSuccessfulAt: now },
        overall: metric(40, "percent"),
        cores: [
          { index: 0, usage: metric(30, "percent") },
          { index: 5, usage: metric(50, "percent") },
        ],
        load: {
          oneMinute: metric(0.5, "load"),
          fiveMinutes: metric(0.4, "load"),
          fifteenMinutes: metric(0.3, "load"),
        },
        logicalCpuCount: 2,
      },
      error: null,
    };
    expect(parseCPUCurrentResponse(response).data?.cores).toHaveLength(2);
    response.data.cores[1].index = 0;
    expect(() => parseCPUCurrentResponse(response)).toThrow(
      ContractValidationError,
    );
  });

  it("keeps unavailable history separate from gaps", () => {
    const response: Record<string, any> = {
      apiVersion: "v1",
      requestId: "request-001",
      generatedAt: now,
      data: {
        resource: { kind: "cpu", id: "host-cpu", displayName: "Overall CPU" },
        metric: "overall",
        coreIndex: null,
        unit: "percent",
        range: { preset: "last_5m", start: "2026-08-05T11:55:00Z", end: now },
        bucketDurationSeconds: 60,
        points: [
          {
            start: "2026-08-05T11:55:00Z",
            end: "2026-08-05T11:56:00Z",
            state: "unavailable",
            observedSamples: 1,
            availableSamples: 0,
            minimum: null,
            average: null,
            maximum: null,
          },
          {
            start: "2026-08-05T11:56:00Z",
            end: "2026-08-05T11:57:00Z",
            state: "gap",
            observedSamples: 0,
            availableSamples: 0,
            minimum: null,
            average: null,
            maximum: null,
          },
        ],
      },
      error: null,
    };
    expect(
      parseCPUHistoryResponse(response).data?.points.map(
        (point) => point.state,
      ),
    ).toEqual(["unavailable", "gap"]);
    response.data.points[1].average = 0;
    expect(() => parseCPUHistoryResponse(response)).toThrow(
      ContractValidationError,
    );
  });
});

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
