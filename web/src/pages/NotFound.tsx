import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <section className="grid min-h-[50vh] place-items-center rounded-xl border border-dashed border-[var(--ui-border)] bg-[var(--ui-panel)] p-8">
      <div className="text-center">
        <p className="text-xs uppercase tracking-[0.2em] text-[var(--ui-muted)]">404</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">Page Not Found</h1>
        <p className="mt-3 text-sm text-[var(--ui-muted)]">
          The requested route does not exist under the current UI build.
        </p>
        <Link
          to="/"
          className="mt-6 inline-flex rounded-md bg-[var(--ui-accent)] px-4 py-2 text-sm font-semibold text-[var(--ui-accent-fg)]"
        >
          Back to Dashboard
        </Link>
      </div>
    </section>
  );
}
