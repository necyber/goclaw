import { useEffect } from "react";

import { useWebSocketStore } from "../stores/websocket";

function statusColor(status: "connected" | "disconnected" | "reconnecting") {
  switch (status) {
    case "connected":
      return "bg-emerald-500";
    case "reconnecting":
      return "bg-amber-400";
    case "disconnected":
    default:
      return "bg-red-500";
  }
}

export function ConnectionStatus() {
  const status = useWebSocketStore((state) => state.status);
  const connectionLost = useWebSocketStore((state) => state.connectionLost);
  const connect = useWebSocketStore((state) => state.connect);
  const disconnect = useWebSocketStore((state) => state.disconnect);
  const reconnect = useWebSocketStore((state) => state.reconnect);

  useEffect(() => {
    connect();
    return () => disconnect();
  }, [connect, disconnect]);

  return (
    <div className="flex items-center gap-2 rounded-md border border-[var(--ui-border)] px-2 py-1 text-xs">
      <span className={`h-2.5 w-2.5 rounded-full ${statusColor(status)}`} />
      <span className="capitalize text-[var(--ui-muted)]">{status}</span>
      {connectionLost ? (
        <button
          type="button"
          onClick={reconnect}
          className="rounded border border-[var(--ui-border)] px-2 py-0.5 text-[10px] font-semibold"
        >
          Reconnect
        </button>
      ) : null}
    </div>
  );
}
