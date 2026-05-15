package redis

import (
	"context"
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	goredis "github.com/redis/go-redis/v9"
	"time"
)

type Client interface {
	Ping(context.Context) *goredis.StatusCmd
	Close() error
	Set(context.Context, string, any, time.Duration) *goredis.StatusCmd
	Get(context.Context, string) *goredis.StringCmd
	Del(context.Context, ...string) *goredis.IntCmd
	HSet(context.Context, string, ...any) *goredis.IntCmd
	HGetAll(context.Context, string) *goredis.MapStringStringCmd
	Expire(context.Context, string, time.Duration) *goredis.BoolCmd
	Incr(context.Context, string) *goredis.IntCmd
	Publish(context.Context, string, any) *goredis.IntCmd
	Subscribe(context.Context, ...string) *goredis.PubSub
}

func New(ctx context.Context, cfg config.Config) (Client, error) {
	if len(cfg.RedisClusterAddrs) > 1 {
		c := goredis.NewClusterClient(&goredis.ClusterOptions{Addrs: cfg.RedisClusterAddrs, Password: cfg.RedisPassword})
		return c, c.Ping(ctx).Err()
	}
	c := goredis.NewClient(&goredis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	return c, c.Ping(ctx).Err()
}
