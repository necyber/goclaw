import { requestJSON } from "./client";
import type {
  SubmitWorkflowRequest,
  SubmitWorkflowResponse,
  TaskResultResponse,
  WorkflowDetail,
  WorkflowListResponse,
  WorkflowState,
} from "../types/api";

export type ListWorkflowParams = {
  status?: WorkflowState;
  search?: string;
  limit?: number;
  offset?: number;
};

export async function listWorkflows(
  params: ListWorkflowParams = {},
  signal?: AbortSignal
): Promise<WorkflowListResponse> {
  return requestJSON<WorkflowListResponse>("/api/v1/workflows", {
    method: "GET",
    signal,
    query: {
      status: params.status,
      search: params.search,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    },
  });
}

export async function getWorkflow(id: string, signal?: AbortSignal): Promise<WorkflowDetail> {
  return requestJSON<WorkflowDetail>(`/api/v1/workflows/${encodeURIComponent(id)}`, {
    method: "GET",
    signal,
  });
}

export async function submitWorkflow(
  payload: SubmitWorkflowRequest,
  signal?: AbortSignal
): Promise<SubmitWorkflowResponse> {
  return requestJSON<SubmitWorkflowResponse>("/api/v1/workflows", {
    method: "POST",
    body: payload,
    signal,
  });
}

export async function cancelWorkflow(
  id: string,
  signal?: AbortSignal
): Promise<{ message: string }> {
  return requestJSON<{ message: string }>(`/api/v1/workflows/${encodeURIComponent(id)}/cancel`, {
    method: "POST",
    signal,
  });
}

export async function getTaskResult(
  workflowID: string,
  taskID: string,
  signal?: AbortSignal
): Promise<TaskResultResponse> {
  return requestJSON<TaskResultResponse>(
    `/api/v1/workflows/${encodeURIComponent(workflowID)}/tasks/${encodeURIComponent(taskID)}/result`,
    {
      method: "GET",
      signal,
    }
  );
}
