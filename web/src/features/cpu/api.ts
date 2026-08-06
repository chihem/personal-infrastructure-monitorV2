import { useQuery } from "@tanstack/react-query";

import { requestAPI } from "../../api/client";
import {
  parseCPUCurrentResponse,
  parseCPUHistoryResponse,
  type CPUHistorySeries,
  type CPUSnapshot,
} from "../../api/contracts";
import {
  buildCPUHistoryPath,
  type CPUHistorySelection,
  type CPUSelectedMetric,
} from "./model";

export const cpuQueryKeys = {
  all: ["cpu"] as const,
  current: ["cpu", "current"] as const,
  history: (selection: CPUHistorySelection | null, metric: CPUSelectedMetric) =>
    ["cpu", "history", selection, metric] as const,
};

export function useCurrentCPU() {
  return useQuery({
    queryKey: cpuQueryKeys.current,
    queryFn: fetchCurrentCPU,
  });
}

export function useCPUHistory(
  selection: CPUHistorySelection | null,
  metric: CPUSelectedMetric,
) {
  return useQuery({
    queryKey: cpuQueryKeys.history(selection, metric),
    queryFn: () => fetchCPUHistory(selection as CPUHistorySelection, metric),
    enabled: selection !== null,
  });
}

export async function fetchCurrentCPU(): Promise<CPUSnapshot> {
  const response = parseCPUCurrentResponse(await requestAPI("/api/v1/cpu"));
  if (response.data === null) {
    throw new Error("CPU response did not contain data");
  }
  return response.data;
}

export async function fetchCPUHistory(
  selection: CPUHistorySelection,
  metric: CPUSelectedMetric,
): Promise<CPUHistorySeries> {
  const response = parseCPUHistoryResponse(
    await requestAPI(buildCPUHistoryPath(selection, metric)),
  );
  if (response.data === null) {
    throw new Error("CPU history response did not contain data");
  }
  return response.data;
}
