package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// ─── EVENTS ───────────────────────────────────────────────────────────────────

type EventType string

const (
	EventHeartbeat        EventType = "heartbeat"
	EventAnswer           EventType = "answer"
	EventTabSwitch        EventType = "tab_switch"
	EventFullscreenExit   EventType = "fullscreen_exit"
	EventCheatingDetected EventType = "cheating_detected"
	EventExamPaused       EventType = "exam_paused"
	EventExamFinished     EventType = "exam_finished"
	EventFaceAlert        EventType = "face_alert"
)

type Message struct {
	Event     EventType   `json:"event"`
	AttemptID string      `json:"attempt_id,omitempty"`
	PesertaID string      `json:"peserta_id,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
	Reason    string      `json:"reason,omitempty"`
}

// ─── CLIENT ───────────────────────────────────────────────────────────────────

type Client struct {
	ID        string
	Role      string
	AttemptID string
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *Hub
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("[WS] Write error for client %s: %v", c.ID, err)
			return
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(4096)
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		var event Message
		if err := json.Unmarshal(msg, &event); err != nil {
			continue
		}
		event.PesertaID = c.ID
		c.Hub.Broadcast <- &event
	}
}

// ─── HUB ──────────────────────────────────────────────────────────────────────

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client       // clientID → client
	attempts   map[string]*Client       // attemptID → client (peserta)
	admins     map[string]*Client       // adminID → client
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		attempts:   make(map[string]*Client),
		admins:     make(map[string]*Client),
		Register:   make(chan *Client, 64),
		Unregister: make(chan *Client, 64),
		Broadcast:  make(chan *Message, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if client.Role == "peserta" && client.AttemptID != "" {
				h.attempts[client.AttemptID] = client
			} else if client.Role == "admin" || client.Role == "guru" {
				h.admins[client.ID] = client
			}
			h.mu.Unlock()
			log.Printf("[WS] Client registered: %s (role:%s)", client.ID, client.Role)

		case client := <-h.Unregister:
			h.mu.Lock()
			delete(h.clients, client.ID)
			delete(h.attempts, client.AttemptID)
			delete(h.admins, client.ID)
			h.mu.Unlock()
			close(client.Send)
			log.Printf("[WS] Client disconnected: %s", client.ID)

		case msg := <-h.Broadcast:
			h.handleMessage(msg)
		}
	}
}

func (h *Hub) handleMessage(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// Forward all events to admins/guru for monitoring
	for _, admin := range h.admins {
		select {
		case admin.Send <- data:
		default:
			log.Printf("[WS] Admin %s send buffer full", admin.ID)
		}
	}
}

// SendToAttempt sends a message directly to a peserta by attemptID
func (h *Hub) SendToAttempt(attemptID string, msg *Message) {
	h.mu.RLock()
	client, ok := h.attempts[attemptID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case client.Send <- data:
	default:
	}
}

// BroadcastToAdmins sends a message to all admin/guru clients
func (h *Hub) BroadcastToAdmins(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, admin := range h.admins {
		select {
		case admin.Send <- data:
		default:
		}
	}
}

// GetOnlineAttempts returns list of currently connected attemptIDs
func (h *Hub) GetOnlineAttempts() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.attempts))
	for id := range h.attempts {
		ids = append(ids, id)
	}
	return ids
}
