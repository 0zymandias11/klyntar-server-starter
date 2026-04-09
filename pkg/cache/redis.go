package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"example.com/klyntar-server/config"
	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

func New(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		slog.Error("cannot connect to Redis", "error", err)
		return nil, fmt.Errorf("failed to connect to Redis %s", err)
	}

	slog.Info("Successfully connected to Redis")
	return client, nil
}
