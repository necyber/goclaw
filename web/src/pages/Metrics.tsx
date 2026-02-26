import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { getEngineStatus, getLaneStats } from "../api/admin";
import { fetchMetrics } from "../api/metrics";
import { DurationHistogram, type DurationBucketPoint } from "../components/DurationHistogram";
import {
  ErrorRateChart,
  type ErrorRatePoint,
  type ErrorRateVisibility,
} from "../components/ErrorRateChart";
import { QueueDepthChart, type QueueDepthPoint } from "../components/QueueDepthChart";
import { ResourceGauges } from "../components/ResourceGauges";
import {
  ThroughputChart,
  type ThroughputPoint,
  type ThroughputVisibility,
} from "../components/ThroughputChart";
import { EmptyState } from "../components/common/EmptyState";
import { ErrorState } from "../components/common/ErrorState";
import { Loading } from "../components/common/Loading";
import type { EngineStatus, LaneStats, PrometheusSample } from "../types/api";

type MetricsSnapshot = {
  timestamp: number;
  samples: PrometheusSample[];
  laneStats: LaneStats[];
  engineStatus: EngineStatus | null;
};

type TimeRange = "15m" | "1h" | "6h" | "24h";

const ONE_MINUTE_MS = 60 * 1000;
const ONE_HOUR_MS = 60 * ONE_MINUTE_MS;
const HISTORY_RETENTION_MS = 24 * ONE_HOUR_MS + 5 * ONE_MINUTE_MS;
const MAX_HISTORY_POINTS = 24 * 60 * 6 + 30;
const ACTIVE_STATUSES = new Set(["pending", "scheduled", "running"]);

const TIME_RANGES: Array<{ value: TimeRange; label: string; durationMs: number }> = [
  { value: "15m", label: "15m", durationMs: 15 * ONE_MINUTE_MS },
  { value: "1h", label: "1h", durationMs: ONE_HOUR_MS },
  { value: "6h", label: "6h", durationMs: 6 * ONE_HOUR_MS },
  { value: "24h", label: "24h", durationMs: 24 * ONE_HOUR_MS },
];

type LabelMatcher = (labels: Record<string, string>) => boolean;

function readMetric(
  samples: PrometheusSample[],
  metric: string,
  matcher?: LabelMatcher
): number | null {
  let found = false;
  let value = 0;

  for (const sample of samples) {
    if (sample.metric !== metric) {
      continue;
    }
    if (matcher && !matcher(sample.labels)) {
      continue;
    }
    found = true;
    value += sample.value;
  }

  return found ? value : null;
}

function counterDelta(
  snapshots: MetricsSnapshot[],
  metric: string,
  matcher?: LabelMatcher
): number {
  if (snapshots.length < 2) {
    return 0;
  }
  const first = readMetric(snapshots[0].samples, metric, matcher) ?? 0;
  const last = readMetric(snapshots[snapshots.length - 1].samples, metric, matcher) ?? 0;
  return Math.max(last - first, 0);
}

function formatBound(seconds: number): string {
  if (seconds < 1) {
    return `${Math.round(seconds * 1000)}ms`;
  }
  if (seconds < 60) {
    return `${seconds.toFixed(seconds < 10 ? 1 : 0)}s`;
  }
  return `${(seconds / 60).toFixed(1)}m`;
}

