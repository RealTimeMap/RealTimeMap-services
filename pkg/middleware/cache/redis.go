package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisCache struct {
	r      *redis.Client
	logger *zap.Logger
}

func NewRedisCache(client *redis.Client, logger *zap.Logger) Cache {
	return &RedisCache{
		r:      client,
		logger: logger,
	}
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool) {
	c.logger.Info("start RedisCache.Get", zap.String("key", key))
	val, err := c.r.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		c.logger.Debug("redis miss", zap.String("key", key))
		return nil, false
	} else if err != nil {
		c.logger.Warn("Failed to get value from cache", zap.String("key", key), zap.Error(err))
		return nil, false
	}
	return []byte(val), true
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.logger.Info("start RedisCache.Set", zap.String("key", key))
	return c.r.Set(ctx, key, value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	c.logger.Info("start RedisCache.Delete", zap.String("key", key))
	return c.r.Del(ctx, key).Err()
}
