type ErrorStateProps = {
  message: string;
  onRetry?: () => void;
};

export function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <section className="rounded-xl border border-red-300/60 bg-red-50/60 p-4 dark:border-red-500/40 dark:bg-red-950/20">
      <p className="text-sm font-medium text-red-700 dark:text-red-200">Request failed</p>
      <p className="mt-1 text-sm text-red-700/80 dark:text-red-200/80">{message}</p>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="mt-3 rounded-md border border-red-300 bg-white px-3 py-1 text-xs font-semibold text-red-700 hover:bg-red-100 dark:border-red-500/50 dark:bg-red-950/40 dark:text-red-200 dark:hover:bg-red-900/60"
        >
          Retry
        </button>
      ) : null}
    </section>
  );
}
