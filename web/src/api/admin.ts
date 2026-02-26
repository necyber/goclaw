import { requestJSON } from "./client";
import type { AdminDebugInfo, EngineStatus, LaneStats } from "../types/api";

export async function getEngineStatus(signal?: AbortSignal): Promise<EngineStatus> {
  return requestJSON<EngineStatus>("/api/v1/admin/status", { method: "GET", signal });
}

export async function getLaneStats(signal?: AbortSignal): Promise<LaneStats[]> {
  const response = await requestJSON<{ lanes: LaneStats[] }>("/api/v1/admin/lanes", {
    method: "GET",
    signal
  });
  return response.lanes;
}

export async function pauseWorkflows(signal?: AbortSignal): Promise<{ message: string }> {
  return requestJSON<{ message: string }>("/api/v1/admin/pause", { method: "POST", signal });
}

export async function resumeWorkflows(signal?: AbortSignal): Promise<{ message: string }> {
  return requestJSON<{ message: string }>("/api/v1/admin/resume", { method: "POST", signal });
}

export async function purgeWorkflows(signal?: AbortSignal): Promise<{ message: string; deleted?: number }> {
  return requestJSON<{ message: string; deleted?: number }>("/api/v1/admin/purge", {
    method: "POST",
    signal
  });
}

export async function getDebugInfo(signal?: AbortSignal): Promise<AdminDebugInfo> {
  return requestJSON<AdminDebugInfo>("/api/v1/admin/debug", { method: "GET", signal });
}

