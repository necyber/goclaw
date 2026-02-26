import { useEffect } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";

import { AppShell } from "./layouts/AppShell";
import { useThemeStore } from "./stores/theme";
import { AdminPage } from "./pages/Admin";
import { DashboardPage } from "./pages/Dashboard";
import { MetricsPage } from "./pages/Metrics";
import { NotFoundPage } from "./pages/NotFound";
import { WorkflowDetailPage } from "./pages/WorkflowDetail";
import { WorkflowsPage } from "./pages/Workflows";

function ThemeSync() {
  const initialized = useThemeStore((state) => state.initialized);
  const theme = useThemeStore((state) => state.theme);
  const initialize = useThemeStore((state) => state.initialize);

  useEffect(() => {
    if (!initialized) {
      initialize();
    }
  }, [initialized, initialize]);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
    document.documentElement.style.colorScheme = theme;
  }, [theme]);

  return null;
}

export default function App() {
  return (
    <BrowserRouter basename="/ui">
      <ThemeSync />
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<DashboardPage />} />
          <Route path="workflows" element={<WorkflowsPage />} />
          <Route path="workflows/:id" element={<WorkflowDetailPage />} />
          <Route path="metrics" element={<MetricsPage />} />
          <Route path="admin" element={<AdminPage />} />
          <Route path="dashboard" element={<Navigate to="/" replace />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

