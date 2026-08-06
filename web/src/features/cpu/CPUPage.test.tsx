import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "../../i18n";
import { renderApp } from "../../test/renderApp";
import { formatTunisInput, tunisLocalInputToUTC } from "./model";

const chartMocks = vi.hoisted(() => ({
  setOption: vi.fn(),
  resize: vi.fn(),
  dispose: vi.fn(),
  init: vi.fn(),
}));

vi.mock("echarts/core", () => ({
  use: vi.fn(),
  init: chartMocks.init,
}));
vi.mock("echarts/charts", () => ({ LineChart: {} }));
vi.mock("echarts/components", () => ({
  GridComponent: {},
  MarkLineComponent: {},
  TooltipComponent: {},
}));
vi.mock("echarts/renderers", () => ({ CanvasRenderer: {} }));

describe("CPU page", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    chartMocks.setOption.mockReset();
    chartMocks.resize.mockReset();
    chartMocks.dispose.mockReset();
    chartMocks.init.mockReset();
    chartMocks.init.mockReturnValue({
      setOption: chartMocks.setOption,
      resize: chartMocks.resize,
      dispose: chartMocks.dispose,
    });
    fetchMock = vi.fn((input: string | URL | Request) => {
      const url = String(input);
      return Promise.resolve(
        jsonResponse(
          url.includes("/cpu/history") ? historyResponse() : currentResponse(),
        ),
      );
    });
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows overall CPU first, thresholds, load, dynamic cores, and honest history", async () => {
    const user = userEvent.setup();
    renderApp("/cpu");

    expect(
      screen.getByRole("heading", { name: "Processor performance" }),
    ).toBeInTheDocument();
    expect((await screen.findAllByText("42.0%")).length).toBeGreaterThan(0);
    expect(screen.getByText("Warning at 85%")).toBeInTheDocument();
    expect(screen.getByText("Critical at 95%")).toBeInTheDocument();
    expect(screen.getByText("Load averages")).toBeInTheDocument();
    expect(await screen.findByTestId("cpu-history-chart")).toBeInTheDocument();
    expect(
      screen.getByText(/1 gaps and 1 unavailable buckets/),
    ).toBeInTheDocument();

    await user.click(screen.getByText("Logical vCPU details"));
    expect(screen.getAllByText("vCPU 0").length).toBeGreaterThan(0);
    expect(screen.getAllByText("vCPU 5").length).toBeGreaterThan(0);

    const range = screen.getByRole("combobox", { name: "Time range" });
    expect(within(range).getAllByRole("option")).toHaveLength(10);
    expect(chartMocks.setOption).toHaveBeenCalled();
  });

  it("requests per-core and valid custom history ranges", async () => {
    const user = userEvent.setup();
    renderApp("/cpu");
    await screen.findAllByText("42.0%");

    await user.selectOptions(
      screen.getByRole("combobox", { name: "Measurement" }),
      "core:5",
    );
    await waitFor(() =>
      expect(
        fetchURLs(fetchMock).some(
          (url) => url.includes("metric=core") && url.includes("core=5"),
        ),
      ).toBe(true),
    );

    await user.selectOptions(
      screen.getByRole("combobox", { name: "Time range" }),
      "custom",
    );
    const start = screen.getByLabelText("Start");
    const end = screen.getByLabelText("End");
    const endDate = new Date(Date.now() - 60 * 60 * 1000);
    const startDate = new Date(endDate.getTime() - 60 * 60 * 1000);
    const startInput = formatTunisInput(startDate);
    const endInput = formatTunisInput(endDate);
    await user.clear(start);
    await user.type(start, startInput);
    await user.clear(end);
    await user.type(end, endInput);
    await user.click(screen.getByRole("button", { name: "Apply range" }));

    const expectedStart = encodeURIComponent(tunisLocalInputToUTC(startInput));
    const expectedEnd = encodeURIComponent(tunisLocalInputToUTC(endInput));

    await waitFor(() =>
      expect(
        fetchURLs(fetchMock).some(
          (url) =>
            url.includes("range=custom") &&
            url.includes(`start=${expectedStart}`) &&
            url.includes(`end=${expectedEnd}`),
        ),
      ).toBe(true),
    );
  });

  it("retains values but clearly marks stale evidence", async () => {
    fetchMock.mockImplementation((input: string | URL | Request) => {
      const url = String(input);
      const body = url.includes("/cpu/history")
        ? historyResponse()
        : currentResponse("stale");
      return Promise.resolve(jsonResponse(body));
    });
    renderApp("/cpu");

    expect(
      await screen.findByText("CPU evidence is stale"),
    ).toBeInTheDocument();
    expect(screen.getAllByText("42.0%").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Unknown").length).toBeGreaterThan(0);
  });

  it("remains usable on Android width and switches completely to French", async () => {
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      value: 390,
    });
    window.dispatchEvent(new Event("resize"));
    await i18n.changeLanguage("fr");
    renderApp("/cpu");

    expect(
      screen.getByRole("heading", { name: "Performances du processeur" }),
    ).toBeInTheDocument();
    expect(
      (await screen.findAllByText("Processeur global")).length,
    ).toBeGreaterThan(0);
    await screen.findByTestId("cpu-history-chart");
    expect(
      screen.getByRole("button", { name: "Actualiser les données" }),
    ).toBeVisible();
    expect(screen.getByRole("combobox", { name: "Période" })).toBeVisible();
  });
});

