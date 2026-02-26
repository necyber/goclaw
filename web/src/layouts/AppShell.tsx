import { useEffect, useMemo, useState } from "react";
import { NavLink, Outlet } from "react-router-dom";

import { ConnectionStatus } from "../components/ConnectionStatus";
import { useThemeStore } from "../stores/theme";

type NavItem = {
  path: string;
  label: string;
  icon: string;
};

const NAV_ITEMS: NavItem[] = [
  { path: "/", label: "Dashboard", icon: "DB" },
  { path: "/workflows", label: "Workflows", icon: "WF" },
  { path: "/metrics", label: "Metrics", icon: "MX" },
  { path: "/admin", label: "Admin", icon: "AD" }
];

function getAutoCollapsed(width: number) {
  if (width >= 1024 && width < 1280) {
    return true;
  }
  return false;
}

export function AppShell() {
  const toggleTheme = useThemeStore((state) => state.toggleTheme);
  const theme = useThemeStore((state) => state.theme);
  const [collapsed, setCollapsed] = useState<boolean>(() => {
    if (typeof window === "undefined") {
      return false;
    }
    return getAutoCollapsed(window.innerWidth);
  });

  useEffect(() => {
    const onResize = () => {
      setCollapsed(getAutoCollapsed(window.innerWidth));
    };
    window.addEventListener("resize", onResize);
    onResize();
    return () => window.removeEventListener("resize", onResize);
  }, []);

  const sideClassName = useMemo(
    () =>
      collapsed
        ? "w-18 shrink-0 border-r border-[var(--ui-border)] bg-[var(--ui-panel)]"
        : "w-64 shrink-0 border-r border-[var(--ui-border)] bg-[var(--ui-panel)]",
    [collapsed]
  );

  return (
    <div className="min-h-screen bg-[var(--ui-bg)] text-[var(--ui-fg)]">
      <header className="sticky top-0 z-20 border-b border-[var(--ui-border)] bg-[var(--ui-panel)]/95 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-[1600px] items-center justify-between px-4 lg:px-6">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => setCollapsed((value) => !value)}
              className="rounded-md border border-[var(--ui-border)] px-2 py-1 text-xs font-semibold hover:bg-black/5 dark:hover:bg-white/5"
              aria-label="Toggle sidebar"
            >
              {collapsed ? ">>" : "<<"}
            </button>
            <div className="flex items-center gap-2">
              <div className="grid h-8 w-8 place-items-center rounded bg-[var(--ui-accent)] text-[var(--ui-accent-fg)]">
                GC
              </div>
              <div>
                <p className="text-sm font-semibold leading-tight">GoClaw</p>
                <p className="text-xs text-[var(--ui-muted)]">Web Console</p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <ConnectionStatus />
            <button
              type="button"
              onClick={toggleTheme}
              className="rounded-md border border-[var(--ui-border)] px-3 py-1.5 text-xs font-semibold hover:bg-black/5 dark:hover:bg-white/5"
            >
              {theme === "dark" ? "Light" : "Dark"}
            </button>
          </div>
        </div>
      </header>

      <div className="mx-auto flex max-w-[1600px]">
        <aside className={sideClassName}>
          <nav className="space-y-1 p-2">
            {NAV_ITEMS.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                end={item.path === "/"}
                className={({ isActive }) =>
                  [
                    "group flex items-center gap-3 rounded-md px-3 py-2 text-sm transition",
                    isActive
                      ? "bg-[var(--ui-accent)] text-[var(--ui-accent-fg)]"
                      : "text-[var(--ui-muted)] hover:bg-black/5 hover:text-[var(--ui-fg)] dark:hover:bg-white/5"
                  ].join(" ")
                }
              >
                <span className="inline-flex h-6 w-6 items-center justify-center rounded border border-current/30 text-[10px] font-bold">
                  {item.icon}
                </span>
                {!collapsed && <span className="font-medium">{item.label}</span>}
              </NavLink>
            ))}
          </nav>
        </aside>

        <main className="min-w-0 flex-1 p-4 lg:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
