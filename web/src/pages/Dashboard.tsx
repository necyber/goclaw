export function DashboardPage() {
  return (
    <section className="space-y-4">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="mt-1 text-sm text-[var(--ui-muted)]">
          Overview and live signals for GoClaw runtime.
        </p>
      </header>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {[
          "Active Workflows",
          "Completed (24h)",
          "Failed (24h)",
          "Avg Duration"
        ].map((label) => (
          <article
            key={label}
            className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4"
          >
            <p className="text-xs font-medium uppercase tracking-wide text-[var(--ui-muted)]">{label}</p>
            <p className="mt-2 text-2xl font-semibold">-</p>
          </article>
        ))}
      </div>
    </section>
  );
}

