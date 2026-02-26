import type { PrometheusSample } from "../types/api";
import { parsePrometheusText } from "../lib/prometheus";

export async function fetchMetrics(signal?: AbortSignal): Promise<PrometheusSample[]> {
  const response = await fetch("/metrics", {
    method: "GET",
    headers: { Accept: "text/plain" },
    signal
  });
  if (!response.ok) {
    throw new Error(`Metrics request failed with ${response.status}`);
  }
  return parsePrometheusText(await response.text());
}
