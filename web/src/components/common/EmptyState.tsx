type EmptyStateProps = {
  title: string;
  description: string;
};

export function EmptyState({ title, description }: EmptyStateProps) {
  return (
    <section className="grid place-items-center rounded-xl border border-dashed border-[var(--ui-border)] bg-[var(--ui-panel)] px-6 py-12 text-center">
      <div className="mb-4 grid h-16 w-16 place-items-center rounded-full border border-[var(--ui-border)] text-xl">
        []
      </div>
      <h3 className="text-lg font-semibold">{title}</h3>
      <p className="mt-2 max-w-md text-sm text-[var(--ui-muted)]">{description}</p>
    </section>
  );
}

