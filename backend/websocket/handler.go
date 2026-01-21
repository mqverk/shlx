package websocket

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/mqverk/shlx/backend/session"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Configure CORS properly for production
	},
}

// Handler manages WebSocket connections
type Handler struct {
	manager *session.Manager
}

// NewHandler creates a new WebSocket handler
func NewHandler(manager *session.Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// HandleConnection handles a WebSocket connection
func (h *Handler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Read initial join message
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Printf("Failed to read join message: %v", err)
		return
	}

	var joinMsg JoinMessage
	if err := json.Unmarshal(msg, &joinMsg); err != nil {
		log.Printf("Invalid join message: %v", err)
		return
	}

	sess, ok := h.manager.GetSession(joinMsg.SessionID)
	if !ok {
		h.sendError(conn, "Session not found")
		return
	}

	// Determine role
	role := session.RoleReadOnly
	if joinMsg.Token == sess.Owner {
		role = session.RoleOwner
	} else if joinMsg.Role == "interactive" {
		role = session.RoleInteractive
	}

	user, userID := sess.AddUser(role)
	defer sess.RemoveUser(userID)

	// Send welcome message with user info
	h.sendJSON(conn, Message{
		Type: "welcome",
		Data: map[string]interface{}{
			"userId":    userID,
			"role":      role,
			"sessionId": sess.ID,
			"users":     sess.GetUsers(),
		},
	})

	// Broadcast user joined
	sess.Broadcast(h.marshalMessage(Message{
		Type: "user_joined",
		Data: map[string]interface{}{
			"userId": userID,
			"role":   role,
		},
	}))

	// Start reading from PTY and writing to WebSocket
	done := make(chan struct{})
	go h.readFromPTY(sess, user, conn, done)

	// Read from WebSocket and write to PTY
	h.readFromWebSocket(sess, userID, conn, done)

	// Broadcast user left
	sess.Broadcast(h.marshalMessage(Message{
		Type: "user_left",
		Data: map[string]interface{}{
			"userId": userID,
		},
	}))
}

func (h *Handler) readFromPTY(sess *session.Session, user *session.User, conn *websocket.Conn, done chan struct{}) {
	defer close(done)

	for {
		select {
		case data, ok := <-user.WriteChan:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func (h *Handler) readFromWebSocket(sess *session.Session, userID string, conn *websocket.Conn, done chan struct{}) {
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if err != io.EOF && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			// Terminal input
			if err := sess.Write(userID, data); err != nil {
				log.Printf("Failed to write to PTY: %v", err)
			}

		case websocket.TextMessage:
			// Control messages
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			h.handleControlMessage(sess, userID, msg)
		}
	}
}

func (h *Handler) handleControlMessage(sess *session.Session, userID string, msg Message) {
	switch msg.Type {
	case "resize":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			rows := uint16(data["rows"].(float64))
			cols := uint16(data["cols"].(float64))
			sess.PTY.Resize(rows, cols)
		}
	case "ping":
		// Health check, ignore
	}
}

func (h *Handler) sendError(conn *websocket.Conn, message string) {
	h.sendJSON(conn, Message{
		Type: "error",
		Data: map[string]interface{}{"message": message},
	})
}

func (h *Handler) sendJSON(conn *websocket.Conn, msg Message) {
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

func (h *Handler) marshalMessage(msg Message) []byte {
	data, _ := json.Marshal(msg)
	return data
}

// Message types
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type JoinMessage struct {
	SessionID string `json:"sessionId"`
	Token     string `json:"token"`
	Role      string `json:"role"`
}
