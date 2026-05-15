package middleware

import (
	"fmt"
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	cbtredis "github.com/cbt-ai/enterprise-cbt/internal/pkg/redis"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/gofiber/fiber/v2"
	"strconv"
	"time"
)

func RateLimit(cfg config.Config, client cbtredis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if client == nil {
			return c.Next()
		}
		key := fmt.Sprintf("rl:%s:%s", c.IP(), c.Path())
		ctx := c.Context()
		n, err := client.Incr(ctx, key).Result()
		if err == nil && n == 1 {
			_ = client.Expire(ctx, key, cfg.RateLimitWindow).Err()
		}
		if err == nil && int(n) > cfg.RateLimitMax {
			return response.Error(c, fiber.StatusTooManyRequests, "rate limit exceeded", nil)
		}
		c.Set("X-RateLimit-Window", strconv.Itoa(int(cfg.RateLimitWindow/time.Second)))
		return c.Next()
	}
}
