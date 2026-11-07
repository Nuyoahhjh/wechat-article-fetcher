package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"wechat-article-fetcher/server/internal/protocol"
	"wechat-article-fetcher/server/internal/service"
)

const (
	maxMessageSize = 8 * 1024 * 1024
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 25 * time.Second
)

var ErrAgentOffline = errors.New("agent is offline")

type MessageHandler interface {
	Handle(ctx context.Context, agentID string, message protocol.Envelope) error
}

type Hub struct {
	mu          sync.RWMutex
	connections map[string]*agentConnection
	handler     MessageHandler
	agents      *service.AgentService
	logger      *slog.Logger
	upgrader    websocket.Upgrader
}

type agentConnection struct {
	agentID   string
	conn      *websocket.Conn
	send      chan protocol.Envelope
	done      chan struct{}
	closeOnce sync.Once
}

type Handler struct {
	hub     *Hub
	agents  *service.AgentService
	logger  *slog.Logger
	allowed map[string]bool
}

func NewHub(
	handler MessageHandler,
	agents *service.AgentService,
	allowedOrigins []string,
	logger *slog.Logger,
) *Hub {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = true
	}
	hub := &Hub{
		connections: make(map[string]*agentConnection),
		handler:     handler,
		agents:      agents,
		logger:      logger,
	}
	hub.upgrader = websocket.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			return origin == "" || allowed[origin]
		},
	}
	return hub
}

func NewHandler(hub *Hub, agents *service.AgentService, logger *slog.Logger) *Handler {
	return &Handler{hub: hub, agents: agents, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimSpace(chi.URLParam(r, "agentID"))
	if agentID == "" {
		http.Error(w, "agent id is required", http.StatusBadRequest)
		return
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}
	if _, err := h.agents.Authenticate(r.Context(), agentID, token); err != nil {
		http.Error(w, "invalid agent credentials", http.StatusUnauthorized)
		return
	}
	conn, err := h.hub.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("websocket upgrade failed", "agent_id", agentID, "error", err)
		return
	}
	h.hub.Serve(conn, agentID)
}

func (h *Hub) Serve(conn *websocket.Conn, agentID string) {
	client := &agentConnection{
		agentID: agentID,
		conn:    conn,
		send:    make(chan protocol.Envelope, 64),
		done:    make(chan struct{}),
	}
	h.register(client)
	go client.writeLoop()
	// Update agent status to online
	h.logger.Info("MarkOnline called", "agent_id", agentID)
	if err := h.agents.MarkOnline(context.Background(), agentID); err != nil {
		h.logger.Warn("failed to mark agent online", "agent_id", agentID, "error", err)
	}
	client.readLoop(h)
	h.unregister(client)
	client.close()
}

func (h *Hub) register(client *agentConnection) {
	h.mu.Lock()
	previous := h.connections[client.agentID]
	h.connections[client.agentID] = client
	h.mu.Unlock()
	if previous != nil {
		previous.close()
	}
	h.logger.Info("agent websocket opened", "agent_id", client.agentID)
}

func (h *Hub) unregister(client *agentConnection) {
	h.mu.Lock()
	current := h.connections[client.agentID]
	if current == client {
		delete(h.connections, client.agentID)
	}
	h.mu.Unlock()
	if current != client {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.agents.MarkOffline(ctx, client.agentID)
	h.logger.Info("agent websocket closed", "agent_id", client.agentID)
}

func (h *Hub) SendToAgent(agentID string, message protocol.Envelope) error {
	h.mu.RLock()
	client := h.connections[agentID]
	h.mu.RUnlock()
	if client == nil {
		return ErrAgentOffline
	}
	select {
	case client.send <- message:
		return nil
	case <-client.done:
		return ErrAgentOffline
	case <-time.After(3 * time.Second):
		return errors.New("agent send queue is full")
	}
}

func (h *Hub) IsOnline(agentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connections[agentID] != nil
}

func (h *Hub) DisconnectAgent(agentID string) {
	h.mu.RLock()
	client := h.connections[agentID]
	h.mu.RUnlock()
	if client != nil {
		client.close()
	}
}

func (c *agentConnection) readLoop(hub *Hub) {
	defer c.close()
	helloReceived := false
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				hub.logger.Debug("agent websocket read ended", "agent_id", c.agentID, "error", err)
			}
			return
		}
		var message protocol.Envelope
		if err := json.Unmarshal(payload, &message); err != nil {
			c.enqueueProtocolError("", "invalid_json", "message must be a JSON envelope", false)
			continue
		}
		if err := validateEnvelope(&message, c.agentID); err != nil {
			c.enqueueProtocolError(message.ID, "invalid_envelope", err.Error(), false)
			continue
		}
		if !helloReceived && message.Type != protocol.TypeAgentHello {
			c.enqueueProtocolError(message.ID, "connection_not_initialized", "agent.hello must be the first message", false)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = hub.handler.Handle(ctx, c.agentID, message)
		cancel()
		if err != nil {
			hub.logger.Warn(
				"agent message failed",
				"agent_id", c.agentID,
				"type", message.Type,
				"message_id", message.ID,
				"error", err,
			)
			c.enqueueProtocolError(message.ID, "message_failed", err.Error(), false)
			continue
		}
		if message.Type == protocol.TypeAgentHello {
			helloReceived = true
		}
		if message.Type != protocol.TypeAgentHeartbeat && message.Type != protocol.TypeAgentLog {
			c.enqueue(map[string]any{
				"reply_to": message.ID,
			}, protocol.TypeMessageAck)
		}
	}
}

func (c *agentConnection) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case message := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteJSON(message); err != nil {
				c.close()
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *agentConnection) enqueue(data any, messageType string) {
	message, err := newMessage(messageType, data)
	if err == nil {
		select {
		case c.send <- message:
		case <-c.done:
		}
	}
}

func (c *agentConnection) enqueueProtocolError(replyTo, code, message string, retryable bool) {
	envelope, err := protocol.New(protocol.TypeProtocolError, uuid.NewString(), nil)
	if err != nil {
		return
	}
	envelope.RequestID = replyTo
	envelope.Error = &protocol.Error{Code: code, Message: message, Retryable: retryable}
	select {
	case c.send <- envelope:
	case <-c.done:
	}
}

func (c *agentConnection) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func validateEnvelope(message *protocol.Envelope, agentID string) error {
	if message.Version == "" {
		message.Version = protocol.Version
	}
	if message.Version != protocol.Version {
		return fmt.Errorf("unsupported protocol version %q", message.Version)
	}
	if strings.TrimSpace(message.ID) == "" {
		return errors.New("message id is required")
	}
	if strings.TrimSpace(message.Type) == "" {
		return errors.New("message type is required")
	}
	if message.AgentID != "" && message.AgentID != agentID {
		return errors.New("agent_id does not match the authenticated connection")
	}
	message.AgentID = agentID
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now().UTC()
	}
	return nil
}

func newMessage(messageType string, data any) (protocol.Envelope, error) {
	return protocol.New(messageType, uuid.NewString(), data)
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
