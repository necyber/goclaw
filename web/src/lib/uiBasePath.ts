const DEFAULT_UI_BASE_PATH = "/ui";
const RUNTIME_BASE_PATH_PLACEHOLDER = "__GOCLAW_UI_BASE_PATH_VALUE__";

function normalizeUIBasePath(basePath: string): string {
  const trimmed = basePath.trim();
  if (!trimmed) {
    return DEFAULT_UI_BASE_PATH;
  }
  const withLeadingSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  const withoutTrailingSlash =
    withLeadingSlash.length > 1 ? withLeadingSlash.replace(/\/+$/, "") : withLeadingSlash;
  return withoutTrailingSlash || DEFAULT_UI_BASE_PATH;
}

function isPlaceholder(basePath: string): boolean {
  const normalized = normalizeUIBasePath(basePath);
  return normalized.includes(RUNTIME_BASE_PATH_PLACEHOLDER);
}

function inferBasePathFromModuleURL(moduleURL: string): string | null {
  try {
    const parsed = new URL(moduleURL);
    const path = parsed.pathname;

    const assetsIndex = path.indexOf("/assets/");
    if (assetsIndex > 0) {
      return normalizeUIBasePath(path.slice(0, assetsIndex));
    }

    const srcIndex = path.indexOf("/src/");
    if (srcIndex > 0) {
      return normalizeUIBasePath(path.slice(0, srcIndex));
    }
  } catch {
    // Ignore invalid module URL values and fallback.
  }
  return null;
}

export function initializeUIBasePath(moduleURL: string) {
  if (typeof window === "undefined") {
    return;
  }

  const configured = window.__GOCLAW_UI_BASE_PATH__;
  if (typeof configured === "string" && configured.trim() !== "" && !isPlaceholder(configured)) {
    window.__GOCLAW_UI_BASE_PATH__ = normalizeUIBasePath(configured);
    return;
  }

  const inferred = inferBasePathFromModuleURL(moduleURL);
  window.__GOCLAW_UI_BASE_PATH__ = inferred || DEFAULT_UI_BASE_PATH;
}

export function getUIBasePath(): string {
  if (typeof window === "undefined") {
    return DEFAULT_UI_BASE_PATH;
  }

  const configured = window.__GOCLAW_UI_BASE_PATH__;
  if (typeof configured === "string" && configured.trim() !== "" && !isPlaceholder(configured)) {
    return normalizeUIBasePath(configured);
  }

  return DEFAULT_UI_BASE_PATH;
}
