export type WebSocketConnectionState = "connected" | "disconnected" | "reconnecting";

export interface WebSocketEventMessage<TPayload = unknown> {
  type: string;
  timestamp: string;
  payload: TPayload;
}

type ConnectionStateListener = (state: WebSocketConnectionState) => void;
type MessageListener = (message: WebSocketEventMessage) => void;

const MAX_RECONNECT_ATTEMPTS = 10;
const MAX_BACKOFF_MS = 30_000;

export class RealtimeWebSocketClient {
  private socket: WebSocket | null = null;
  private reconnectAttempts = 0;
  private reconnectTimer: number | null = null;
  private heartbeatTimer: number | null = null;
  private state: WebSocketConnectionState = "disconnected";

  private readonly onStateChange: ConnectionStateListener;
  private readonly onMessage: MessageListener;

  constructor(onStateChange: ConnectionStateListener, onMessage: MessageListener) {
    this.onStateChange = onStateChange;
    this.onMessage = onMessage;
  }

  connect() {
    if (
      this.socket &&
      (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    this.clearReconnectTimer();
    this.createSocket();
  }

  disconnect() {
    this.clearReconnectTimer();
    this.clearHeartbeatTimer();
    this.reconnectAttempts = 0;
    this.setState("disconnected");
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }

  manualReconnect() {
    this.reconnectAttempts = 0;
    this.connect();
  }

  send(message: unknown) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return;
    }
    this.socket.send(JSON.stringify(message));
  }

  subscribeWorkflow(workflowID: string) {
    this.send({ type: "subscribe", workflow_id: workflowID });
  }

  unsubscribeWorkflow(workflowID: string) {
    this.send({ type: "unsubscribe", workflow_id: workflowID });
  }

  private createSocket() {
    const scheme = window.location.protocol === "https:" ? "wss" : "ws";
    const url = `${scheme}://${window.location.host}/ws/events`;
    const socket = new WebSocket(url);
    this.socket = socket;

    socket.addEventListener("open", () => {
      this.reconnectAttempts = 0;
      this.setState("connected");
      this.startHeartbeat();
    });

    socket.addEventListener("message", (event) => {
      try {
        const parsed = JSON.parse(event.data as string) as WebSocketEventMessage;
        this.onMessage(parsed);
      } catch {
        // Ignore malformed events.
      }
    });

    socket.addEventListener("close", () => {
      this.clearHeartbeatTimer();
      if (this.state === "disconnected") {
        return;
      }
      this.scheduleReconnect();
    });

    socket.addEventListener("error", () => {
      socket.close();
    });
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      this.setState("disconnected");
      return;
    }

    this.reconnectAttempts += 1;
    this.setState("reconnecting");
    const delay = Math.min(2 ** (this.reconnectAttempts - 1) * 1000, MAX_BACKOFF_MS);

    this.clearReconnectTimer();
    this.reconnectTimer = window.setTimeout(() => {
      this.createSocket();
    }, delay);
  }

  private startHeartbeat() {
    this.clearHeartbeatTimer();
    this.heartbeatTimer = window.setInterval(() => {
      this.send({ type: "ping", timestamp: new Date().toISOString() });
    }, 30_000);
  }

  private setState(nextState: WebSocketConnectionState) {
    this.state = nextState;
    this.onStateChange(nextState);
  }

  private clearReconnectTimer() {
    if (this.reconnectTimer !== null) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private clearHeartbeatTimer() {
    if (this.heartbeatTimer !== null) {
      window.clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }
}
