import { afterEach, describe, expect, it, vi } from "vitest";

import { APIRequestError, requestAPI, type APIPath } from "./client";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("requestAPI", () => {
  it("requests JSON from the same-origin versioned API", async () => {
    const fetchMock = vi.fn(async () =>
      Promise.resolve(
        new Response(JSON.stringify({ status: "ok" }), {
          headers: { "Content-Type": "application/json" },
          status: 200,
        }),
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      requestAPI<{ status: string }>("/api/v1/health"),
    ).resolves.toEqual({ status: "ok" });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/health",
      expect.objectContaining({
        cache: "no-store",
        credentials: "same-origin",
        redirect: "error",
      }),
    );
  });

  it("rejects paths outside the versioned API", async () => {
    await expect(requestAPI("https://example.com" as APIPath)).rejects.toThrow(
      "same-origin",
    );
  });

  it("returns a bounded status error without reflecting response content", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        Promise.resolve(
          new Response("private backend detail", {
            headers: { "Content-Type": "text/plain" },
            status: 503,
          }),
        ),
      ),
    );

    const result = requestAPI("/api/v1/health");
    await expect(result).rejects.toBeInstanceOf(APIRequestError);
    await expect(result).rejects.not.toThrow("private backend detail");
  });
});
