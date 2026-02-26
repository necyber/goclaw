import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";

import {
  getDebugInfo,
  getEngineStatus,
  getLaneStats,
  pauseWorkflows,
  purgeWorkflows,
  resumeWorkflows
} from "../api/admin";
import { fetchMetrics } from "../api/metrics";
import { EmptyState } from "../components/common/EmptyState";
import { ErrorState } from "../components/common/ErrorState";
import { Loading } from "../components/common/Loading";
import type { AdminDebugInfo, EngineStatus, LaneStats, PrometheusSample } from "../types/api";

type LaneHistoryPoint = {
  timestamp: number;
  queueDepth: number;
};

type ClusterNode = {
  id: string;
  address: string;
  status: string;
  lastHeartbeat: string;
};

const LANE_HISTORY_LIMIT = 180;

function formatNumber(value: number | undefined): string {
  if (value === undefined || Number.isNaN(value)) {
    return "-";
  }
  return new Intl.NumberFormat().format(Math.round(value));
}

function formatBytes(bytes: number | undefined): string {
  if (bytes === undefined || Number.isNaN(bytes)) {
    return "-";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[unitIndex]}`;
}

function stateBadgeClass(state: EngineStatus["state"] | undefined): string {
  switch (state) {
    case "running":
      return "bg-emerald-500";
    case "idle":
      return "bg-amber-500";
    case "stopped":
      return "bg-slate-400";
    case "error":
      return "bg-red-500";
    default:
      return "bg-slate-500";
  }
}

function readMetric(samples: PrometheusSample[], status: string): number {
  return samples.reduce((total, sample) => {
    if (sample.metric !== "workflow_submissions_total") {
      return total;
    }
    if (sample.labels.status !== status) {
      return total;
    }
    return total + sample.value;
  }, 0);
}

function downloadContent(content: string, fileName: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function parseClusterNodes(value: unknown): ClusterNode[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((item) => {
      const row = asRecord(item);
      if (!row) {
        return null;
      }
      return {
        id: String(row.id ?? row.node_id ?? "unknown"),
        address: String(row.address ?? row.addr ?? "-"),
        status: String(row.status ?? "unknown"),
        lastHeartbeat: String(row.last_heartbeat ?? row.lastHeartbeat ?? "-")
      };
    })
    .filter((item): item is ClusterNode => item !== null);
}

function extractClusterNodes(debugInfo: AdminDebugInfo | null): ClusterNode[] {
  if (!debugInfo) {
    return [];
  }
  const system = asRecord(debugInfo.system);
  if (!system) {
    return [];
  }

  const directNodes = parseClusterNodes(system.cluster_nodes ?? system.nodes);
  if (directNodes.length > 0) {
    return directNodes;
  }

  const cluster = asRecord(system.cluster);
  if (!cluster) {
    return [];
  }
  return parseClusterNodes(cluster.nodes);
}

function formatTimeTick(value: number) {
  return new Date(value).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit"
  });
}

function toErrorText(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return "unknown error";
}

export function AdminPage() {
  const mountedRef = useRef(true);
  const [engineStatus, setEngineStatus] = useState<EngineStatus | null>(null);
  const [lanes, setLanes] = useState<LaneStats[]>([]);
  const [laneHistory, setLaneHistory] = useState<Record<string, LaneHistoryPoint[]>>({});
  const [debugInfo, setDebugInfo] = useState<AdminDebugInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [debugError, setDebugError] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);
  const [expandedLane, setExpandedLane] = useState<string | null>(null);
  const [busyAction, setBusyAction] = useState<"pause" | "resume" | "purge" | "debug" | "metrics" | null>(
    null
  );
  const [purgeEstimate, setPurgeEstimate] = useState(0);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const refreshAdmin = useCallback(async () => {
    const [statusResult, lanesResult] = await Promise.allSettled([getEngineStatus(), getLaneStats()]);

    if (!mountedRef.current) {
      return;
    }

    const issues: string[] = [];
    if (statusResult.status === "fulfilled") {
      setEngineStatus(statusResult.value);
    } else {
      issues.push(`engine status: ${toErrorText(statusResult.reason)}`);
    }

    if (lanesResult.status === "fulfilled") {
      const now = Date.now();
      setLanes(lanesResult.value);
      setLaneHistory((previous) => {
        const next: Record<string, LaneHistoryPoint[]> = { ...previous };
        for (const lane of lanesResult.value) {
          const history = next[lane.name] ?? [];
          next[lane.name] = [...history, { timestamp: now, queueDepth: lane.queue_depth }].slice(
            -LANE_HISTORY_LIMIT
          );
        }
        return next;
      });
    } else {
      issues.push(`lane stats: ${toErrorText(lanesResult.reason)}`);
    }

    setError(issues.length > 0 ? issues.join(" | ") : null);
    setLoading(false);
  }, []);

  const refreshDebugInfo = useCallback(async () => {
    try {
      const info = await getDebugInfo();
      if (!mountedRef.current) {
        return;
      }
      setDebugInfo(info);
      setDebugError(null);
    } catch (refreshError) {
      if (!mountedRef.current) {
        return;
      }
      setDebugError(`debug info: ${toErrorText(refreshError)}`);
    }
  }, []);

  const refreshPurgeEstimate = useCallback(async () => {
    try {
      const samples = await fetchMetrics();
      if (!mountedRef.current) {
        return;
      }
      const completed = readMetric(samples, "completed");
      const failed = readMetric(samples, "failed");
      setPurgeEstimate(Math.max(0, Math.round(completed + failed)));
    } catch {
      if (!mountedRef.current) {
        return;
      }
      setPurgeEstimate(0);
    }
  }, []);

  useEffect(() => {
    void refreshAdmin();
    const timer = window.setInterval(() => {
      void refreshAdmin();
    }, 5000);
    return () => window.clearInterval(timer);
  }, [refreshAdmin]);

  useEffect(() => {
    void refreshDebugInfo();
    void refreshPurgeEstimate();
    const timer = window.setInterval(() => {
      void refreshDebugInfo();
      void refreshPurgeEstimate();
    }, 30_000);
    return () => window.clearInterval(timer);
  }, [refreshDebugInfo, refreshPurgeEstimate]);

  const onPause = async () => {
    const confirmed = window.confirm(
      "Pause workflow processing? Running workflows will continue, but no new workflows will start."
    );
    if (!confirmed) {
      return;
    }
    setBusyAction("pause");
    try {
      const response = await pauseWorkflows();
      setActionMessage(response.message);
      await refreshAdmin();
    } catch (actionError) {
      setError(`pause failed: ${toErrorText(actionError)}`);
    } finally {
      setBusyAction(null);
    }
  };

  const onResume = async () => {
    const confirmed = window.confirm("Resume workflow processing now?");
    if (!confirmed) {
      return;
    }
    setBusyAction("resume");
    try {
      const response = await resumeWorkflows();
      setActionMessage(response.message);
      await refreshAdmin();
    } catch (actionError) {
      setError(`resume failed: ${toErrorText(actionError)}`);
    } finally {
      setBusyAction(null);
    }
  };

  const onPurge = async () => {
    const countText = purgeEstimate > 0 ? `${purgeEstimate}` : "unknown";
    const confirmed = window.confirm(
      `Purge completed and failed workflows now? Estimated removable workflows: ${countText}.`
    );
    if (!confirmed) {
      return;
    }

    setBusyAction("purge");
    try {
      const response = await purgeWorkflows();
      const deleted = response.deleted ?? 0;
      setActionMessage(`${response.message} (deleted: ${deleted})`);
      await Promise.all([refreshAdmin(), refreshPurgeEstimate()]);
    } catch (actionError) {
      setError(`purge failed: ${toErrorText(actionError)}`);
    } finally {
      setBusyAction(null);
    }
  };

  const onExportDebug = async () => {
    setBusyAction("debug");
    try {
      const payload = debugInfo ?? (await getDebugInfo());
      downloadContent(
        JSON.stringify(payload, null, 2),
        `goclaw-debug-${Date.now()}.json`,
        "application/json"
      );
      setActionMessage("Debug info exported.");
    } catch (actionError) {
      setError(`debug export failed: ${toErrorText(actionError)}`);
    } finally {
      setBusyAction(null);
    }
  };

  const onExportMetrics = async () => {
    setBusyAction("metrics");
    try {
      const response = await fetch("/metrics", {
        method: "GET",
        headers: { Accept: "text/plain" }
      });
      if (!response.ok) {
        throw new Error(`metrics export HTTP ${response.status}`);
      }
      const text = await response.text();
      downloadContent(text, `goclaw-metrics-${Date.now()}.txt`, "text/plain;charset=utf-8");
      setActionMessage("Prometheus metrics exported.");
    } catch (actionError) {
      setError(`metrics export failed: ${toErrorText(actionError)}`);
    } finally {
      setBusyAction(null);
    }
  };

  const clusterNodes = useMemo(() => extractClusterNodes(debugInfo), [debugInfo]);
  const state = engineStatus?.state ?? "unknown";

  return (
    <section className="space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Admin</h1>
          <p className="mt-1 text-sm text-[var(--ui-muted)]">
            Engine controls, lane diagnostics, and export tools.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => void onPause()}
            disabled={busyAction !== null}
            className="rounded-md border border-amber-300 px-3 py-2 text-xs font-semibold text-amber-700 disabled:opacity-50 dark:border-amber-500/50 dark:text-amber-200"
          >
            {busyAction === "pause" ? "Pausing..." : "Pause"}
          </button>
          <button
            type="button"
            onClick={() => void onResume()}
            disabled={busyAction !== null}
            className="rounded-md border border-emerald-300 px-3 py-2 text-xs font-semibold text-emerald-700 disabled:opacity-50 dark:border-emerald-500/50 dark:text-emerald-200"
          >
            {busyAction === "resume" ? "Resuming..." : "Resume"}
          </button>
          <button
            type="button"
            onClick={() => void onPurge()}
            disabled={busyAction !== null}
            className="rounded-md border border-red-300 px-3 py-2 text-xs font-semibold text-red-700 disabled:opacity-50 dark:border-red-500/50 dark:text-red-200"
          >
            {busyAction === "purge" ? "Purging..." : "Purge Workflows"}
          </button>
          <button
            type="button"
            onClick={() => void onExportDebug()}
            disabled={busyAction !== null}
            className="rounded-md border border-[var(--ui-border)] px-3 py-2 text-xs font-semibold disabled:opacity-50"
          >
            {busyAction === "debug" ? "Exporting..." : "Export Debug Info"}
          </button>
          <button
            type="button"
            onClick={() => void onExportMetrics()}
            disabled={busyAction !== null}
            className="rounded-md border border-[var(--ui-border)] px-3 py-2 text-xs font-semibold disabled:opacity-50"
          >
            {busyAction === "metrics" ? "Exporting..." : "Export Metrics"}
          </button>
        </div>
      </header>

      {actionMessage ? (
        <section className="rounded-xl border border-emerald-300/70 bg-emerald-50/70 p-3 text-sm text-emerald-900 dark:border-emerald-500/40 dark:bg-emerald-950/25 dark:text-emerald-100">
          {actionMessage}
        </section>
      ) : null}

      {error ? <ErrorState message={error} onRetry={() => void refreshAdmin()} /> : null}
      {debugError ? (
        <section className="rounded-xl border border-amber-300/70 bg-amber-50/70 p-3 text-sm text-amber-900 dark:border-amber-500/40 dark:bg-amber-950/25 dark:text-amber-100">
          Debug information unavailable: {debugError}
        </section>
      ) : null}

      <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">Engine Status</h2>
        <div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <article className="rounded-lg border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-3">
            <p className="text-xs uppercase tracking-wide text-[var(--ui-muted)]">State</p>
            <p className="mt-2 inline-flex items-center gap-2 text-lg font-semibold capitalize">
              <span className={`h-2.5 w-2.5 rounded-full ${stateBadgeClass(engineStatus?.state)}`} />
              {state}
            </p>
          </article>
          <article className="rounded-lg border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-3">
            <p className="text-xs uppercase tracking-wide text-[var(--ui-muted)]">Uptime</p>
            <p className="mt-2 text-lg font-semibold">{engineStatus?.uptime ?? "-"}</p>
          </article>
          <article className="rounded-lg border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-3">
            <p className="text-xs uppercase tracking-wide text-[var(--ui-muted)]">Version</p>
            <p className="mt-2 text-lg font-semibold">{engineStatus?.version ?? "-"}</p>
          </article>
          <article className="rounded-lg border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-3">
            <p className="text-xs uppercase tracking-wide text-[var(--ui-muted)]">Active Workflows</p>
            <p className="mt-2 text-lg font-semibold">{formatNumber(engineStatus?.active_workflows)}</p>
          </article>
          <article className="rounded-lg border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-3">
            <p className="text-xs uppercase tracking-wide text-[var(--ui-muted)]">Goroutines</p>
            <p className="mt-2 text-lg font-semibold">{formatNumber(engineStatus?.goroutines)}</p>
          </article>
          <article className="rounded-lg border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-3">
            <p className="text-xs uppercase tracking-wide text-[var(--ui-muted)]">Memory</p>
            <p className="mt-2 text-lg font-semibold">{formatBytes(engineStatus?.memory_bytes)}</p>
          </article>
        </div>
      </section>

      <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
            Lane Statistics
          </h2>
          <p className="text-xs text-[var(--ui-muted)]">Auto-refresh: 5s</p>
        </div>

        {loading && lanes.length === 0 ? <Loading label="Loading lane stats..." skeletonRows={4} /> : null}

        {!loading && lanes.length === 0 ? (
          <EmptyState
            title="No lane stats available"
            description="No lane metrics were returned by the admin API."
          />
        ) : null}

        {lanes.length > 0 ? (
          <div className="overflow-x-auto rounded-lg border border-[var(--ui-border)]">
            <table className="min-w-full divide-y divide-[var(--ui-border)] text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-[var(--ui-muted)]">
                  <th className="px-4 py-3">Lane</th>
                  <th className="px-4 py-3">Queue Depth</th>
                  <th className="px-4 py-3">Workers</th>
                  <th className="px-4 py-3">Throughput/s</th>
                  <th className="px-4 py-3">Error Rate</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--ui-border)]">
                {lanes.map((lane) => {
                  const expanded = expandedLane === lane.name;
                  const history = laneHistory[lane.name] ?? [];
                  return (
                    <Fragment key={lane.name}>
                      <tr
                        className="cursor-pointer hover:bg-black/5 dark:hover:bg-white/5"
                        onClick={() => setExpandedLane((current) => (current === lane.name ? null : lane.name))}
                      >
                        <td className="px-4 py-3 font-semibold">{lane.name}</td>
                        <td className="px-4 py-3">{formatNumber(lane.queue_depth)}</td>
                        <td className="px-4 py-3">{formatNumber(lane.workers)}</td>
                        <td className="px-4 py-3">{lane.throughput_per_sec.toFixed(2)}</td>
                        <td className="px-4 py-3">{(lane.error_rate * 100).toFixed(2)}%</td>
                      </tr>
                      {expanded ? (
                        <tr>
                          <td className="px-4 py-4" colSpan={5}>
                            <div className="grid gap-3 lg:grid-cols-[1fr_260px]">
                              <div className="h-52 rounded-md border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-2">
                                <ResponsiveContainer width="100%" height="100%">
                                  <LineChart data={history}>
                                    <CartesianGrid strokeDasharray="3 3" />
                                    <XAxis
                                      dataKey="timestamp"
                                      type="number"
                                      domain={["dataMin", "dataMax"]}
                                      tickFormatter={formatTimeTick}
                                      minTickGap={16}
                                    />
                                    <YAxis allowDecimals={false} />
                                    <Tooltip
                                      labelFormatter={(value) => new Date(Number(value)).toLocaleString()}
                                      formatter={(value: number) => [value.toFixed(0), "queue depth"]}
                                    />
                                    <Line
                                      type="monotone"
                                      dataKey="queueDepth"
                                      stroke="#1d4ed8"
                                      strokeWidth={2}
                                      dot={false}
                                    />
                                  </LineChart>
                                </ResponsiveContainer>
                              </div>
                              <aside className="space-y-2 rounded-md border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-3 text-sm">
                                <p className="font-semibold">Lane Details</p>
                                <p className="text-[var(--ui-muted)]">
                                  Queue depth points: {history.length}
                                </p>
                                <p className="text-[var(--ui-muted)]">
                                  Worker count: {formatNumber(lane.workers)}
                                </p>
                                <p className="text-[var(--ui-muted)]">
                                  Throughput: {lane.throughput_per_sec.toFixed(2)} tasks/s
                                </p>
                                <p className="text-[var(--ui-muted)]">
                                  Error rate: {(lane.error_rate * 100).toFixed(2)}%
                                </p>
                              </aside>
                            </div>
                          </td>
                        </tr>
                      ) : null}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>

      <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
          Cluster Information
        </h2>
        {clusterNodes.length === 0 ? (
          <p className="mt-3 text-sm text-[var(--ui-muted)]">Standalone mode</p>
        ) : (
          <div className="mt-3 overflow-x-auto rounded-lg border border-[var(--ui-border)]">
            <table className="min-w-full divide-y divide-[var(--ui-border)] text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-[var(--ui-muted)]">
                  <th className="px-4 py-3">Node ID</th>
                  <th className="px-4 py-3">Address</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Last Heartbeat</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--ui-border)]">
                {clusterNodes.map((node) => (
                  <tr key={`${node.id}-${node.address}`}>
                    <td className="px-4 py-3 font-mono text-xs">{node.id}</td>
                    <td className="px-4 py-3">{node.address}</td>
                    <td className="px-4 py-3 capitalize">{node.status}</td>
                    <td className="px-4 py-3">{node.lastHeartbeat}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </section>
  );
}
