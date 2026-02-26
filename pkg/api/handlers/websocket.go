package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/goclaw/goclaw/pkg/logger"
	"github.com/gorilla/websocket"
)

const (
	defaultWSMaxConnections = 100
	defaultPingInterval     = 30 * time.Second
	defaultPongTimeout      = 10 * time.Second
	defaultWriteTimeout     = 10 * time.Second
	defaultSendBuffer       = 32
)

// WebSocketConfig configures websocket handler behavior.
type WebSocketConfig struct {
	AllowedOrigins []string
	MaxConnections int
	PingInterval   time.Duration
	PongTimeout    time.Duration
}

// EventMessage is the websocket event format.
type EventMessage struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

type incomingMessage struct {
	Type       string         `json:"type"`
	WorkflowID string         `json:"workflow_id,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type wsClient struct {
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]struct{}
	mu            sync.RWMutex
	closeOnce     sync.Once
}

func newWSClient(conn *websocket.Conn) *wsClient {
	return &wsClient{
		conn:          conn,
		send:          make(chan []byte, defaultSendBuffer),
		subscriptions: make(map[string]struct{}),
	}
}

func (c *wsClient) close() {
	c.closeOnce.Do(func() {
		close(c.send)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

func (c *wsClient) subscribe(workflowID string) {
	if workflowID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscriptions[workflowID] = struct{}{}
}

func (c *wsClient) unsubscribe(workflowID string) {
	if workflowID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscriptions, workflowID)
}

func (c *wsClient) shouldReceive(workflowID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.subscriptions) == 0 {
		return true
	}
	if workflowID == "" {
		return false
	}
	_, ok := c.subscriptions[workflowID]
	return ok
}

// ConnectionManager manages active websocket clients.
type ConnectionManager struct {
	mu             sync.RWMutex
	clients        map[*wsClient]struct{}
	maxConnections int
}

// NewConnectionManager creates a manager with max connection limit.
func NewConnectionManager(maxConnections int) *ConnectionManager {
	if maxConnections <= 0 {
		maxConnections = defaultWSMaxConnections
	}
	return &ConnectionManager{
		clients:        make(map[*wsClient]struct{}),
		maxConnections: maxConnections,
	}
}

// Register registers a websocket client.
func (m *ConnectionManager) Register(client *wsClient) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.clients) >= m.maxConnections {
		return errors.New("websocket connection limit reached")
	}
	m.clients[client] = struct{}{}
	return nil
}

// Unregister unregisters a websocket client.
func (m *ConnectionManager) Unregister(client *wsClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.clients[client]; !ok {
		return
	}
	delete(m.clients, client)
	client.close()
}

// Count returns active connection count.
func (m *ConnectionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// CanAccept reports whether there is capacity for one more connection.
func (m *ConnectionManager) CanAccept() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients) < m.maxConnections
}

// Broadcast broadcasts event to matching clients.
func (m *ConnectionManager) Broadcast(event EventMessage) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	workflowID := workflowIDFromPayload(event.Payload)

	m.mu.RLock()
	clients := make([]*wsClient, 0, len(m.clients))
	for client := range m.clients {
		clients = append(clients, client)
	}
	m.mu.RUnlock()

	for _, client := range clients {
		if !client.shouldReceive(workflowID) {
			continue
		}
		select {
		case client.send <- payload:
		default:
			m.Unregister(client)
		}
	}

	return nil
}

// Close closes all active websocket connections.
func (m *ConnectionManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for client := range m.clients {
		client.close()
		delete(m.clients, client)
	}
}

// WebSocketHandler handles /ws/events.
type WebSocketHandler struct {
	log          logger.Logger
	manager      *ConnectionManager
	upgrader     websocket.Upgrader
	pingInterval time.Duration
	pongTimeout  time.Duration
	writeTimeout time.Duration
}

// NewWebSocketHandler creates a websocket handler.
func NewWebSocketHandler(log logger.Logger, cfg WebSocketConfig) *WebSocketHandler {
	if cfg.MaxConnections <= 0 {
		cfg.MaxConnections = defaultWSMaxConnections
	}
	if cfg.PingInterval <= 0 {
		cfg.PingInterval = defaultPingInterval
	}
	if cfg.PongTimeout <= 0 {
		cfg.PongTimeout = defaultPongTimeout
	}

	handler := &WebSocketHandler{
		log:          log,
		manager:      NewConnectionManager(cfg.MaxConnections),
		pingInterval: cfg.PingInterval,
		pongTimeout:  cfg.PongTimeout,
		writeTimeout: defaultWriteTimeout,
	}

	allowedOrigins := append([]string(nil), cfg.AllowedOrigins...)
	handler.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return isWebSocketOriginAllowed(r, allowedOrigins)
		},
	}

	return handler
}

// ServeHTTP upgrades HTTP to websocket and starts client loops.
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return
	}
	if !h.manager.CanAccept() {
		http.Error(w, "websocket connection limit reached", http.StatusServiceUnavailable)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if h.log != nil {
			h.log.Warn("websocket upgrade failed", "error", err)
		}
		return
	}

	client := newWSClient(conn)
	if err := h.manager.Register(client); err != nil {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "too many websocket connections"),
			time.Now().Add(h.writeTimeout),
		)
		_ = conn.Close()
		return
	}

	go h.writePump(client)
	h.readPump(client)
}

func (h *WebSocketHandler) readPump(client *wsClient) {
	defer h.manager.Unregister(client)

	readDeadline := h.pingInterval + h.pongTimeout
	client.conn.SetReadLimit(1 << 20)
	_ = client.conn.SetReadDeadline(time.Now().Add(readDeadline))
	client.conn.SetPongHandler(func(_ string) error {
		return client.conn.SetReadDeadline(time.Now().Add(readDeadline))
	})

	for {
		_, data, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) && h.log != nil {
				h.log.Warn("websocket read error", "error", err)
			}
			return
		}
		h.handleIncomingMessage(client, data)
	}
}

func (h *WebSocketHandler) writePump(client *wsClient) {
	ticker := time.NewTicker(h.pingInterval)
	defer func() {
		ticker.Stop()
		h.manager.Unregister(client)
	}()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				_ = client.conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
					time.Now().Add(h.writeTimeout),
				)
				return
			}
			_ = client.conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))
			if err := client.conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(h.writeTimeout)); err != nil {
				return
			}
		}
	}
}

func (h *WebSocketHandler) handleIncomingMessage(client *wsClient, raw []byte) {
	var message incomingMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return
	}

	workflowID := strings.TrimSpace(message.WorkflowID)
	if workflowID == "" && message.Payload != nil {
		if value, ok := message.Payload["workflow_id"].(string); ok {
			workflowID = strings.TrimSpace(value)
		}
	}

	switch strings.ToLower(strings.TrimSpace(message.Type)) {
	case "subscribe":
		client.subscribe(workflowID)
	case "unsubscribe":
		client.unsubscribe(workflowID)
	}
}

// Broadcast sends an event to matching websocket clients.
func (h *WebSocketHandler) Broadcast(event EventMessage) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	return h.manager.Broadcast(event)
}

// Close closes all websocket clients.
func (h *WebSocketHandler) Close() {
	h.manager.Close()
}

func workflowIDFromPayload(payload any) string {
	if payload == nil {
		return ""
	}
	switch value := payload.(type) {
	case map[string]any:
		if workflowID, ok := value["workflow_id"].(string); ok {
			return workflowID
		}
	case map[string]string:
		return value["workflow_id"]
	}
	return ""
}

func isWebSocketOriginAllowed(r *http.Request, allowedOrigins []string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" || strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(originURL.Host, r.Host)
}
