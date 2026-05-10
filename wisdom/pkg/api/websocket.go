package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/wisdom/pkg/metabolism"
	"github.com/google/wisdom/pkg/observability"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Dynamic origin check can be added later
	},
}

// WSMessage represents the structure of messages sent over the WebSocket.
type WSMessage struct {
	Type    string         `json:"type"`    // CHAT_CHUNK, NOTIFICATION, ERROR, THOUGHT
	Payload map[string]any `json:"payload"`
}

// WSManager handles active WebSocket connections.
type WSManager struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

// NewWSManager initializes a new WebSocket manager.
func NewWSManager() *WSManager {
	return &WSManager{
		clients: make(map[*websocket.Conn]bool),
	}
}

// Broadcast sends a message to all connected clients.
func (m *WSManager) Broadcast(msg WSMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for client := range m.clients {
		err := client.WriteJSON(msg)
		if err != nil {
			observability.Logger.Error("WebSocket broadcast error", "error", err)
			client.Close()
			delete(m.clients, client)
		}
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleWS")
	defer span.End()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		observability.Logger.Error("WebSocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	s.wsManager.mu.Lock()
	s.wsManager.clients[conn] = true
	s.wsManager.mu.Unlock()

	defer func() {
		s.wsManager.mu.Lock()
		delete(s.wsManager.clients, conn)
		s.wsManager.mu.Unlock()
	}()

	observability.Logger.Info("New WebSocket connection established")

	var lastFrame []byte

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				observability.Logger.Error("WebSocket read error", "error", err)
			}
			break
		}

		if messageType == websocket.BinaryMessage {
			// Real-time visual frame received
			lastFrame = data
			continue
		}

		// Handle Text Messages (JSON)
		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Handle incoming messages
		switch msg.Type {
		case "CHAT_REQUEST":
			if lastFrame != nil {
				msg.Payload["visual_context"] = lastFrame
			}
			s.handleWSChat(ctx, conn, msg.Payload)
		default:
			observability.Logger.Warn("Unknown WebSocket message type", "type", msg.Type)
		}
	}
}

func (s *Server) handleWSChat(ctx context.Context, conn *websocket.Conn, payload map[string]any) {
	startTime := time.Now()
	message, ok := payload["message"].(string)
	if !ok {
		conn.WriteJSON(WSMessage{Type: "ERROR", Payload: map[string]any{"error": "missing message"}})
		return
	}

	var response string
	var contextStrings []string
	var err error

	if visual, ok := payload["visual_context"].([]byte); ok {
		// Multi-modal response
		observability.Logger.Info("Processing multi-modal chat request with visual context")
		response, err = s.chat.LLM.IngestDocument(ctx, visual, "image/jpeg") // Re-using IngestDocument for frames
	} else {
		response, contextStrings, err = s.chat.Ask(ctx, "anonymous", message)
	}

	if err != nil {
		conn.WriteJSON(WSMessage{Type: "ERROR", Payload: map[string]any{"error": err.Error()}})
		return
	}

	// Record metabolic usage
	tokensIn := len(message) / 4
	tokensOut := len(response) / 4
	s.tracker.Record("anonymous", metabolism.Usage{
		TokensIn:    tokensIn,
		TokensOut:   tokensOut,
		SignalUnits: len(contextStrings),
		Duration:    time.Since(startTime),
	})
	s.broadcastMetabolism("anonymous")

	conn.WriteJSON(WSMessage{
		Type: "CHAT_RESPONSE",
		Payload: map[string]any{
			"response": response,
			"context":  contextStrings,
		},
	})
}

// Notify broadcasts a system-wide notification via WebSocket.
func (s *Server) Notify(msgType string, payload map[string]any) {
	s.wsManager.Broadcast(WSMessage{
		Type:    msgType,
		Payload: payload,
	})
}

