type ResourceGaugesProps = {
  memoryUsedBytes: number | null;
  memoryPercent: number | null;
  goroutines: number | null;
  goroutinePercent: number | null;
  cpuPercent: number | null;
};

function formatBytes(bytes: number | null): string {
  if (bytes === null || Number.isNaN(bytes)) {
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

function formatPercent(value: number | null) {
  if (value === null || Number.isNaN(value)) {
    return "-";
  }
  return `${value.toFixed(1)}%`;
}

function gaugeClass(percent: number | null) {
  if (percent === null) {
    return "bg-slate-300 dark:bg-slate-600";
  }
  if (percent >= 90) {
    return "bg-red-500";
  }
  if (percent >= 70) {
    return "bg-amber-500";
  }
  return "bg-emerald-500";
}

function GaugeCard({
  title,
  subtitle,
  value,
  percent,
  percentageLabel,
}: {
  title: string;
  subtitle: string;
  value: string;
  percent: number | null;
  percentageLabel?: string;
}) {
  const width = percent === null ? 0 : Math.max(0, Math.min(percent, 100));
  return (
    <article className="rounded-lg border border-[var(--ui-border)] bg-[var(--ui-bg)]/40 p-3">
      <p className="text-xs font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
        {title}
      </p>
      <p className="mt-1 text-lg font-semibold">{value}</p>
      <p className="text-xs text-[var(--ui-muted)]">{subtitle}</p>
      <div className="mt-3 h-2 overflow-hidden rounded-full bg-black/10 dark:bg-white/10">
        <div className={`h-full ${gaugeClass(percent)}`} style={{ width: `${width}%` }} />
      </div>
      <p className="mt-1 text-right text-xs text-[var(--ui-muted)]">
        {percentageLabel ?? formatPercent(percent)}
      </p>
    </article>
  );
}

export function ResourceGauges({
  memoryUsedBytes,
  memoryPercent,
  goroutines,
  goroutinePercent,
  cpuPercent,
}: ResourceGaugesProps) {
  return (
    <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
      <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
        Runtime Resources
      </h3>
      <div className="grid gap-3 lg:grid-cols-3">
        <GaugeCard
          title="Memory"
          value={formatBytes(memoryUsedBytes)}
          subtitle="Heap usage"
          percent={memoryPercent}
        />
        <GaugeCard
          title="Goroutines"
          value={goroutines === null ? "-" : `${Math.round(goroutines)}`}
          subtitle="Runtime goroutine count"
          percent={goroutinePercent}
        />
        <GaugeCard
          title="CPU"
          value={formatPercent(cpuPercent)}
          subtitle="Process CPU utilization"
          percent={cpuPercent}
          percentageLabel={cpuPercent === null ? "-" : `${cpuPercent.toFixed(2)}%`}
        />
      </div>
    </section>
  );
}
