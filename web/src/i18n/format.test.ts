import { describe, expect, it } from "vitest";

import { DISPLAY_TIMEZONE, formatDashboardTime } from "./format";

describe("formatDashboardTime", () => {
  it("formats supported languages in the approved timezone", () => {
    const timestamp = "2026-08-05T12:00:00Z";

    expect(DISPLAY_TIMEZONE).toBe("Africa/Tunis");
    expect(formatDashboardTime(timestamp, "en")).toContain("13:00:00");
    expect(formatDashboardTime(timestamp, "fr")).toContain("13:00:00");
  });

  it("rejects invalid timestamps", () => {
    expect(() => formatDashboardTime("invalid", "en")).toThrow(
      "dashboard timestamp is invalid",
    );
  });
});
