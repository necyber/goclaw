import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";

export type ThroughputPoint = {
  timestamp: number;
  submitted: number;
  completed: number;
};

export type ThroughputVisibility = { submitted: boolean; completed: boolean };

type ThroughputChartProps = {
  data: ThroughputPoint[];
  visible: ThroughputVisibility;
  onToggle: (key: "submitted" | "completed") => void;
};

function formatTimeTick(value: number) {
  return new Date(value).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit"
  });
}

export function ThroughputChart({ data, visible, onToggle }: ThroughputChartProps) {
  return (
    <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
      <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
        Throughput (1m granularity)
      </h3>
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="timestamp"
              type="number"
              domain={["dataMin", "dataMax"]}
              minTickGap={24}
              tickFormatter={formatTimeTick}
            />
            <YAxis />
            <Tooltip
              labelFormatter={(value) => new Date(Number(value)).toLocaleString()}
              formatter={(value: number, name: string) => [value.toFixed(2), name]}
            />
            <Legend
              onClick={(payload: any) => {
                if (payload.dataKey === "submitted" || payload.dataKey === "completed") {
                  onToggle(payload.dataKey);
                }
              }}
            />
            {visible.submitted ? (
              <Line type="monotone" dataKey="submitted" stroke="#1d4ed8" strokeWidth={2} dot={false} />
            ) : null}
            {visible.completed ? (
              <Line type="monotone" dataKey="completed" stroke="#059669" strokeWidth={2} dot={false} />
            ) : null}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </section>
  );
}
