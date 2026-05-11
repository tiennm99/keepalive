package adapter

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func init() {
	Registry["redis"] = func() (Adapter, error) { return &redisAdapter{}, nil }
}

type redisAdapter struct {
	client *redis.Client
	key    string
}

func (a *redisAdapter) Connect(ctx context.Context) error {
	url, err := envOrFail("REDIS_URL")
	if err != nil {
		return err
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return err
	}
	a.client = redis.NewClient(opt)
	a.key = envOr("COUNTER_KEY", "counter")
	return a.client.Ping(ctx).Err()
}

func (a *redisAdapter) Increment(ctx context.Context) (int64, error) {
	return a.client.Incr(ctx, a.key).Result()
}

func (a *redisAdapter) Close(_ context.Context) error {
	return a.client.Close()
}
