package screen_recordings

import (
	"encoding/json"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/contrib/websocket"
	"time"
)

type SocketMessage struct {
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
	SentAt  time.Time `json:"sent_at"`
}

func WebSocketHandler(deps shared.Deps) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		defer c.Close()
		_ = c.SetReadDeadline(time.Now().Add(75 * time.Second))
		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			var incoming SocketMessage
			_ = json.Unmarshal(msg, &incoming)
			outgoing := SocketMessage{Type: "screen_recordings.ack", Payload: incoming.Payload, SentAt: time.Now().UTC()}
			body, _ := json.Marshal(outgoing)
			_ = c.WriteMessage(mt, body)
		}
	}
}
