package adapter

import (
	"context"

	"github.com/valkey-io/valkey-go"
)

func init() {
	Registry["valkey"] = func() (Adapter, error) { return &valkeyAdapter{}, nil }
}

type valkeyAdapter struct {
	client valkey.Client
	key    string
}

func (a *valkeyAdapter) Connect(_ context.Context) error {
	url, err := envOrFail("VALKEY_URL")
	if err != nil {
		return err
	}
	opt, err := valkey.ParseURL(url)
	if err != nil {
		return err
	}
	client, err := valkey.NewClient(opt)
	if err != nil {
		return err
	}
	a.client = client
	a.key = envOr("COUNTER_KEY", "counter")
	return nil
}

func (a *valkeyAdapter) Increment(ctx context.Context) (int64, error) {
	return a.client.Do(ctx, a.client.B().Incr().Key(a.key).Build()).AsInt64()
}

func (a *valkeyAdapter) Close(_ context.Context) error {
	a.client.Close()
	return nil
}
