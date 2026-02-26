import type { PrometheusSample } from "../types/api";

function parseLabels(raw: string): Record<string, string> {
  const labels: Record<string, string> = {};
  if (!raw) {
    return labels;
  }

  const pairRegex = /([^=,\s]+)\s*=\s*"((?:\\.|[^"\\])*)"/g;
  let match = pairRegex.exec(raw);
  while (match) {
    const key = match[1];
    const value = match[2]
      .replace(/\\n/g, "\n")
      .replace(/\\t/g, "\t")
      .replace(/\\"/g, "\"")
      .replace(/\\\\/g, "\\");
    labels[key] = value;
    match = pairRegex.exec(raw);
  }
  return labels;
}

export function parsePrometheusText(input: string): PrometheusSample[] {
  const samples: PrometheusSample[] = [];

  for (const line of input.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const match = trimmed.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{.*\})?\s+(.+)$/);
    if (!match) {
      continue;
    }

    const metric = match[1];
    const labelsRaw = match[2];
    const rawValue = match[3];
    const value = Number(rawValue);
    if (Number.isNaN(value)) {
      continue;
    }

    const labels = labelsRaw ? parseLabels(labelsRaw.slice(1, -1)) : {};
    samples.push({ metric, labels, value });
  }

  return samples;
}
