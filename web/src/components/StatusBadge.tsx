import type { WorkflowState } from "../types/api";

type StatusBadgeProps = {
  status: WorkflowState | string;
};

function colorClass(status: string) {
  switch (status) {
    case "running":
      return "bg-blue-100 text-blue-700 ring-blue-300 dark:bg-blue-900/40 dark:text-blue-200";
    case "completed":
      return "bg-emerald-100 text-emerald-700 ring-emerald-300 dark:bg-emerald-900/40 dark:text-emerald-200";
    case "failed":
      return "bg-red-100 text-red-700 ring-red-300 dark:bg-red-900/40 dark:text-red-200";
    case "cancelled":
      return "bg-amber-100 text-amber-700 ring-amber-300 dark:bg-amber-900/40 dark:text-amber-200";
    case "scheduled":
      return "bg-slate-100 text-slate-700 ring-slate-300 dark:bg-slate-700/50 dark:text-slate-200";
    case "pending":
    default:
      return "bg-zinc-100 text-zinc-700 ring-zinc-300 dark:bg-zinc-700/50 dark:text-zinc-200";
  }
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold capitalize ring-1 ring-inset ${colorClass(
        status
      )}`}
    >
      {status}
    </span>
  );
}
