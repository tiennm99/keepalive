package adapter

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func init() {
	Registry["redis"] = func(cfg Config) (Adapter, error) {
		url, err := cfg.Required("url")
		if err != nil {
			return nil, err
		}
		return &redisAdapter{
			url: url,
			key: cfg.Optional("counter_key", "counter"),
		}, nil
	}
}

type redisAdapter struct {
	client *redis.Client
	url    string
	key    string
}

func (a *redisAdapter) Connect(ctx context.Context) error {
	opt, err := redis.ParseURL(a.url)
	if err != nil {
		return err
	}
	a.client = redis.NewClient(opt)
	return a.client.Ping(ctx).Err()
}

func (a *redisAdapter) Increment(ctx context.Context) (int64, error) {
	return a.client.Incr(ctx, a.key).Result()
}

func (a *redisAdapter) Close(_ context.Context) error {
	return a.client.Close()
}
