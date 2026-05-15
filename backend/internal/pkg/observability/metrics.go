package observability

import (
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry     = prometheus.NewRegistry()
	registerOnce sync.Once
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "cbt_http_request_duration_seconds", Help: "HTTP request latency", Buckets: prometheus.DefBuckets}, []string{"method", "path", "status"})
)

func Register(app *fiber.App) {
	registerOnce.Do(func() {
		registry.MustRegister(
			prometheus.NewGoCollector(),
			prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
			httpDuration,
		)
	})
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))
}
func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		routePath := c.Path()
		if c.Route() != nil {
			routePath = c.Route().Path
		}
		httpDuration.WithLabelValues(c.Method(), routePath, strconv.Itoa(c.Response().StatusCode())).Observe(time.Since(start).Seconds())
		return err
	}
}
