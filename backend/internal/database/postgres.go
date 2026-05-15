package database

import (
	"context"
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func NewPostgresPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.PostgresMaxConns
	poolCfg.MinConns = cfg.PostgresMinConns
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.HealthCheckPeriod = 30 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}
	return pool, pool.Ping(ctx)
}
