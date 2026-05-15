package exam_sessions

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type SocketMessage struct {
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
	SentAt  time.Time `json:"sent_at"`
}

func WebSocketHandler(deps shared.Deps) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		defer c.Close()
		sessionID, _ := uuid.Parse(c.Query("session_id"))
		if sessionID == uuid.Nil {
			_ = c.WriteJSON(SocketMessage{Type: "error", Payload: "session_id is required", SentAt: time.Now().UTC()})
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer func() {
			cancel()
			disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer disconnectCancel()
			MarkSessionDisconnected(disconnectCtx, deps, sessionID)
		}()
		channel := "exam_session:" + sessionID.String() + ":events"
		var writeMu sync.Mutex
		write := func(message SocketMessage) bool {
			writeMu.Lock()
			defer writeMu.Unlock()
			_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
			return c.WriteJSON(message) == nil
		}

		if deps.Redis != nil {
			pubsub := deps.Redis.Subscribe(ctx, channel)
			defer pubsub.Close()
			go func() {
				for event := range pubsub.Channel() {
					var message SocketMessage
					if err := json.Unmarshal([]byte(event.Payload), &message); err != nil {
						message = SocketMessage{Type: "broadcast", Payload: event.Payload, SentAt: time.Now().UTC()}
					}
					if !write(message) {
						cancel()
						return
					}
				}
			}()
		}

		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			pingTicker := time.NewTicker(25 * time.Second)
			defer pingTicker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					state := timerState(ctx, deps, sessionID)
					if !write(SocketMessage{Type: "timer.tick", Payload: state, SentAt: time.Now().UTC()}) {
						cancel()
						return
					}
				case <-pingTicker.C:
					writeMu.Lock()
					_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
					err := c.WriteMessage(websocket.PingMessage, []byte("ping"))
					writeMu.Unlock()
					if err != nil {
						cancel()
						return
					}
				}
			}
		}()

		_ = c.SetReadDeadline(time.Now().Add(75 * time.Second))
		c.SetPongHandler(func(string) error {
			return c.SetReadDeadline(time.Now().Add(75 * time.Second))
		})
		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				return
			}
			_ = c.SetReadDeadline(time.Now().Add(75 * time.Second))
			var incoming SocketMessage
			if err := json.Unmarshal(raw, &incoming); err != nil {
				_ = write(SocketMessage{Type: "error", Payload: "invalid websocket message", SentAt: time.Now().UTC()})
				continue
			}
			incoming.SentAt = time.Now().UTC()
			body, _ := json.Marshal(incoming)
			if deps.Redis != nil {
				_ = deps.Redis.Publish(ctx, channel, body).Err()
			}
			_ = write(SocketMessage{Type: "exam_sessions.ack", Payload: incoming.Payload, SentAt: time.Now().UTC()})
		}
	}
}

func timerState(ctx context.Context, deps shared.Deps, sessionID uuid.UUID) map[string]any {
	state := map[string]any{"session_id": sessionID.String(), "server_time": time.Now().UTC()}
	if deps.Redis != nil {
		values, err := deps.Redis.HGetAll(ctx, "exam_session:"+sessionID.String()).Result()
		if err == nil {
			for key, value := range values {
				state[key] = value
			}
			if values["status"] == "reconnecting" && values["timer_paused"] == "true" {
				if rawRemaining, ok := values["remaining_seconds"]; ok {
					if parsed, parseErr := strconv.ParseInt(rawRemaining, 10, 64); parseErr == nil {
						state["remaining_seconds"] = parsed
					}
				}
				state["timer_paused"] = true
				return state
			}
			if rawEndsAt, ok := values["ends_at"]; ok {
				if endsAt, parseErr := time.Parse(time.RFC3339Nano, rawEndsAt); parseErr == nil {
					remaining := int64(time.Until(endsAt).Seconds())
					if remaining < 0 {
						remaining = 0
					}
					state["remaining_seconds"] = remaining
					state["ends_at"] = endsAt
				}
			} else if rawRemaining, ok := values["remaining_seconds"]; ok {
				if parsed, parseErr := strconv.ParseInt(rawRemaining, 10, 64); parseErr == nil {
					state["remaining_seconds"] = parsed
				}
			}
		}
	}
	if _, ok := state["remaining_seconds"]; !ok && deps.DB != nil {
		var endsAt time.Time
		var status string
		var rawMetadata []byte
		if err := deps.DB.QueryRow(ctx, `SELECT ends_at, status_enum::text, COALESCE(metadata,'{}'::jsonb) FROM exam_sessions WHERE id=$1 AND deleted_at IS NULL`, sessionID).Scan(&endsAt, &status, &rawMetadata); err == nil {
			metadata := map[string]any{}
			_ = json.Unmarshal(rawMetadata, &metadata)
			recovery := mapFromAny(metadata["recovery"])
			if status == "reconnecting" && boolFromAny(recovery["timer_paused"]) {
				state["remaining_seconds"] = int64(numberFromAny(recovery["remaining_seconds"]))
				state["status"] = status
				state["timer_paused"] = true
				state["recovery_status"] = stringFromAny(recovery["status"], "paused")
				return state
			}
			remaining := int64(time.Until(endsAt).Seconds())
			if remaining < 0 {
				remaining = 0
			}
			state["remaining_seconds"] = remaining
			state["status"] = status
			state["ends_at"] = endsAt
		}
	}
	return state
}
