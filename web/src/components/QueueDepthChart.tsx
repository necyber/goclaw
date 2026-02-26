import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export type QueueDepthPoint = {
  timestamp: number;
  [lane: string]: number;
};

type QueueDepthChartProps = {
  data: QueueDepthPoint[];
  lanes: string[];
  visible: Record<string, boolean>;
  onToggle: (lane: string) => void;
};

const COLOR_PALETTE = [
  "#1d4ed8",
  "#0f766e",
  "#b45309",
  "#9333ea",
  "#be123c",
  "#0e7490",
  "#4d7c0f",
  "#9a3412",
];

function laneColor(index: number) {
  return COLOR_PALETTE[index % COLOR_PALETTE.length];
}

function formatTimeTick(value: number) {
  return new Date(value).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function QueueDepthChart({ data, lanes, visible, onToggle }: QueueDepthChartProps) {
  return (
    <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
      <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
        Lane Queue Depth
      </h3>
      <div className="h-72">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="timestamp"
              type="number"
              domain={["dataMin", "dataMax"]}
              tickFormatter={formatTimeTick}
              minTickGap={24}
            />
            <YAxis allowDecimals={false} />
            <Tooltip
              labelFormatter={(value) => new Date(Number(value)).toLocaleString()}
              formatter={(value: number, name: string) => [value.toFixed(0), name]}
            />
            <Legend
              onClick={(payload: { dataKey?: unknown }) => {
                if (typeof payload.dataKey === "string") {
                  onToggle(payload.dataKey);
                }
              }}
            />
            {lanes.map((lane, index) =>
              visible[lane] ? (
                <Area
                  key={lane}
                  type="monotone"
                  dataKey={lane}
                  stackId="lane-depth"
                  stroke={laneColor(index)}
                  fill={laneColor(index)}
                  fillOpacity={0.35}
                />
              ) : null
            )}
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </section>
  );
}
