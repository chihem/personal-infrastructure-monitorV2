import { useQuery } from "@tanstack/react-query";

import { requestAPI } from "../../api/client";
import {
  parseMemoryCurrentResponse,
  parseMemoryHistoryResponse,
  type MemoryHistorySeries,
  type MemoryMetric,
  type MemorySnapshot,
} from "../../api/contracts";
import { buildMemoryHistoryPath, type MemoryHistorySelection } from "./model";

export const memoryQueryKeys = {
  all: ["memory"] as const,
  current: ["memory", "current"] as const,
  history: (selection: MemoryHistorySelection | null, metric: MemoryMetric) =>
    ["memory", "history", selection, metric] as const,
};

export function useCurrentMemory() {
  return useQuery({
    queryKey: memoryQueryKeys.current,
    queryFn: fetchCurrentMemory,
  });
}

export function useMemoryHistory(
  selection: MemoryHistorySelection | null,
  metric: MemoryMetric,
) {
  return useQuery({
    queryKey: memoryQueryKeys.history(selection, metric),
    queryFn: () =>
      fetchMemoryHistory(selection as MemoryHistorySelection, metric),
    enabled: selection !== null,
  });
}

export async function fetchCurrentMemory(): Promise<MemorySnapshot> {
  const response = parseMemoryCurrentResponse(
    await requestAPI("/api/v1/memory"),
  );
  if (response.data === null) {
    throw new Error("memory response did not contain data");
  }
  return response.data;
}

export async function fetchMemoryHistory(
  selection: MemoryHistorySelection,
  metric: MemoryMetric,
): Promise<MemoryHistorySeries> {
  const response = parseMemoryHistoryResponse(
    await requestAPI(buildMemoryHistoryPath(selection, metric)),
  );
  if (response.data === null) {
    throw new Error("memory history response did not contain data");
  }
  return response.data;
}
