import { render, screen } from "@testing-library/react";

import { DagView } from "./DagView";
import { StatusBadge } from "./StatusBadge";
import { ThroughputChart } from "./ThroughputChart";

describe("StatusBadge", () => {
  it("renders failed status with matching style class", () => {
    render(<StatusBadge status="failed" />);
    const badge = screen.getByText("failed");
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain("bg-red-100");
  });
});

describe("ThroughputChart", () => {
  it("renders chart container", () => {
    render(
      <div style={{ width: 900, height: 360 }}>
        <ThroughputChart
          data={[
            {
              timestamp: Date.now(),
              submitted: 3,
              completed: 2
            }
          ]}
          visible={{ submitted: true, completed: true }}
          onToggle={vi.fn()}
        />
      </div>
    );

    expect(screen.getByText(/throughput/i)).toBeInTheDocument();
  });
});

describe("DagView", () => {
  it("renders DAG container and controls", () => {
    render(
      <DagView
        tasks={[
          {
            id: "task-1",
            name: "Task 1",
            status: "pending",
            depends_on: []
          }
        ]}
      />
    );

    expect(screen.getByText("Fit to View")).toBeInTheDocument();
    expect(screen.getByText("Task Details")).toBeInTheDocument();
  });
});
