import { describe, expect, it } from "vitest";

import {
  createAppQueryClient,
  DASHBOARD_REFRESH_INTERVAL_MS,
} from "./queryClient";

describe("createAppQueryClient", () => {
  it("uses the approved one-minute foreground refresh policy", () => {
    const options = createAppQueryClient().getDefaultOptions();

    expect(options.queries?.refetchInterval).toBe(
      DASHBOARD_REFRESH_INTERVAL_MS,
    );
    expect(options.queries?.refetchIntervalInBackground).toBe(false);
    expect(options.queries?.staleTime).toBe(DASHBOARD_REFRESH_INTERVAL_MS);
    expect(options.queries?.retry).toBe(1);
    expect(options.mutations?.retry).toBe(false);
  });
});
