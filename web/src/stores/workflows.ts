import { create } from "zustand";

import { cancelWorkflow, getWorkflow, listWorkflows, submitWorkflow } from "../api/workflows";
import type {
  SubmitWorkflowRequest,
  WorkflowDetail,
  WorkflowListResponse,
  WorkflowState,
  WorkflowSummary
} from "../types/api";
import { useWebSocketStore } from "./websocket";
import type { WebSocketEventMessage } from "../lib/websocket";

type WorkflowStoreState = {
  workflows: WorkflowSummary[];
  total: number;
  limit: number;
  offset: number;
  search: string;
  statusFilter: WorkflowState | "all";
  selectedWorkflow: WorkflowDetail | null;
  loadingList: boolean;
  loadingDetail: boolean;
  error: string | null;
  setSearch: (search: string) => void;
  setStatusFilter: (status: WorkflowState | "all") => void;
  setPage: (pageIndex: number) => void;
  loadWorkflows: () => Promise<void>;
  loadWorkflowDetail: (workflowID: string) => Promise<void>;
  submitWorkflowJSON: (json: string) => Promise<string>;
  cancelWorkflowByID: (workflowID: string) => Promise<void>;
};

let subscribedToWS = false;

function applyWorkflowStateEvent(state: WorkflowStoreState, event: WebSocketEventMessage) {
  const payload = event.payload as Record<string, unknown>;
  const workflowID = String(payload.workflow_id ?? "");
  const nextState = String(payload.new_state ?? "") as WorkflowState;
  if (!workflowID || !nextState) {
    return state;
  }

  const workflows = state.workflows.map((item) =>
    item.id === workflowID ? { ...item, status: nextState } : item
  );
  const selectedWorkflow =
    state.selectedWorkflow && state.selectedWorkflow.id === workflowID
      ? { ...state.selectedWorkflow, status: nextState }
      : state.selectedWorkflow;
  return { ...state, workflows, selectedWorkflow };
}

function applyTaskStateEvent(state: WorkflowStoreState, event: WebSocketEventMessage) {
  const payload = event.payload as Record<string, unknown>;
  const workflowID = String(payload.workflow_id ?? "");
  const taskID = String(payload.task_id ?? "");
  const nextState = String(payload.new_state ?? "") as WorkflowState;
  if (!workflowID || !taskID || !nextState) {
    return state;
  }
  if (!state.selectedWorkflow || state.selectedWorkflow.id !== workflowID) {
    return state;
  }

  const tasks = state.selectedWorkflow.tasks.map((task) =>
    task.id === taskID
      ? {
          ...task,
          status: nextState,
          error: payload.error ? String(payload.error) : task.error,
          result: payload.result ?? task.result
        }
      : task
  );

  return {
    ...state,
    selectedWorkflow: {
      ...state.selectedWorkflow,
      tasks
    }
  };
}

function registerWebSocketBridge(set: (updater: (state: WorkflowStoreState) => WorkflowStoreState) => void) {
  if (subscribedToWS) {
    return;
  }
  subscribedToWS = true;

  const ws = useWebSocketStore.getState();
  ws.onEvent("workflow.state_changed", (event) => {
    set((state) => applyWorkflowStateEvent(state, event));
  });
  ws.onEvent("task.state_changed", (event) => {
    set((state) => applyTaskStateEvent(state, event));
  });
}

export const useWorkflowStore = create<WorkflowStoreState>((set, get) => {
  registerWebSocketBridge(set);

  return {
    workflows: [],
    total: 0,
    limit: 20,
    offset: 0,
    search: "",
    statusFilter: "all",
    selectedWorkflow: null,
    loadingList: false,
    loadingDetail: false,
    error: null,
    setSearch: (search) => set({ search, offset: 0 }),
    setStatusFilter: (statusFilter) => set({ statusFilter, offset: 0 }),
    setPage: (pageIndex) => set({ offset: Math.max(pageIndex, 0) * get().limit }),
    loadWorkflows: async () => {
      set({ loadingList: true, error: null });
      try {
        const statusFilter = get().statusFilter;
        const response: WorkflowListResponse = await listWorkflows({
          limit: get().limit,
          offset: get().offset,
          status: statusFilter === "all" ? undefined : statusFilter,
          search: get().search || undefined
        });
        set({
          workflows: response.workflows,
          total: response.total,
          loadingList: false,
          error: null
        });
      } catch (err) {
        set({ loadingList: false, error: (err as Error).message });
      }
    },
    loadWorkflowDetail: async (workflowID) => {
      set({ loadingDetail: true, error: null });
      try {
        const workflow = await getWorkflow(workflowID);
        set({ selectedWorkflow: workflow, loadingDetail: false, error: null });
      } catch (err) {
        set({ loadingDetail: false, error: (err as Error).message });
      }
    },
    submitWorkflowJSON: async (json) => {
      let payload: SubmitWorkflowRequest;
      try {
        payload = JSON.parse(json) as SubmitWorkflowRequest;
      } catch (err) {
        throw new Error(`Invalid JSON: ${(err as Error).message}`);
      }
      const response = await submitWorkflow(payload);
      await get().loadWorkflows();
      return response.id;
    },
    cancelWorkflowByID: async (workflowID) => {
      await cancelWorkflow(workflowID);
      if (get().selectedWorkflow?.id === workflowID) {
        await get().loadWorkflowDetail(workflowID);
      }
      await get().loadWorkflows();
    }
  };
});
