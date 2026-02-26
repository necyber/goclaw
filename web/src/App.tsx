import { useMemo, useState } from "react";

type Theme = "light" | "dark";

export default function App() {
  const preferred = useMemo<Theme>(() => {
    if (typeof window === "undefined") {
      return "light";
    }
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }, []);

  const [theme, setTheme] = useState<Theme>(preferred);

  return (
    <div data-theme={theme} className="min-h-screen bg-[var(--ui-bg)] text-[var(--ui-fg)]">
      <main className="mx-auto max-w-4xl px-6 py-16">
        <h1 className="text-4xl font-semibold tracking-tight">GoClaw Web UI</h1>
        <p className="mt-4 text-base text-[var(--ui-muted)]">
          Week 12 scaffold is ready. Routing, pages, and realtime features will be implemented in
          subsequent tasks.
        </p>
        <div className="mt-8 flex items-center gap-3">
          <button
            type="button"
            onClick={() => setTheme((prev) => (prev === "light" ? "dark" : "light"))}
            className="rounded-md border border-[var(--ui-border)] bg-[var(--ui-panel)] px-4 py-2 text-sm font-medium hover:opacity-90"
          >
            Toggle Theme
          </button>
          <span className="text-sm text-[var(--ui-muted)]">Current: {theme}</span>
        </div>
      </main>
    </div>
  );
}
