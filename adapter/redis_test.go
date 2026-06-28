package adapter

import "testing"

func TestRedisFactoryUsesRootCounterKeyWithoutNamespace(t *testing.T) {
	got := newRedisAdapterForTest(t, Config{
		"url":         "redis://cache.example.com:6379",
		"counter_key": "counter",
	})

	if got.key != "counter" {
		t.Fatalf("key = %q, want %q", got.key, "counter")
	}
}

func TestRedisFactoryPrefixesCounterKeyWithNamespace(t *testing.T) {
	got := newRedisAdapterForTest(t, Config{
		"url":         "redis://cache.example.com:6379",
		"counter_key": "counter",
		"namespace":   "keepalive",
	})

	if got.key != "keepalive:counter" {
		t.Fatalf("key = %q, want %q", got.key, "keepalive:counter")
	}
}

func TestRedisFactoryTreatsBlankNamespaceAsRoot(t *testing.T) {
	got := newRedisAdapterForTest(t, Config{
		"url":         "redis://cache.example.com:6379",
		"counter_key": "counter",
		"namespace":   "  ",
	})

	if got.key != "counter" {
		t.Fatalf("key = %q, want %q", got.key, "counter")
	}
}

func newRedisAdapterForTest(t *testing.T, cfg Config) *redisAdapter {
	t.Helper()

	adapterValue, err := Registry["redis"](cfg)
	if err != nil {
		t.Fatalf("redis factory returned error: %v", err)
	}
	got, ok := adapterValue.(*redisAdapter)
	if !ok {
		t.Fatalf("redis factory returned %T, want *redisAdapter", adapterValue)
	}
	return got
}
