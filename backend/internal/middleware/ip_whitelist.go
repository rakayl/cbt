package middleware

import (
	"net"

	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/gofiber/fiber/v2"
)

func IPWhitelist(cfg config.Config) fiber.Handler {
	allowed := make([]*net.IPNet, 0, len(cfg.IPWhitelist))
	for _, raw := range cfg.IPWhitelist {
		if _, block, err := net.ParseCIDR(raw); err == nil {
			allowed = append(allowed, block)
			continue
		}
		if ip := net.ParseIP(raw); ip != nil {
			mask := net.CIDRMask(32, 32)
			if ip.To4() == nil {
				mask = net.CIDRMask(128, 128)
			}
			allowed = append(allowed, &net.IPNet{IP: ip, Mask: mask})
		}
	}
	return func(c *fiber.Ctx) error {
		if len(allowed) == 0 {
			return c.Next()
		}
		ip := net.ParseIP(c.IP())
		for _, block := range allowed {
			if block.Contains(ip) {
				return c.Next()
			}
		}
		return response.Error(c, fiber.StatusForbidden, "ip address is not allowed", nil)
	}
}
