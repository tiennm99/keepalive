package adapter

import (
	"context"

	"github.com/valkey-io/valkey-go"
)

func init() {
	Registry["valkey"] = func(cfg Config) (Adapter, error) {
		url, err := cfg.Required("url")
		if err != nil {
			return nil, err
		}
		return &valkeyAdapter{
			url: url,
			key: cfg.Optional("counter_key", "counter"),
		}, nil
	}
}

type valkeyAdapter struct {
	client valkey.Client
	url    string
	key    string
}

func (a *valkeyAdapter) Connect(ctx context.Context) error {
	opt, err := valkey.ParseURL(a.url)
	if err != nil {
		return err
	}
	client, err := valkey.NewClient(opt)
	if err != nil {
		return err
	}
	a.client = client
	if err := a.client.Do(ctx, a.client.B().Setnx().Key(a.key).Value("0").Build()).Error(); err != nil {
		a.client.Close()
		return err
	}
	return nil
}

func (a *valkeyAdapter) Increment(ctx context.Context) (int64, error) {
	return a.client.Do(ctx, a.client.B().Incr().Key(a.key).Build()).AsInt64()
}

func (a *valkeyAdapter) Close(_ context.Context) error {
	if a.client == nil {
		return nil
	}
	a.client.Close()
	return nil
}
