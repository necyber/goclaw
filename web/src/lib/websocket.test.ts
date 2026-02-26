import { RealtimeWebSocketClient } from "./websocket";

class MockWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;
  static instances: MockWebSocket[] = [];

  readonly url: string;
  readyState = MockWebSocket.CONNECTING;
  sent: string[] = [];
  private listeners: Record<string, Array<(event: any) => void>> = {};

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  addEventListener(type: string, listener: (event: any) => void) {
    this.listeners[type] = [...(this.listeners[type] ?? []), listener];
  }

  removeEventListener(type: string, listener: (event: any) => void) {
    this.listeners[type] = (this.listeners[type] ?? []).filter((item) => item !== listener);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
    this.emit("close", {});
  }

  open() {
    this.readyState = MockWebSocket.OPEN;
    this.emit("open", {});
  }

  emitMessage(data: string) {
    this.emit("message", { data });
  }

  emitClose() {
    this.readyState = MockWebSocket.CLOSED;
    this.emit("close", {});
  }

  emit(type: string, event: any) {
    for (const listener of this.listeners[type] ?? []) {
      listener(event);
    }
  }
}

describe("RealtimeWebSocketClient", () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.useFakeTimers();
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("connects and sends heartbeat pings", () => {
    const stateChanges: string[] = [];
    const client = new RealtimeWebSocketClient(
      (state) => stateChanges.push(state),
      vi.fn()
    );

    client.connect();
    expect(MockWebSocket.instances.length).toBe(1);
    const socket = MockWebSocket.instances[0];
    expect(socket.url.endsWith("/ws/events")).toBe(true);

    socket.open();
    expect(stateChanges).toContain("connected");

    vi.advanceTimersByTime(30_000);
    expect(socket.sent.length).toBe(1);
    expect(JSON.parse(socket.sent[0]).type).toBe("ping");
  });

  it("dispatches parsed messages to the listener", () => {
    const received: Array<{ type: string }> = [];
    const client = new RealtimeWebSocketClient(vi.fn(), (message) => {
      received.push({ type: message.type });
    });

    client.connect();
    const socket = MockWebSocket.instances[0];
    socket.open();

    socket.emitMessage(JSON.stringify({ type: "workflow.state_changed", timestamp: "now", payload: {} }));
    socket.emitMessage("not-json");

    expect(received).toEqual([{ type: "workflow.state_changed" }]);
  });

  it("reconnects with exponential backoff after close", () => {
    const states: string[] = [];
    const client = new RealtimeWebSocketClient((state) => states.push(state), vi.fn());

    client.connect();
    let socket = MockWebSocket.instances[0];
    socket.open();
    socket.emitClose();

    expect(states[states.length - 1]).toBe("reconnecting");
    vi.advanceTimersByTime(1000);
    expect(MockWebSocket.instances.length).toBe(2);

    socket = MockWebSocket.instances[1];
    socket.open();
    expect(states[states.length - 1]).toBe("connected");
  });

  it("stops reconnecting after max attempts", () => {
    const states: string[] = [];
    const client = new RealtimeWebSocketClient((state) => states.push(state), vi.fn());

    client.connect();
    let socket = MockWebSocket.instances[0];
    socket.open();

    for (let attempt = 0; attempt < 10; attempt += 1) {
      socket.emitClose();
      const delay = Math.min(2 ** attempt * 1000, 30_000);
      vi.advanceTimersByTime(delay);
      socket = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    }

    socket.emitClose();
    expect(states[states.length - 1]).toBe("disconnected");
  });
});
