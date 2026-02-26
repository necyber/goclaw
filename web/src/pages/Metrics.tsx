import { EmptyState } from "../components/common/EmptyState";

export function MetricsPage() {
  return (
    <section className="space-y-4">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Metrics</h1>
        <p className="mt-1 text-sm text-[var(--ui-muted)]">
          Throughput, queue depth, and system runtime indicators.
        </p>
      </header>
      <EmptyState
        title="No metric panels yet"
        description="Metrics charts are implemented in a later section of this change."
      />
    </section>
  );
}

