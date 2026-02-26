import type { PrometheusSample } from "../types/api";

export function parsePrometheusText(input: string): PrometheusSample[] {
  const samples: PrometheusSample[] = [];

  for (const line of input.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const spaceIndex = trimmed.lastIndexOf(" ");
    if (spaceIndex <= 0 || spaceIndex === trimmed.length - 1) {
      continue;
    }

    const keyPart = trimmed.slice(0, spaceIndex);
    const valuePart = trimmed.slice(spaceIndex + 1);
    const value = Number(valuePart);
    if (Number.isNaN(value)) {
      continue;
    }

    const braceStart = keyPart.indexOf("{");
    const braceEnd = keyPart.lastIndexOf("}");
    if (braceStart === -1 || braceEnd === -1 || braceEnd < braceStart) {
      samples.push({ metric: keyPart, labels: {}, value });
      continue;
    }

    const metric = keyPart.slice(0, braceStart);
    const labelsRaw = keyPart.slice(braceStart + 1, braceEnd);
    const labels: Record<string, string> = {};

    for (const pair of labelsRaw.split(",")) {
      const [rawKey, rawValue] = pair.split("=");
      if (!rawKey || !rawValue) {
        continue;
      }
      const key = rawKey.trim();
      const unquoted = rawValue.trim().replace(/^"/, "").replace(/"$/, "");
      labels[key] = unquoted.replace(/\\"/g, "\"");
    }

    samples.push({ metric, labels, value });
  }

  return samples;
}

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

