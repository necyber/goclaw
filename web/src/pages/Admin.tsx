import { EmptyState } from "../components/common/EmptyState";

export function AdminPage() {
  return (
    <section className="space-y-4">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Admin</h1>
        <p className="mt-1 text-sm text-[var(--ui-muted)]">
          Engine controls, lane diagnostics, and export tools.
        </p>
      </header>
      <EmptyState
        title="Admin controls pending"
        description="Admin actions and lane management will be wired in a subsequent section."
      />
    </section>
  );
}

