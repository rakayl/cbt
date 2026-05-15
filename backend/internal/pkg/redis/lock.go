package redis

import (
	"context"
	goredis "github.com/redis/go-redis/v9"
	"time"
)

func AcquireLock(ctx context.Context, client *goredis.Client, key string, value string, ttl time.Duration) (bool, error) {
	return client.SetNX(ctx, "lock:"+key, value, ttl).Result()
}
