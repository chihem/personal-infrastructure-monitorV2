import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "../../i18n";
import { renderApp } from "../../test/renderApp";
import { renderWithI18n } from "../../test/renderWithI18n";
import { ContainerMemoryRanking } from "./MemoryPage";
import { formatTunisInput, tunisLocalInputToUTC } from "./model";

const chartMocks = vi.hoisted(() => ({
  setOption: vi.fn(),
  resize: vi.fn(),
  dispose: vi.fn(),
  init: vi.fn(),
}));

vi.mock("echarts/core", () => ({ use: vi.fn(), init: chartMocks.init }));
vi.mock("echarts/charts", () => ({ LineChart: {} }));
vi.mock("echarts/components", () => ({
  GridComponent: {},
  MarkLineComponent: {},
  TooltipComponent: {},
}));
vi.mock("echarts/renderers", () => ({ CanvasRenderer: {} }));

describe("memory page", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    await i18n.changeLanguage("en");
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
          url.includes("/memory/history")
            ? historyResponse()
            : currentResponse(),
        ),
      );
    });
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows distinct RAM values, thresholds, no-swap, pressure, and honest history", async () => {
    renderApp("/memory");

    expect(
      screen.getByRole("heading", { name: "Memory and pressure" }),
    ).toBeInTheDocument();
    expect((await screen.findAllByText("62.5%")).length).toBeGreaterThan(0);
    expect(screen.getByText("Warning at 85%")).toBeInTheDocument();
    expect(screen.getByText("Critical at 95%")).toBeInTheDocument();
    expect(screen.getByText("11 GiB")).toBeInTheDocument();
    expect(screen.getByText("4.1 GiB")).toBeInTheDocument();
    expect(screen.getByText("No swap configured")).toBeInTheDocument();
    expect(screen.getByText("Some tasks stalled")).toBeInTheDocument();
    expect(screen.getByText("All tasks stalled")).toBeInTheDocument();
    expect(
      await screen.findByTestId("memory-history-chart"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/1 gaps and 1 unavailable buckets/),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Container usage not connected yet"),
    ).toBeInTheDocument();

    const range = screen.getByRole("combobox", { name: "Time range" });
    expect(within(range).getAllByRole("option")).toHaveLength(10);
    expect(
      within(
        screen.getByRole("combobox", { name: "Measurement" }),
      ).getAllByRole("option"),
    ).toHaveLength(13);
    expect(chartMocks.setOption).toHaveBeenCalled();
  });

  it("keeps RAM visible while missing PSI is explicitly unavailable", async () => {
    fetchMock.mockImplementation((input: string | URL | Request) => {
      const url = String(input);
      return Promise.resolve(
        jsonResponse(
          url.includes("/memory/history")
            ? historyResponse()
            : currentResponse("fresh", true),
        ),
      );
    });
    renderApp("/memory");

    expect(await screen.findByText("11 GiB")).toBeInTheDocument();
    expect(
      screen.getByText("Pressure evidence unavailable"),
    ).toBeInTheDocument();
    expect(screen.queryByText("0.0%", { exact: true })).not.toBeInTheDocument();
  });

  it("requests selected metrics and valid custom ranges", async () => {
    const user = userEvent.setup();
    renderApp("/memory");
    await screen.findByText("11 GiB");

    await user.selectOptions(
      screen.getByRole("combobox", { name: "Measurement" }),
      "pressure_full_total",
    );
    await waitFor(() =>
      expect(
        fetchURLs(fetchMock).some((url) =>
          url.includes("metric=pressure_full_total"),
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

  it("supports the ranking empty state and a French Android-sized layout", async () => {
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      value: 390,
    });
    window.dispatchEvent(new Event("resize"));
    await i18n.changeLanguage("fr");
    renderApp("/memory");

    expect(
      screen.getByRole("heading", { name: "Mémoire et pression" }),
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("button", { name: "Actualiser les données" }),
    ).toBeVisible();
    expect(screen.getByRole("combobox", { name: "Période" })).toBeVisible();

    renderWithI18n(<ContainerMemoryRanking state="empty" />);
    expect(screen.getByText("Aucun conteneur à classer")).toBeInTheDocument();
  });
});

function currentResponse(
  freshness: "fresh" | "stale" = "fresh",
  missingPressure = false,
) {
  const observedAt = "2026-08-07T12:00:00Z";
  const pressureWindow = missingPressure
    ? {
        average10Seconds: unavailable("percent", "not_supported"),
        average60Seconds: unavailable("percent", "not_supported"),
        average300Seconds: unavailable("percent", "not_supported"),
        total: unavailable("microseconds", "not_supported"),
      }
    : {
        average10Seconds: metric(0.2, "percent"),
        average60Seconds: metric(0.1, "percent"),
        average300Seconds: metric(0.05, "percent"),
        total: metric(2_400_000, "microseconds"),
      };
  return envelope({
    resource: { kind: "memory", id: "host-memory", displayName: "Memory" },
    freshness: { state: freshness, observedAt, lastSuccessfulAt: observedAt },
    total: metric(11 * 1024 ** 3, "bytes"),
    used: metric(6.875 * 1024 ** 3, "bytes"),
    available: metric(4.125 * 1024 ** 3, "bytes"),
    free: metric(1.25 * 1024 ** 3, "bytes"),
    cached: metric(2.5 * 1024 ** 3, "bytes"),
    buffered: metric(384 * 1024 ** 2, "bytes"),
    usage: metric(62.5, "percent"),
    swap: {
      configured: false,
      total: unavailable("bytes", "not_configured"),
      used: unavailable("bytes", "not_configured"),
      free: unavailable("bytes", "not_configured"),
    },
    pressure: { some: pressureWindow, full: pressureWindow },
  });
}

function historyResponse() {
  return envelope({
    resource: { kind: "memory", id: "host-memory", displayName: "Memory" },
    metric: "usage",
    unit: "percent",
    range: {
      preset: "last_1h",
      start: "2026-08-07T11:00:00Z",
      end: "2026-08-07T12:00:00Z",
    },
    bucketDurationSeconds: 60,
    points: [
      observedPoint("2026-08-07T11:57:00Z", 55, 60, 62.5),
      {
        start: "2026-08-07T11:58:00Z",
        end: "2026-08-07T11:59:00Z",
        state: "unavailable",
        observedSamples: 1,
        availableSamples: 0,
        minimum: null,
        average: null,
        maximum: null,
      },
      {
        start: "2026-08-07T11:59:00Z",
        end: "2026-08-07T12:00:00Z",
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

function metric(value: number, unit: "bytes" | "percent" | "microseconds") {
  return { availability: "available", value, unit, reasonCode: null };
}

function unavailable(
  unit: "bytes" | "percent" | "microseconds",
  reasonCode: "not_supported" | "not_configured",
) {
  return { availability: "unavailable", value: null, unit, reasonCode };
}

function envelope(data: unknown) {
  return {
    apiVersion: "v1",
    requestId: "memory-page-test",
    generatedAt: "2026-08-07T12:00:00Z",
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
