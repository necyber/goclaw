import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ReferenceArea,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export type ErrorRatePoint = {
  timestamp: number;
  workflowErrorRate: number;
  taskErrorRate: number;
};

export type ErrorRateVisibility = {
  workflowErrorRate: boolean;
  taskErrorRate: boolean;
};

type ErrorRateChartProps = {
  data: ErrorRatePoint[];
  spikes: number[];
  visible: ErrorRateVisibility;
  onToggle: (key: "workflowErrorRate" | "taskErrorRate") => void;
};

function formatTimeTick(value: number) {
  return new Date(value).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function ErrorRateChart({ data, spikes, visible, onToggle }: ErrorRateChartProps) {
  return (
    <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
      <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
        Error Rate
      </h3>
      <div className="h-72">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="timestamp"
              type="number"
              domain={["dataMin", "dataMax"]}
              tickFormatter={formatTimeTick}
              minTickGap={24}
            />
            <YAxis unit="%" domain={[0, 100]} />
            <Tooltip
              labelFormatter={(value) => new Date(Number(value)).toLocaleString()}
              formatter={(value: number, name: string) => [`${value.toFixed(2)}%`, name]}
            />
            {spikes.map((ts) => (
              <ReferenceArea
                key={`spike-${ts}`}
                x1={ts - 30_000}
                x2={ts + 30_000}
                y1={0}
                y2={100}
                fill="rgba(220,38,38,0.12)"
              />
            ))}
            <ReferenceLine y={10} stroke="#dc2626" strokeDasharray="4 4" />
            <Legend
              onClick={(payload: { dataKey?: unknown }) => {
                if (
                  payload.dataKey === "workflowErrorRate" ||
                  payload.dataKey === "taskErrorRate"
                ) {
                  onToggle(payload.dataKey);
                }
              }}
            />
            {visible.workflowErrorRate ? (
              <Line
                type="monotone"
                dataKey="workflowErrorRate"
                name="Workflow errors"
                stroke="#dc2626"
                strokeWidth={2}
                dot={false}
              />
            ) : null}
            {visible.taskErrorRate ? (
              <Line
                type="monotone"
                dataKey="taskErrorRate"
                name="Task errors"
                stroke="#2563eb"
                strokeWidth={2}
                dot={false}
              />
            ) : null}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </section>
  );
}
