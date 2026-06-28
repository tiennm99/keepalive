package adapter

import (
	"context"
	"strings"

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
			key: redisCounterKey(cfg.Optional("counter_key", "counter"), cfg.Optional("namespace", "")),
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
	if err := a.client.Ping(ctx).Err(); err != nil {
		a.client.Close()
		return err
	}
	if err := a.client.SetNX(ctx, a.key, 0, 0).Err(); err != nil {
		a.client.Close()
		return err
	}
	return nil
}

func (a *redisAdapter) Increment(ctx context.Context) (int64, error) {
	return a.client.Incr(ctx, a.key).Result()
}

func (a *redisAdapter) Close(_ context.Context) error {
	if a.client == nil {
		return nil
	}
	return a.client.Close()
}

func redisCounterKey(counterKey, namespace string) string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return counterKey
	}
	return namespace + ":" + counterKey
}