function currentResponse(freshness: "fresh" | "stale" = "fresh") {
  const observedAt = "2026-08-05T12:00:00Z";
  return envelope({
    resource: { kind: "cpu", id: "host-cpu", displayName: "Overall CPU" },
    freshness: {
      state: freshness,
      observedAt,
      lastSuccessfulAt: observedAt,
    },
    overall: metric(42, "percent"),
    cores: [
      { index: 0, usage: metric(30, "percent") },
      { index: 5, usage: metric(54, "percent") },
    ],
    load: {
      oneMinute: metric(1.2, "load"),
      fiveMinutes: metric(0.9, "load"),
      fifteenMinutes: metric(0.7, "load"),
    },
    logicalCpuCount: 2,
  });
}

function historyResponse() {
  return envelope({
    resource: { kind: "cpu", id: "host-cpu", displayName: "Overall CPU" },
    metric: "overall",
    coreIndex: null,
    unit: "percent",
    range: {
      preset: "last_1h",
      start: "2026-08-05T11:00:00Z",
      end: "2026-08-05T12:00:00Z",
    },
    bucketDurationSeconds: 60,
    points: [
      observedPoint("2026-08-05T11:57:00Z", 20, 30, 42),
      {
        start: "2026-08-05T11:58:00Z",
        end: "2026-08-05T11:59:00Z",
        state: "unavailable",
        observedSamples: 1,
        availableSamples: 0,
        minimum: null,
        average: null,
        maximum: null,
      },
      {
        start: "2026-08-05T11:59:00Z",
        end: "2026-08-05T12:00:00Z",
        state: "gap",
        observedSamples: 0,
        availableSamples: 0,
        minimum: null,
        average: null,
        maximum: null,
      },
    ],
  });
}

function observedPoint(
  start: string,
  minimum: number,
  average: number,
  maximum: number,
) {
  return {
    start,
    end: new Date(new Date(start).getTime() + 60_000).toISOString(),
    state: "observed",
    observedSamples: 1,
    availableSamples: 1,
    minimum,
    average,
    maximum,
  };
}

function metric(value: number, unit: "percent" | "load") {
  return { availability: "available", value, unit, reasonCode: null };
}

function envelope(data: unknown) {
  return {
    apiVersion: "v1",
    requestId: "cpu-page-test",
    generatedAt: "2026-08-05T12:00:00Z",
    data,
    error: null,
  };
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function fetchURLs(fetchMock: ReturnType<typeof vi.fn>): string[] {
  return fetchMock.mock.calls.map((call) => String(call[0]));
}
