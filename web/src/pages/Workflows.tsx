import { EmptyState } from "../components/common/EmptyState";

export function WorkflowsPage() {
  return (
    <section className="space-y-4">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Workflows</h1>
        <p className="mt-1 text-sm text-[var(--ui-muted)]">
          Browse, filter, and inspect workflow execution states.
        </p>
      </header>
      <EmptyState
        title="No workflows loaded"
        description="Workflow list/table UI is implemented in the next section."
      />
    </section>
  );
}

