import { getUIBasePath, initializeUIBasePath } from "./uiBasePath";

describe("uiBasePath runtime helpers", () => {
  afterEach(() => {
    delete window.__GOCLAW_UI_BASE_PATH__;
  });

  it("uses configured runtime base path when provided", () => {
    window.__GOCLAW_UI_BASE_PATH__ = "/dashboard/";

    initializeUIBasePath("https://example.com/ignored/assets/index.js");

    expect(getUIBasePath()).toBe("/dashboard");
  });

  it("infers custom base path from production assets url when runtime value is placeholder", () => {
    window.__GOCLAW_UI_BASE_PATH__ = "__GOCLAW_UI_BASE_PATH_VALUE__";

    initializeUIBasePath("https://example.com/ops/ui/assets/index.js");

    expect(getUIBasePath()).toBe("/ops/ui");
  });

  it("infers custom base path from dev module url", () => {
    window.__GOCLAW_UI_BASE_PATH__ = "__GOCLAW_UI_BASE_PATH_VALUE__";

    initializeUIBasePath("http://localhost:8080/dashboard/src/main.tsx");

    expect(getUIBasePath()).toBe("/dashboard");
  });

  it("falls back to default base path when runtime value is missing and module url is root", () => {
    delete window.__GOCLAW_UI_BASE_PATH__;

    initializeUIBasePath("http://localhost:8080/src/main.tsx");

    expect(getUIBasePath()).toBe("/ui");
  });
});
