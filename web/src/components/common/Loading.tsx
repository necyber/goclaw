type LoadingProps = {
  label?: string;
  skeletonRows?: number;
};

export function Loading({ label = "Loading...", skeletonRows = 3 }: LoadingProps) {
  return (
    <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
      <div className="flex items-center gap-3">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-[var(--ui-border)] border-t-[var(--ui-accent)]" />
        <p className="text-sm text-[var(--ui-muted)]">{label}</p>
      </div>
      <div className="mt-4 space-y-2">
        {Array.from({ length: skeletonRows }, (_, index) => (
          <div
            key={index}
            className="h-3 animate-pulse rounded bg-black/5 dark:bg-white/10"
            style={{ width: `${92 - index * 10}%` }}
          />
        ))}
      </div>
    </section>
  );
}
