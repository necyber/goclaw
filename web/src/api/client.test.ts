import { ApiError, requestJSON } from "./client";

describe("requestJSON", () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends query params and parses JSON response", async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      })
    );

    const response = await requestJSON<{ ok: boolean }>("/api/v1/workflows", {
      query: { limit: 20, search: "demo" }
    });

    expect(response.ok).toBe(true);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/v1/workflows");
    expect(url).toContain("limit=20");
    expect(url).toContain("search=demo");
    expect(init.method).toBe("GET");
  });

  it("returns undefined for 204 responses", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));

    const response = await requestJSON<void>("/api/v1/empty");
    expect(response).toBeUndefined();
  });

  it("throws ApiError for failed requests", async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          error: { message: "invalid payload", code: "INVALID", request_id: "req-123" }
        }),
        {
          status: 400,
          headers: { "Content-Type": "application/json" }
        }
      )
    );

    await expect(requestJSON("/api/v1/workflows", { method: "POST", body: {} })).rejects.toMatchObject({
      name: "ApiError",
      message: "invalid payload",
      status: 400,
      code: "INVALID",
      requestId: "req-123"
    } satisfies Partial<ApiError>);
  });
});
