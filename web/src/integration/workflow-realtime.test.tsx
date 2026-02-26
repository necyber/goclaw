import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import type { WebSocketEventMessage } from "../lib/websocket";

const wsMock = vi.hoisted(() => ({
  workflowStateHandler: null as ((event: WebSocketEventMessage) => void) | null,
}));

vi.mock("../api/workflows", () => ({
  listWorkflows: vi.fn(),
  getWorkflow: vi.fn(),
  submitWorkflow: vi.fn(),
  cancelWorkflow: vi.fn(),
}));

vi.mock("../stores/websocket", () => ({
  useWebSocketStore: {
    getState: () => ({
      onEvent: (eventType: string, handler: (event: WebSocketEventMessage) => void) => {
        if (eventType === "workflow.state_changed") {
          wsMock.workflowStateHandler = handler;
        }
        return () => {};
      },
    }),
  },
}));

import { listWorkflows, submitWorkflow } from "../api/workflows";
import { WorkflowsPage } from "../pages/Workflows";
import { useWorkflowStore } from "../stores/workflows";

const mockedListWorkflows = vi.mocked(listWorkflows);
const mockedSubmitWorkflow = vi.mocked(submitWorkflow);

describe("workflow realtime integration", () => {
  beforeEach(() => {
    mockedListWorkflows.mockReset();
    mockedSubmitWorkflow.mockReset();

    mockedListWorkflows.mockResolvedValue({
      workflows: [
        {
          id: "wf-1",
          name: "Realtime Demo",
          status: "pending",
          created_at: new Date().toISOString(),
          task_count: 1,
        },
      ],
      total: 1,
      limit: 20,
      offset: 0,
    });
    mockedSubmitWorkflow.mockResolvedValue({
      id: "wf-1",
      name: "Realtime Demo",
      status: "pending",
    });

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

  it("flows from submit to websocket state change and updates UI", async () => {
    render(
      <MemoryRouter initialEntries={["/workflows"]}>
        <WorkflowsPage />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText("Realtime Demo")).toBeInTheDocument();
    });
    expect(screen.getByText("pending")).toBeInTheDocument();

    await useWorkflowStore
      .getState()
      .submitWorkflowJSON(JSON.stringify({ name: "Realtime Demo", tasks: [] }));

    expect(mockedSubmitWorkflow).toHaveBeenCalledTimes(1);
    expect(wsMock.workflowStateHandler).not.toBeNull();

    wsMock.workflowStateHandler?.({
      type: "workflow.state_changed",
      timestamp: new Date().toISOString(),
      payload: {
        workflow_id: "wf-1",
        new_state: "completed",
      },
    });

    await waitFor(() => {
      expect(screen.getByText("completed")).toBeInTheDocument();
    });
  });
});
