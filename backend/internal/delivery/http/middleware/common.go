package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── CORS ─────────────────────────────────────────────────────────────────────

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ─── LOGGER ───────────────────────────────────────────────────────────────────

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return "[GIN] " + param.TimeStamp.Format("2006/01/02 15:04:05") +
			" | " + param.Method +
			" | " + param.Path +
			" | " + param.ClientIP +
			" | " + param.Latency.String() +
			" | " + http.StatusText(param.StatusCode) + "\n"
	})
}

// ─── RATE LIMIT ───────────────────────────────────────────────────────────────

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientState
	max     int
	window  time.Duration
}

type clientState struct {
	count    int
	resetAt  time.Time
}

func RateLimit(max int, window time.Duration) gin.HandlerFunc {
	rl := &rateLimiter{
		clients: make(map[string]*clientState),
		max:     max,
		window:  window,
	}

	// Periodic cleanup
	go func() {
		for range time.Tick(window) {
			rl.mu.Lock()
			now := time.Now()
			for k, v := range rl.clients {
				if now.After(v.resetAt) {
					delete(rl.clients, k)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		rl.mu.Lock()
		state, ok := rl.clients[ip]
		if !ok || time.Now().After(state.resetAt) {
			rl.clients[ip] = &clientState{count: 1, resetAt: time.Now().Add(rl.window)}
			rl.mu.Unlock()
			c.Next()
			return
		}
		state.count++
		if state.count > rl.max {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "rate limit exceeded",
			})
			return
		}
		rl.mu.Unlock()
		c.Next()
	}
}
