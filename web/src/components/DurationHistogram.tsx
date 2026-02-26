import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

export type DurationBucketPoint = {
  bucket: string;
  count: number;
};

type DurationHistogramProps = {
  data: DurationBucketPoint[];
};

export function DurationHistogram({ data }: DurationHistogramProps) {
  return (
    <section className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
      <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
        Task Duration Distribution
      </h3>
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="bucket" interval={0} angle={-15} textAnchor="end" height={60} />
            <YAxis allowDecimals={false} />
            <Tooltip formatter={(value: number) => [Math.round(value), "tasks"]} />
            <Bar dataKey="count" fill="#0f766e" />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </section>
  );
}
