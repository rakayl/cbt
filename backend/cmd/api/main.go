package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/cbt-ai/enterprise-cbt/internal/database"
	"github.com/cbt-ai/enterprise-cbt/internal/middleware"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/logger"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/observability"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/rabbitmq"
	cbtredis "github.com/cbt-ai/enterprise-cbt/internal/pkg/redis"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/storage"
	"github.com/cbt-ai/enterprise-cbt/internal/routes"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/getsentry/sentry-go"
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.Load()
	log, err := logger.New(cfg)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	if cfg.SentryDSN != "" {
		_ = sentry.Init(sentry.ClientOptions{Dsn: cfg.SentryDSN, Environment: cfg.AppEnv})
		defer sentry.Flush(2 * time.Second)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.NewPostgresPool(ctx, cfg)
	if err != nil {
		log.Fatal("postgres connection failed", zap.Error(err))
	}
	defer db.Close()
	rdb, err := cbtredis.New(ctx, cfg)
	if err != nil {
		log.Fatal("redis connection failed", zap.Error(err))
	}
	defer rdb.Close()
	rabbit, err := rabbitmq.New(cfg.RabbitMQURL)
	if err != nil {
		log.Fatal("rabbitmq connection failed", zap.Error(err))
	}
	defer rabbit.Close()
	minioClient, err := storage.NewMinIO(ctx, cfg)
	if err != nil {
		log.Fatal("object storage connection failed", zap.Error(err))
	}
	deps := shared.Deps{Config: cfg, DB: db, Redis: rdb, Rabbit: rabbit, Storage: minioClient, Logger: log}
	app := fiber.New(fiber.Config{AppName: cfg.AppName, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second, BodyLimit: 100 * 1024 * 1024, ErrorHandler: func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		var e *fiber.Error
		if errors.As(err, &e) {
			code = e.Code
		}
		return response.Error(c, code, err.Error(), nil)
	}})
	app.Use(fiberzap.New(fiberzap.Config{Logger: log}))
	app.Use(middleware.SecureHeaders(), middleware.CORS(cfg), middleware.IPWhitelist(cfg), middleware.CSRF(), middleware.RequestSigning(cfg), middleware.Tenant(), middleware.RateLimit(cfg, rdb), observability.Middleware(), middleware.Audit(log))
	app.Get("/healthz", func(c *fiber.Ctx) error {
		if err := db.Ping(c.Context()); err != nil {
			return response.Error(c, fiber.StatusServiceUnavailable, "database unavailable", nil)
		}
		if err := rdb.Ping(c.Context()).Err(); err != nil {
			return response.Error(c, fiber.StatusServiceUnavailable, "redis unavailable", nil)
		}
		return response.OK(c, fiber.Map{"status": "ok", "service": cfg.AppName, "time": time.Now().UTC()})
	})
	app.Get("/readyz", func(c *fiber.Ctx) error { return response.OK(c, fiber.Map{"status": "ready"}) })
	observability.Register(app)
	routes.Register(app, deps)
	go func() {
		if err := app.Listen(":" + cfg.AppPort); err != nil {
			log.Fatal("api stopped", zap.Error(err))
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer shutdownCancel()
	go func() {
		<-shutdownCtx.Done()
		if shutdownCtx.Err() != nil {
			log.Fatal("forced shutdown", zap.Error(shutdownCtx.Err()))
		}
	}()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Fatal("graceful shutdown failed", zap.Error(err))
	}
	fmt.Println("server stopped gracefully")
}
