import { useParams } from "react-router-dom";

import { EmptyState } from "../components/common/EmptyState";

export function WorkflowDetailPage() {
  const { id = "" } = useParams();

  return (
    <section className="space-y-4">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Workflow Detail</h1>
        <p className="mt-1 break-all text-sm text-[var(--ui-muted)]">
          Workflow ID: <span className="font-mono">{id || "-"}</span>
        </p>
      </header>
      <EmptyState
        title="Detail panels pending"
        description="Task table, DAG, and action controls are implemented in the next section."
      />
    </section>
  );
}

