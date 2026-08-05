import { QueryClient } from "@tanstack/react-query";

export const DASHBOARD_REFRESH_INTERVAL_MS = 60_000;

export function createAppQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        refetchInterval: DASHBOARD_REFRESH_INTERVAL_MS,
        refetchIntervalInBackground: false,
        refetchOnWindowFocus: true,
        retry: 1,
        staleTime: DASHBOARD_REFRESH_INTERVAL_MS,
      },
      mutations: {
        retry: false,
      },
    },
  });
}
