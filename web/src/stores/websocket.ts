import { create } from "zustand";

import { RealtimeWebSocketClient, type WebSocketConnectionState, type WebSocketEventMessage } from "../lib/websocket";

type EventHandler = (event: WebSocketEventMessage) => void;

type WebSocketState = {
  status: WebSocketConnectionState;
  latestEvent: WebSocketEventMessage | null;
  connectionLost: boolean;
  client: RealtimeWebSocketClient | null;
  handlers: Record<string, EventHandler[]>;
  setStatus: (status: WebSocketConnectionState) => void;
  connect: () => void;
  disconnect: () => void;
  reconnect: () => void;
  subscribeWorkflow: (workflowID: string) => void;
  unsubscribeWorkflow: (workflowID: string) => void;
  onEvent: (eventType: string, handler: EventHandler) => () => void;
};

function dispatchEvent(handlers: Record<string, EventHandler[]>, event: WebSocketEventMessage) {
  const byType = handlers[event.type] ?? [];
  const wildcard = handlers["*"] ?? [];
  for (const handler of [...byType, ...wildcard]) {
    handler(event);
  }
}

export const useWebSocketStore = create<WebSocketState>((set, get) => ({
  status: "disconnected",
  latestEvent: null,
  connectionLost: false,
  client: null,
  handlers: {},
  setStatus: (status) =>
    set({
      status,
      connectionLost: status === "disconnected" ? get().connectionLost : false
    }),
  connect: () => {
    let client = get().client;
    if (!client) {
      client = new RealtimeWebSocketClient(
        (status) => {
          set((state) => ({
            status,
            connectionLost: status === "disconnected" && state.status === "reconnecting"
          }));
        },
        (event) => {
          set({ latestEvent: event });
          dispatchEvent(get().handlers, event);
        }
      );
      set({ client });
    }
    client.connect();
  },
  disconnect: () => {
    const client = get().client;
    if (client) {
      client.disconnect();
    }
    set({ status: "disconnected", connectionLost: false });
  },
  reconnect: () => {
    const client = get().client;
    if (client) {
      client.manualReconnect();
      return;
    }
    get().connect();
  },
  subscribeWorkflow: (workflowID) => {
    const client = get().client;
    if (!client || !workflowID) {
      return;
    }
    client.subscribeWorkflow(workflowID);
  },
  unsubscribeWorkflow: (workflowID) => {
    const client = get().client;
    if (!client || !workflowID) {
      return;
    }
    client.unsubscribeWorkflow(workflowID);
  },
  onEvent: (eventType, handler) => {
    set((state) => ({
      handlers: {
        ...state.handlers,
        [eventType]: [...(state.handlers[eventType] ?? []), handler]
      }
    }));
    return () => {
      set((state) => ({
        handlers: {
          ...state.handlers,
          [eventType]: (state.handlers[eventType] ?? []).filter((item) => item !== handler)
        }
      }));
    };
  }
}));