function formatBytes(bytes: number | null): string {
  if (bytes === null) {
    return "-";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[index]}`;
}

function formatCount(value: number): string {
  return new Intl.NumberFormat().format(Math.round(value));
}

function formatDuration(seconds: number | null): string {
  if (seconds === null || Number.isNaN(seconds)) {
    return "-";
  }
  if (seconds < 1) {
    return `${Math.round(seconds * 1000)} ms`;
  }
  if (seconds < 60) {
    return `${seconds.toFixed(2)} s`;
  }
  return `${(seconds / 60).toFixed(1)} min`;
}

function toErrorText(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return "unknown error";
}

function filterSnapshots(snapshots: MetricsSnapshot[], windowMs: number): MetricsSnapshot[] {
  if (snapshots.length === 0) {
    return [];
  }
  const end = snapshots[snapshots.length - 1].timestamp;
  const start = end - windowMs;
  return snapshots.filter((snapshot) => snapshot.timestamp >= start);
}

function downsample<T>(items: T[], maxPoints: number): T[] {
  if (items.length <= maxPoints) {
    return items;
  }
  const step = Math.ceil(items.length / maxPoints);
  return items.filter((_, index) => index % step === 0 || index === items.length - 1);
}

function getByLabel(
  samples: PrometheusSample[],
  metric: string,
  labelName: string
): Map<string, number> {
  const values = new Map<string, number>();
  for (const sample of samples) {
    if (sample.metric !== metric) {
      continue;
    }
    const labelValue = sample.labels[labelName];
    if (!labelValue) {
      continue;
    }
    values.set(labelValue, (values.get(labelValue) ?? 0) + sample.value);
  }
  return values;
}

function buildThroughputData(snapshots: MetricsSnapshot[]): ThroughputPoint[] {
  if (snapshots.length === 0) {
    return [];
  }

  const endMinute =
    Math.floor(snapshots[snapshots.length - 1].timestamp / ONE_MINUTE_MS) * ONE_MINUTE_MS;
  const startMinute = endMinute - (60 - 1) * ONE_MINUTE_MS;
  const buckets = new Map<number, { submitted: number; completed: number }>();

  for (let index = 1; index < snapshots.length; index += 1) {
    const current = snapshots[index];
    const previous = snapshots[index - 1];

    const submittedDelta = Math.max(
      (readMetric(current.samples, "workflow_submissions_total") ?? 0) -
        (readMetric(previous.samples, "workflow_submissions_total") ?? 0),
      0
    );
    const completedDelta = Math.max(
      (readMetric(
        current.samples,
        "workflow_submissions_total",
        (labels) => labels.status === "completed"
      ) ?? 0) -
        (readMetric(
          previous.samples,
          "workflow_submissions_total",
          (labels) => labels.status === "completed"
        ) ?? 0),
      0
    );

    const minute = Math.floor(current.timestamp / ONE_MINUTE_MS) * ONE_MINUTE_MS;
    if (minute < startMinute) {
      continue;
    }
    const bucket = buckets.get(minute) ?? { submitted: 0, completed: 0 };
    bucket.submitted += submittedDelta;
    bucket.completed += completedDelta;
    buckets.set(minute, bucket);
  }

  const output: ThroughputPoint[] = [];
  for (let ts = startMinute; ts <= endMinute; ts += ONE_MINUTE_MS) {
    const bucket = buckets.get(ts) ?? { submitted: 0, completed: 0 };
    output.push({
      timestamp: ts,
      submitted: Number(bucket.submitted.toFixed(2)),
      completed: Number(bucket.completed.toFixed(2)),
    });
  }
  return output;
}

function buildDurationHistogram(snapshots: MetricsSnapshot[]): DurationBucketPoint[] {
  if (snapshots.length === 0) {
    return [];
  }

  const first = snapshots[0];
  const last = snapshots[snapshots.length - 1];
  const firstBuckets = getByLabel(first.samples, "task_duration_seconds_bucket", "le");
  const lastBuckets = getByLabel(last.samples, "task_duration_seconds_bucket", "le");
  const labels = Array.from(lastBuckets.keys()).sort((left, right) => {
    const leftValue = left === "+Inf" ? Number.POSITIVE_INFINITY : Number(left);
    const rightValue = right === "+Inf" ? Number.POSITIVE_INFINITY : Number(right);
    return leftValue - rightValue;
  });

  let previousUpper = 0;
  let previousCumulative = 0;
  const output: DurationBucketPoint[] = [];

  for (const label of labels) {
    const upper = label === "+Inf" ? Number.POSITIVE_INFINITY : Number(label);
    if (Number.isNaN(upper)) {
      continue;
    }

    const cumulative = Math.max((lastBuckets.get(label) ?? 0) - (firstBuckets.get(label) ?? 0), 0);
    const count = Math.max(cumulative - previousCumulative, 0);
    const bucketLabel =
      upper === Number.POSITIVE_INFINITY
        ? `>${formatBound(previousUpper)}`
        : `${formatBound(previousUpper)}-${formatBound(upper)}`;

    output.push({
      bucket: bucketLabel,
      count: Math.round(count),
    });

    previousCumulative = cumulative;
    if (upper !== Number.POSITIVE_INFINITY) {
      previousUpper = upper;
    }
  }

  return output;
}

function collectLanes(snapshots: MetricsSnapshot[]): string[] {
  const names = new Set<string>();
  for (const snapshot of snapshots) {
    for (const lane of snapshot.laneStats) {
      names.add(lane.name);
    }
    for (const sample of snapshot.samples) {
      if (sample.metric !== "lane_queue_depth" && sample.metric !== "redis_lane_queue_depth") {
        continue;
      }
      const laneName = sample.labels.lane_name;
      if (laneName) {
        names.add(laneName);
      }
    }
  }
  return Array.from(names).sort();
}

function readLaneDepth(snapshot: MetricsSnapshot, lane: string): number {
  const laneStat = snapshot.laneStats.find((item) => item.name === lane);
  if (laneStat) {
    return laneStat.queue_depth;
  }

  const directDepth = readMetric(
    snapshot.samples,
    "lane_queue_depth",
    (labels) => labels.lane_name === lane
  );
  const redisDepth = readMetric(
    snapshot.samples,
    "redis_lane_queue_depth",
    (labels) => labels.lane_name === lane
  );
  return (directDepth ?? 0) + (redisDepth ?? 0);
}

function buildQueueDepthData(snapshots: MetricsSnapshot[], lanes: string[]): QueueDepthPoint[] {
  return snapshots.map((snapshot) => {
    const point: QueueDepthPoint = { timestamp: snapshot.timestamp };
    for (const lane of lanes) {
      point[lane] = readLaneDepth(snapshot, lane);
    }
    return point;
  });
}

function buildErrorRateData(snapshots: MetricsSnapshot[]): {
  points: ErrorRatePoint[];
  spikes: number[];
} {
  const points: ErrorRatePoint[] = [];
  const spikes: number[] = [];

  for (let index = 1; index < snapshots.length; index += 1) {
    const current = snapshots[index];
    const previous = snapshots[index - 1];

    const workflowTotal = Math.max(
      (readMetric(current.samples, "workflow_submissions_total") ?? 0) -
        (readMetric(previous.samples, "workflow_submissions_total") ?? 0),
      0
    );
    const workflowFailed = Math.max(
      (readMetric(
        current.samples,
        "workflow_submissions_total",
        (labels) => labels.status === "failed"
      ) ?? 0) -
        (readMetric(
          previous.samples,
          "workflow_submissions_total",
          (labels) => labels.status === "failed"
        ) ?? 0),
      0
    );
    const taskTotal = Math.max(
      (readMetric(current.samples, "task_executions_total") ?? 0) -
        (readMetric(previous.samples, "task_executions_total") ?? 0),
      0
    );
    const taskFailed = Math.max(
      (readMetric(
        current.samples,
        "task_executions_total",
        (labels) => labels.status === "failed"
      ) ?? 0) -
        (readMetric(
          previous.samples,
          "task_executions_total",
          (labels) => labels.status === "failed"
        ) ?? 0),
      0
    );

    const workflowErrorRate = workflowTotal > 0 ? (workflowFailed / workflowTotal) * 100 : 0;
    const taskErrorRate = taskTotal > 0 ? (taskFailed / taskTotal) * 100 : 0;

    points.push({
      timestamp: current.timestamp,
      workflowErrorRate,
      taskErrorRate,
    });

    if (workflowErrorRate > 10 || taskErrorRate > 10) {
      spikes.push(current.timestamp);
    }
  }

  return { points, spikes };
}

function averageWorkflowDurationSeconds(snapshots: MetricsSnapshot[]): number | null {
  if (snapshots.length === 0) {
    return null;
  }

  const deltaSum = counterDelta(snapshots, "workflow_duration_seconds_sum");
  const deltaCount = counterDelta(snapshots, "workflow_duration_seconds_count");
  if (deltaCount > 0) {
    return deltaSum / deltaCount;
  }

  const current = snapshots[snapshots.length - 1];
  const totalSum = readMetric(current.samples, "workflow_duration_seconds_sum") ?? 0;
  const totalCount = readMetric(current.samples, "workflow_duration_seconds_count") ?? 0;
  if (totalCount <= 0) {
    return null;
  }
  return totalSum / totalCount;
}

function resolveGoroutines(snapshot: MetricsSnapshot): number | null {
  return snapshot.engineStatus?.goroutines ?? readMetric(snapshot.samples, "go_goroutines");
}

function resolveMemoryBytes(snapshot: MetricsSnapshot): number | null {
  if (snapshot.engineStatus?.memory_bytes !== undefined) {
    return snapshot.engineStatus.memory_bytes;
  }
  return (
    readMetric(snapshot.samples, "go_memstats_heap_inuse_bytes") ??
    readMetric(snapshot.samples, "go_memstats_heap_alloc_bytes")
  );
}

function resolveMemoryCapacity(snapshot: MetricsSnapshot): number | null {
  return (
    readMetric(snapshot.samples, "go_memstats_heap_sys_bytes") ??
    readMetric(snapshot.samples, "process_resident_memory_bytes")
  );
}

function resolveCPUPercent(snapshots: MetricsSnapshot[]): number | null {
  if (snapshots.length < 2) {
    return null;
  }

  const current = snapshots[snapshots.length - 1];
  const previous = snapshots[snapshots.length - 2];
  const currentCPU = readMetric(current.samples, "process_cpu_seconds_total");
  const previousCPU = readMetric(previous.samples, "process_cpu_seconds_total");
  if (currentCPU === null || previousCPU === null) {
    return null;
  }

  const elapsedSeconds = (current.timestamp - previous.timestamp) / 1000;
  if (elapsedSeconds <= 0) {
    return null;
  }

  const cpuPercent = ((currentCPU - previousCPU) / elapsedSeconds) * 100;
  if (!Number.isFinite(cpuPercent)) {
    return null;
  }
  return Math.max(0, Math.min(cpuPercent, 100));
}

export function MetricsPage() {
  const mountedRef = useRef(true);
  const [snapshots, setSnapshots] = useState<MetricsSnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [metricsError, setMetricsError] = useState<string | null>(null);
  const [partialWarning, setPartialWarning] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>("1h");
  const [throughputVisible, setThroughputVisible] = useState<ThroughputVisibility>({
    submitted: true,
    completed: true,
  });
  const [errorRateVisible, setErrorRateVisible] = useState<ErrorRateVisibility>({
    workflowErrorRate: true,
    taskErrorRate: true,
  });
  const [queueVisible, setQueueVisible] = useState<Record<string, boolean>>({});

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const refreshMetrics = useCallback(async () => {
    const [metricsResult, engineResult, lanesResult] = await Promise.allSettled([
      fetchMetrics(),
      getEngineStatus(),
      getLaneStats(),
    ]);

    if (!mountedRef.current) {
      return;
    }

    if (metricsResult.status !== "fulfilled") {
      setMetricsError(`Metrics unavailable: ${toErrorText(metricsResult.reason)}`);
      setLoading(false);
      return;
    }

    const now = Date.now();
    const snapshot: MetricsSnapshot = {
      timestamp: now,
      samples: metricsResult.value,
      laneStats: lanesResult.status === "fulfilled" ? lanesResult.value : [],
      engineStatus: engineResult.status === "fulfilled" ? engineResult.value : null,
    };

    setSnapshots((previous) => {
      const retained = previous.filter((item) => item.timestamp >= now - HISTORY_RETENTION_MS);
      const next = [...retained, snapshot];
      if (next.length > MAX_HISTORY_POINTS) {
        return next.slice(next.length - MAX_HISTORY_POINTS);
      }
      return next;
    });

    setMetricsError(null);
    if (engineResult.status === "rejected" || lanesResult.status === "rejected") {
      const reasons = [
        engineResult.status === "rejected" ? "engine status" : null,
        lanesResult.status === "rejected" ? "lane stats" : null,
      ]
        .filter((item): item is string => Boolean(item))
        .join(" + ");
      setPartialWarning(`Metrics loaded, but ${reasons} are unavailable.`);
    } else {
      setPartialWarning(null);
    }

    setLoading(false);
  }, []);

  useEffect(() => {
    void refreshMetrics();
    const timer = window.setInterval(() => {
      void refreshMetrics();
    }, 10_000);

    return () => window.clearInterval(timer);
  }, [refreshMetrics]);

  const oneHourHistory = useMemo(() => filterSnapshots(snapshots, ONE_HOUR_MS), [snapshots]);
  const rangeMs = TIME_RANGES.find((item) => item.value === timeRange)?.durationMs ?? ONE_HOUR_MS;
  const selectedHistory = useMemo(() => filterSnapshots(snapshots, rangeMs), [snapshots, rangeMs]);
  const selectedHistoryForChart = useMemo(
    () => downsample(selectedHistory, 240),
    [selectedHistory]
  );
  const history24h = useMemo(() => filterSnapshots(snapshots, 24 * ONE_HOUR_MS), [snapshots]);

  const throughputData = useMemo(() => buildThroughputData(oneHourHistory), [oneHourHistory]);
  const durationHistogram = useMemo(
    () => buildDurationHistogram(selectedHistory),
    [selectedHistory]
  );
  const laneNames = useMemo(() => collectLanes(selectedHistoryForChart), [selectedHistoryForChart]);
  const queueDepthData = useMemo(
    () => buildQueueDepthData(selectedHistoryForChart, laneNames),
    [selectedHistoryForChart, laneNames]
  );
  const errorRate = useMemo(
    () => buildErrorRateData(selectedHistoryForChart),
    [selectedHistoryForChart]
  );

  useEffect(() => {
    setQueueVisible((current) => {
      const next: Record<string, boolean> = {};
      for (const lane of laneNames) {
        next[lane] = current[lane] ?? true;
      }
      return next;
    });
  }, [laneNames]);

  const latest = snapshots.length > 0 ? snapshots[snapshots.length - 1] : null;
  const lastKnownTimestamp = latest?.timestamp ?? null;

  const overview = useMemo(() => {
    if (!latest) {
      return {
        activeWorkflows: 0,
        completed24h: 0,
        failed24h: 0,
        averageDurationSeconds: null as number | null,
      };
    }

    let activeWorkflows = latest.engineStatus?.active_workflows ?? null;
    if (activeWorkflows === null) {
      const activeFromStatus = readMetric(latest.samples, "workflow_active_count", (labels) =>
        ACTIVE_STATUSES.has(labels.status)
      );
      activeWorkflows =
        activeFromStatus !== null && activeFromStatus > 0
          ? activeFromStatus
          : (readMetric(latest.samples, "workflow_active_count") ?? 0);
    }

    return {
      activeWorkflows: activeWorkflows ?? 0,
      completed24h: counterDelta(
        history24h,
        "workflow_submissions_total",
        (labels) => labels.status === "completed"
      ),
      failed24h: counterDelta(
        history24h,
        "workflow_submissions_total",
        (labels) => labels.status === "failed"
      ),
      averageDurationSeconds: averageWorkflowDurationSeconds(history24h),
    };
  }, [history24h, latest]);

  const resources = useMemo(() => {
    if (!latest) {
      return {
        memoryBytes: null as number | null,
        memoryPercent: null as number | null,
        goroutines: null as number | null,
        goroutinePercent: null as number | null,
        cpuPercent: null as number | null,
      };
    }

    const memoryBytes = resolveMemoryBytes(latest);
    const memoryCapacity = resolveMemoryCapacity(latest);
    const memoryPercent =
      memoryBytes !== null && memoryCapacity !== null && memoryCapacity > 0
        ? Math.max(0, Math.min((memoryBytes / memoryCapacity) * 100, 100))
        : null;

    const goroutines = resolveGoroutines(latest);
    const historicalMaxGoroutines = selectedHistory.reduce((max, snapshot) => {
      const value = resolveGoroutines(snapshot);
      return value !== null ? Math.max(max, value) : max;
    }, 0);
    const goroutinePercent =
      goroutines !== null && historicalMaxGoroutines > 0
        ? (goroutines / historicalMaxGoroutines) * 100
        : null;

    return {
      memoryBytes,
      memoryPercent,
      goroutines,
      goroutinePercent,
      cpuPercent: resolveCPUPercent(snapshots),
    };
  }, [latest, selectedHistory, snapshots]);

  const unavailableWithCachedData = Boolean(metricsError && snapshots.length > 0);

  return (
    <section className="space-y-4">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Metrics</h1>
          <p className="mt-1 text-sm text-[var(--ui-muted)]">
            Throughput, queue depth, and runtime indicators.
          </p>
        </div>
        <div className="inline-flex rounded-md border border-[var(--ui-border)] bg-[var(--ui-panel)] p-1">
          {TIME_RANGES.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => setTimeRange(option.value)}
              className={`rounded px-3 py-1 text-sm ${
                option.value === timeRange
                  ? "bg-[var(--ui-accent)] text-[var(--ui-accent-fg)]"
                  : "text-[var(--ui-muted)]"
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      </header>

      {loading && snapshots.length === 0 ? (
        <Loading label="Loading metrics..." skeletonRows={4} />
      ) : null}

      {metricsError && snapshots.length === 0 ? (
        <ErrorState message={metricsError} onRetry={() => void refreshMetrics()} />
      ) : null}

      {unavailableWithCachedData ? (
        <section className="rounded-xl border border-amber-300/70 bg-amber-50/70 p-3 text-sm text-amber-900 dark:border-amber-500/40 dark:bg-amber-950/25 dark:text-amber-100">
          Metrics unavailable. Showing last known data from{" "}
          {lastKnownTimestamp ? new Date(lastKnownTimestamp).toLocaleString() : "N/A"}.
        </section>
      ) : null}

      {partialWarning ? (
        <section className="rounded-xl border border-blue-300/70 bg-blue-50/70 p-3 text-sm text-blue-900 dark:border-blue-500/40 dark:bg-blue-950/25 dark:text-blue-100">
          {partialWarning}
        </section>
      ) : null}

      {!loading && snapshots.length === 0 ? (
        <EmptyState
          title="No metrics data"
          description="No Prometheus samples are available yet. Submit workflows and refresh metrics."
        />
      ) : null}

      {snapshots.length > 0 ? (
        <>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <article className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
                Active Workflows
              </p>
              <p className="mt-2 text-2xl font-semibold">{formatCount(overview.activeWorkflows)}</p>
            </article>
            <article className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
                Completed (24h)
              </p>
              <p className="mt-2 text-2xl font-semibold">{formatCount(overview.completed24h)}</p>
            </article>
            <article className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
                Failed (24h)
              </p>
              <p className="mt-2 text-2xl font-semibold">{formatCount(overview.failed24h)}</p>
            </article>
            <article className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
                Avg Duration
              </p>
              <p className="mt-2 text-2xl font-semibold">
                {formatDuration(overview.averageDurationSeconds)}
              </p>
            </article>
          </div>

          <ThroughputChart
            data={throughputData}
            visible={throughputVisible}
            onToggle={(key) =>
              setThroughputVisible((current) => ({
                ...current,
                [key]: !current[key],
              }))
            }
          />

          <div className="grid gap-4 xl:grid-cols-2">
            <DurationHistogram data={durationHistogram} />
            <ErrorRateChart
              data={errorRate.points}
              spikes={errorRate.spikes}
              visible={errorRateVisible}
              onToggle={(key) =>
                setErrorRateVisible((current) => ({
                  ...current,
                  [key]: !current[key],
                }))
              }
            />
          </div>

          <QueueDepthChart
            data={queueDepthData}
            lanes={laneNames}
            visible={queueVisible}
            onToggle={(lane) =>
              setQueueVisible((current) => ({
                ...current,
                [lane]: !current[lane],
              }))
            }
          />

          <ResourceGauges
            memoryUsedBytes={resources.memoryBytes}
            memoryPercent={resources.memoryPercent}
            goroutines={resources.goroutines}
            goroutinePercent={resources.goroutinePercent}
            cpuPercent={resources.cpuPercent}
          />

          <footer className="text-xs text-[var(--ui-muted)]">
            Last updated: {lastKnownTimestamp ? new Date(lastKnownTimestamp).toLocaleString() : "-"}
            <span className="mx-2">|</span>
            Selected range: {timeRange}
            <span className="mx-2">|</span>
            Memory: {formatBytes(resources.memoryBytes)}
          </footer>
        </>
      ) : null}
    </section>
  );
}
