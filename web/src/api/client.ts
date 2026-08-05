const API_PREFIX = "/api/v1/";

export type APIPath = `${typeof API_PREFIX}${string}`;

export class APIRequestError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "APIRequestError";
    this.status = status;
  }
}

export async function requestAPI<T>(
  path: APIPath,
  options: RequestInit = {},
): Promise<T> {
  if (!path.startsWith(API_PREFIX)) {
    throw new TypeError("API requests must use a same-origin /api/v1/ path");
  }

  const headers = new Headers(options.headers);
  headers.set("Accept", "application/json");

  const response = await fetch(path, {
    ...options,
    cache: "no-store",
    credentials: "same-origin",
    headers,
    redirect: "error",
  });
  if (!response.ok) {
    throw new APIRequestError(
      response.status,
      `API request failed with status ${response.status}`,
    );
  }

  const contentType = response.headers.get("Content-Type") ?? "";
  if (!contentType.toLowerCase().includes("application/json")) {
    throw new APIRequestError(
      response.status,
      "API response did not contain JSON",
    );
  }
  return (await response.json()) as T;
}
