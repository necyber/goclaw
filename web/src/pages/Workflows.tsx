import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { EmptyState } from "../components/common/EmptyState";
import { ErrorState } from "../components/common/ErrorState";
import { Loading } from "../components/common/Loading";
import { StatusBadge } from "../components/StatusBadge";
import { SubmitWorkflowDialog } from "../components/SubmitWorkflowDialog";
import { useWorkflowStore } from "../stores/workflows";

const NON_TERMINAL = new Set(["pending", "scheduled", "running"]);

export function WorkflowsPage() {
  const navigate = useNavigate();
  const workflows = useWorkflowStore((state) => state.workflows);
  const total = useWorkflowStore((state) => state.total);
  const limit = useWorkflowStore((state) => state.limit);
  const offset = useWorkflowStore((state) => state.offset);
  const loading = useWorkflowStore((state) => state.loadingList);
  const error = useWorkflowStore((state) => state.error);
  const search = useWorkflowStore((state) => state.search);
  const statusFilter = useWorkflowStore((state) => state.statusFilter);
  const loadWorkflows = useWorkflowStore((state) => state.loadWorkflows);
  const setPage = useWorkflowStore((state) => state.setPage);
  const setSearch = useWorkflowStore((state) => state.setSearch);
  const setStatusFilter = useWorkflowStore((state) => state.setStatusFilter);
  const [openSubmit, setOpenSubmit] = useState(false);

  useEffect(() => {
    void loadWorkflows();
  }, [loadWorkflows, offset, search, statusFilter]);

  useEffect(() => {
    if (!workflows.some((item) => NON_TERMINAL.has(item.status))) {
      return;
    }
    const timer = window.setInterval(() => {
      void loadWorkflows();
    }, 2000);
    return () => window.clearInterval(timer);
  }, [workflows, loadWorkflows]);

  const pageIndex = useMemo(() => Math.floor(offset / limit), [offset, limit]);
  const hasPrev = offset > 0;
  const hasNext = offset + limit < total;

  return (
    <section className="space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Workflows</h1>
          <p className="mt-1 text-sm text-[var(--ui-muted)]">
            Monitor and operate submitted workflows.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setOpenSubmit(true)}
          className="rounded-md bg-[var(--ui-accent)] px-4 py-2 text-sm font-semibold text-[var(--ui-accent-fg)]"
        >
          Submit Workflow
        </button>
      </header>

      <div className="grid gap-3 rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-3 md:grid-cols-[220px_1fr]">
        <select
          className="rounded-md border border-[var(--ui-border)] bg-transparent px-3 py-2 text-sm"
          value={statusFilter}
          onChange={(event) => setStatusFilter(event.target.value as typeof statusFilter)}
        >
          <option value="all">All statuses</option>
          <option value="pending">Pending</option>
          <option value="scheduled">Scheduled</option>
          <option value="running">Running</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
          <option value="cancelled">Cancelled</option>
        </select>
        <input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Search by workflow name"
          className="rounded-md border border-[var(--ui-border)] bg-transparent px-3 py-2 text-sm"
        />
      </div>

      {loading ? <Loading label="Loading workflows..." skeletonRows={5} /> : null}
      {!loading && error ? (
        <ErrorState message={error} onRetry={() => void loadWorkflows()} />
      ) : null}
      {!loading && !error && workflows.length === 0 ? (
        <EmptyState
          title="No workflows found"
          description="Try adjusting status filter/search, or submit a new workflow."
        />
      ) : null}

      {!loading && !error && workflows.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)]">
          <table className="min-w-full divide-y divide-[var(--ui-border)] text-sm">
            <thead>
              <tr className="text-left text-xs uppercase tracking-wide text-[var(--ui-muted)]">
                <th className="px-4 py-3">ID</th>
                <th className="px-4 py-3">Name</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Created</th>
                <th className="px-4 py-3">Tasks</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--ui-border)]">
              {workflows.map((workflow) => (
                <tr
                  key={workflow.id}
                  className="cursor-pointer hover:bg-black/5 dark:hover:bg-white/5"
                  onClick={() => navigate(`/workflows/${workflow.id}`)}
                >
                  <td className="px-4 py-3 font-mono text-xs">
                    <Link to={`/workflows/${workflow.id}`} className="hover:underline">
                      {workflow.id}
                    </Link>
                  </td>
                  <td className="px-4 py-3 font-medium">{workflow.name}</td>
                  <td className="px-4 py-3">
                    <StatusBadge status={workflow.status} />
                  </td>
                  <td className="px-4 py-3 text-[var(--ui-muted)]">
                    {new Date(workflow.created_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-3">{workflow.task_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      <footer className="flex items-center justify-between rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] px-4 py-3 text-sm">
        <p className="text-[var(--ui-muted)]">
          Page {pageIndex + 1} · {total} total
        </p>
        <div className="flex gap-2">
          <button
            type="button"
            disabled={!hasPrev}
            onClick={() => setPage(pageIndex - 1)}
            className="rounded-md border border-[var(--ui-border)] px-3 py-1.5 disabled:opacity-50"
          >
            Previous
          </button>
          <button
            type="button"
            disabled={!hasNext}
            onClick={() => setPage(pageIndex + 1)}
            className="rounded-md border border-[var(--ui-border)] px-3 py-1.5 disabled:opacity-50"
          >
            Next
          </button>
        </div>
      </footer>

      <SubmitWorkflowDialog open={openSubmit} onClose={() => setOpenSubmit(false)} />
    </section>
  );
}
