import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ThemeMode = "light" | "dark";

type ThemeState = {
  initialized: boolean;
  theme: ThemeMode;
  initialize: () => void;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
};

function detectSystemTheme(): ThemeMode {
  if (typeof window === "undefined") {
    return "light";
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      initialized: false,
      theme: "light",
      initialize: () => {
        const state = get();
        if (state.initialized) {
          return;
        }
        const current = state.theme;
        if (!current) {
          set({ theme: detectSystemTheme(), initialized: true });
          return;
        }
        set({ initialized: true });
      },
      setTheme: (theme) => set({ theme, initialized: true }),
      toggleTheme: () => {
        const nextTheme = get().theme === "light" ? "dark" : "light";
        set({ theme: nextTheme, initialized: true });
      },
    }),
    {
      name: "goclaw-theme",
      version: 1,
      partialize: (state) => ({ theme: state.theme }),
      onRehydrateStorage: () => (state) => {
        if (!state) {
          return;
        }
        if (!state.theme) {
          state.theme = detectSystemTheme();
        }
        state.initialized = true;
      },
    }
  )
);
