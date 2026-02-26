import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { useEffect, useMemo, useState } from "react";

import { ApiError } from "../api/client";
import { useWorkflowStore } from "../stores/workflows";

type SubmitWorkflowDialogProps = {
  open: boolean;
  onClose: () => void;
};

const DEFAULT_JSON = JSON.stringify(
  {
    name: "example-workflow",
    description: "Sample workflow definition",
    tasks: [
      {
        id: "task-1",
        name: "Task 1",
        type: "function",
        depends_on: [],
        timeout: 30,
        retries: 0,
      },
    ],
    metadata: {
      source: "web-ui",
    },
  },
  null,
  2
);

export function SubmitWorkflowDialog({ open, onClose }: SubmitWorkflowDialogProps) {
  const submitWorkflowJSON = useWorkflowStore((state) => state.submitWorkflowJSON);
  const [value, setValue] = useState(DEFAULT_JSON);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setError(null);
    }
  }, [open]);

  const jsonError = useMemo(() => {
    try {
      JSON.parse(value);
      return null;
    } catch (err) {
      return (err as Error).message;
    }
  }, [value]);

  const onSubmit = async () => {
    if (jsonError) {
      setError(`Invalid JSON: ${jsonError}`);
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await submitWorkflowJSON(value);
      onClose();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError((err as Error).message);
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
      <div className="fixed inset-0 grid place-items-center p-4">
        <DialogPanel className="w-full max-w-3xl rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
          <DialogTitle className="text-lg font-semibold">Submit Workflow</DialogTitle>
          <p className="mt-1 text-sm text-[var(--ui-muted)]">
            Provide workflow JSON. Input is validated before request submission.
          </p>

          <textarea
            className="mt-4 h-96 w-full rounded-md border border-[var(--ui-border)] bg-transparent p-3 font-mono text-xs"
            value={value}
            onChange={(event) => setValue(event.target.value)}
            spellCheck={false}
          />

          {error ? (
            <p className="mt-2 text-sm text-red-600 dark:text-red-300">{error}</p>
          ) : jsonError ? (
            <p className="mt-2 text-sm text-amber-600 dark:text-amber-300">
              Invalid JSON: {jsonError}
            </p>
          ) : (
            <p className="mt-2 text-sm text-[var(--ui-muted)]">JSON syntax is valid.</p>
          )}

          <div className="mt-4 flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md border border-[var(--ui-border)] px-3 py-1.5 text-sm"
              disabled={submitting}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={onSubmit}
              className="rounded-md bg-[var(--ui-accent)] px-3 py-1.5 text-sm font-semibold text-[var(--ui-accent-fg)] disabled:opacity-60"
              disabled={submitting}
            >
              {submitting ? "Submitting..." : "Submit"}
            </button>
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  );
}
