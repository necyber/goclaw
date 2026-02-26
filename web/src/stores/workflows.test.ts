import type { WebSocketEventMessage } from "../lib/websocket";

const wsMock = vi.hoisted(() => ({
  workflowStateHandler: null as ((event: WebSocketEventMessage) => void) | null,
  taskStateHandler: null as ((event: WebSocketEventMessage) => void) | null,
}));

vi.mock("../api/workflows", () => ({
  listWorkflows: vi.fn(),
  getWorkflow: vi.fn(),
  submitWorkflow: vi.fn(),
  cancelWorkflow: vi.fn(),
}));

vi.mock("./websocket", () => ({
  useWebSocketStore: {
    getState: () => ({
      onEvent: (eventType: string, handler: (event: WebSocketEventMessage) => void) => {
        if (eventType === "workflow.state_changed") {
          wsMock.workflowStateHandler = handler;
        }
        if (eventType === "task.state_changed") {
          wsMock.taskStateHandler = handler;
        }
        return () => {};
      },
    }),
  },
}));

import { listWorkflows, submitWorkflow } from "../api/workflows";
import type { WorkflowDetail, WorkflowSummary } from "../types/api";
import { useWorkflowStore } from "./workflows";

const mockedListWorkflows = vi.mocked(listWorkflows);
const mockedSubmitWorkflow = vi.mocked(submitWorkflow);

const baseWorkflow: WorkflowSummary = {
  id: "wf-1",
  name: "Workflow 1",
  status: "pending",
  created_at: new Date().toISOString(),
  task_count: 1,
};

const baseDetail: WorkflowDetail = {
  id: "wf-1",
  name: "Workflow 1",
  status: "pending",
  created_at: new Date().toISOString(),
  tasks: [
    {
      id: "task-1",
      name: "Task 1",
      status: "pending",
    },
  ],
};

describe("workflow store", () => {
  beforeEach(() => {
    mockedListWorkflows.mockReset();
    mockedSubmitWorkflow.mockReset();
    useWorkflowStore.setState({
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
    });
  });

  it("loads workflows from API", async () => {
    mockedListWorkflows.mockResolvedValue({
      workflows: [baseWorkflow],
      total: 1,
      limit: 20,
      offset: 0,
    });

    await useWorkflowStore.getState().loadWorkflows();
    const next = useWorkflowStore.getState();
    expect(next.workflows).toHaveLength(1);
    expect(next.workflows[0].id).toBe("wf-1");
    expect(next.total).toBe(1);
  });

  it("submits workflow JSON and reloads list", async () => {
    mockedSubmitWorkflow.mockResolvedValue({
      id: "wf-new",
      name: "Created",
      status: "pending",
    });
    mockedListWorkflows.mockResolvedValue({
      workflows: [baseWorkflow],
      total: 1,
      limit: 20,
      offset: 0,
    });

    const workflowID = await useWorkflowStore
      .getState()
      .submitWorkflowJSON(JSON.stringify({ name: "demo", tasks: [] }));

    expect(workflowID).toBe("wf-new");
    expect(mockedSubmitWorkflow).toHaveBeenCalledTimes(1);
    expect(mockedListWorkflows).toHaveBeenCalledTimes(1);
  });

  it("applies websocket workflow and task state updates", () => {
    useWorkflowStore.setState({
      workflows: [baseWorkflow],
      selectedWorkflow: baseDetail,
    });

    expect(wsMock.workflowStateHandler).not.toBeNull();
    expect(wsMock.taskStateHandler).not.toBeNull();

    wsMock.workflowStateHandler?.({
      type: "workflow.state_changed",
      timestamp: new Date().toISOString(),
      payload: {
        workflow_id: "wf-1",
        new_state: "running",
      },
    });

    let state = useWorkflowStore.getState();
    expect(state.workflows[0].status).toBe("running");
    expect(state.selectedWorkflow?.status).toBe("running");

    wsMock.taskStateHandler?.({
      type: "task.state_changed",
      timestamp: new Date().toISOString(),
      payload: {
        workflow_id: "wf-1",
        task_id: "task-1",
        new_state: "completed",
        result: { ok: true },
      },
    });

    state = useWorkflowStore.getState();
    expect(state.selectedWorkflow?.tasks[0].status).toBe("completed");
    expect(state.selectedWorkflow?.tasks[0].result).toEqual({ ok: true });
  });
});
