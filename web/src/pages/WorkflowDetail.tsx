import { Fragment, useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";

import { EmptyState } from "../components/common/EmptyState";
import { ErrorState } from "../components/common/ErrorState";
import { Loading } from "../components/common/Loading";
import { DagView } from "../components/DagView";
import { StatusBadge } from "../components/StatusBadge";
import { useWorkflowStore } from "../stores/workflows";
import { useWebSocketStore } from "../stores/websocket";

const NON_TERMINAL = new Set(["pending", "scheduled", "running"]);

function formatDuration(start?: string | null, end?: string | null) {
  if (!start || !end) {
    return "-";
  }
  const ms = new Date(end).getTime() - new Date(start).getTime();
  if (Number.isNaN(ms) || ms < 0) {
    return "-";
  }
  if (ms < 1000) {
    return `${ms} ms`;
  }
  return `${(ms / 1000).toFixed(2)} s`;
}

export function WorkflowDetailPage() {
  const { id = "" } = useParams();
  const workflow = useWorkflowStore((state) => state.selectedWorkflow);
  const loading = useWorkflowStore((state) => state.loadingDetail);
  const error = useWorkflowStore((state) => state.error);
  const loadWorkflowDetail = useWorkflowStore((state) => state.loadWorkflowDetail);
  const cancelWorkflowByID = useWorkflowStore((state) => state.cancelWorkflowByID);
  const subscribeWorkflow = useWebSocketStore((state) => state.subscribeWorkflow);
  const unsubscribeWorkflow = useWebSocketStore((state) => state.unsubscribeWorkflow);
  const [expandedTaskID, setExpandedTaskID] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"tasks" | "dag">("tasks");

  useEffect(() => {
    if (!id) {
      return;
    }
    void loadWorkflowDetail(id);
  }, [id, loadWorkflowDetail]);

  useEffect(() => {
    if (!id) {
      return;
    }
    subscribeWorkflow(id);
    return () => unsubscribeWorkflow(id);
  }, [id, subscribeWorkflow, unsubscribeWorkflow]);

  useEffect(() => {
    if (!workflow || !NON_TERMINAL.has(workflow.status)) {
      return;
    }
    const timer = window.setInterval(() => {
      if (id) {
        void loadWorkflowDetail(id);
      }
    }, 2000);
    return () => window.clearInterval(timer);
  }, [workflow, id, loadWorkflowDetail]);

  const canCancel = useMemo(
    () => Boolean(workflow && workflow.status === "running" && id),
    [workflow, id]
  );

  const onCancel = async () => {
    if (!id || !canCancel) {
      return;
    }
    const confirmed = window.confirm("Cancel this running workflow?");
    if (!confirmed) {
      return;
    }
    await cancelWorkflowByID(id);
  };

  return (
    <section className="space-y-4">
      <header className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">
              {workflow?.name || "Workflow Detail"}
            </h1>
            <p className="mt-1 break-all text-xs text-[var(--ui-muted)]">ID: {id || "-"}</p>
          </div>
          <div className="flex items-center gap-2">
            {workflow ? <StatusBadge status={workflow.status} /> : null}
            {canCancel ? (
              <button
                type="button"
                onClick={() => void onCancel()}
                className="rounded-md border border-red-300 px-3 py-1.5 text-xs font-semibold text-red-700 hover:bg-red-50 dark:border-red-500/60 dark:text-red-200 dark:hover:bg-red-900/40"
              >
                Cancel Workflow
              </button>
            ) : null}
          </div>
        </div>

        {workflow ? (
          <div className="mt-3 grid gap-2 text-sm text-[var(--ui-muted)] md:grid-cols-3">
            <p>Created: {new Date(workflow.created_at).toLocaleString()}</p>
            <p>
              Started: {workflow.started_at ? new Date(workflow.started_at).toLocaleString() : "-"}
            </p>
            <p>
              Completed:{" "}
              {workflow.completed_at ? new Date(workflow.completed_at).toLocaleString() : "-"}
            </p>
          </div>
        ) : null}

        {workflow?.metadata ? (
          <pre className="mt-3 overflow-auto rounded-md border border-[var(--ui-border)] bg-black/5 p-3 text-xs dark:bg-white/5">
            {JSON.stringify(workflow.metadata, null, 2)}
          </pre>
        ) : null}
      </header>

      {loading ? <Loading label="Loading workflow detail..." skeletonRows={6} /> : null}
      {!loading && error ? (
        <ErrorState message={error} onRetry={() => void loadWorkflowDetail(id)} />
      ) : null}
      {!loading && !error && workflow && workflow.tasks.length === 0 ? (
        <EmptyState
          title="No tasks"
          description="This workflow does not contain task status records."
        />
      ) : null}

      {!loading && !error && workflow && workflow.tasks.length > 0 ? (
        <div className="space-y-3">
          <div className="inline-flex rounded-md border border-[var(--ui-border)] bg-[var(--ui-panel)] p-1">
            <button
              type="button"
              onClick={() => setActiveTab("tasks")}
              className={`rounded px-3 py-1 text-sm ${
                activeTab === "tasks"
                  ? "bg-[var(--ui-accent)] text-[var(--ui-accent-fg)]"
                  : "text-[var(--ui-muted)]"
              }`}
            >
              Tasks
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("dag")}
              className={`rounded px-3 py-1 text-sm ${
                activeTab === "dag"
                  ? "bg-[var(--ui-accent)] text-[var(--ui-accent-fg)]"
                  : "text-[var(--ui-muted)]"
              }`}
            >
              DAG
            </button>
          </div>

          {activeTab === "dag" ? <DagView tasks={workflow.tasks} /> : null}

          {activeTab === "tasks" ? (
            <div className="overflow-x-auto rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)]">
              <table className="min-w-full divide-y divide-[var(--ui-border)] text-sm">
                <thead>
                  <tr className="text-left text-xs uppercase tracking-wide text-[var(--ui-muted)]">
                    <th className="px-4 py-3">ID</th>
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Duration</th>
                    <th className="px-4 py-3">Error</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--ui-border)]">
                  {workflow.tasks.map((task) => (
                    <Fragment key={task.id}>
                      <tr
                        className="cursor-pointer hover:bg-black/5 dark:hover:bg-white/5"
                        onClick={() =>
                          setExpandedTaskID((current) => (current === task.id ? null : task.id))
                        }
                      >
                        <td className="px-4 py-3 font-mono text-xs">{task.id}</td>
                        <td className="px-4 py-3">{task.name}</td>
                        <td className="px-4 py-3">
                          <StatusBadge status={task.status} />
                        </td>
                        <td className="px-4 py-3 text-[var(--ui-muted)]">
                          {formatDuration(task.started_at, task.completed_at)}
                        </td>
                        <td className="px-4 py-3 text-xs text-red-600 dark:text-red-300">
                          {task.error || "-"}
                        </td>
                      </tr>
                      {expandedTaskID === task.id ? (
                        <tr>
                          <td className="px-4 pb-4" colSpan={5}>
                            <pre className="overflow-auto rounded-md border border-[var(--ui-border)] bg-black/5 p-3 text-xs dark:bg-white/5">
                              {JSON.stringify(
                                task.result ?? { message: "No result data" },
                                null,
                                2
                              )}
                            </pre>
                          </td>
                        </tr>
                      ) : null}
                    </Fragment>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
