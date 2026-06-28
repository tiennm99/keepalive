// Package adapter defines the contract every database driver implements.
//
// An Adapter connects to a single backing store, increments a counter on every
// keepalive tick (the cheapest write that proves the cluster is live), and
// releases its resources on Close. New databases plug in by adding a file in
// this package and registering a factory in Registry.
package adapter

import (
	"context"
	"fmt"
)

type Adapter interface {
	Connect(ctx context.Context) error
	Increment(ctx context.Context) (int64, error)
	Close(ctx context.Context) error
}

type Config map[string]string

func (c Config) Required(name string) (string, error) {
	v, ok := c[name]
	if !ok || v == "" {
		return "", fmt.Errorf("config %s is required", name)
	}
	return v, nil
}

func (c Config) Optional(name, def string) string {
	if v, ok := c[name]; ok && v != "" {
		return v
	}
	return def
}

type Factory func(Config) (Adapter, error)

var Registry = map[string]Factory{}

func New(adapterType string, cfg Config) (Adapter, error) {
	f, ok := Registry[adapterType]
	if !ok {
		return nil, fmt.Errorf("unknown adapter %q (known: %v)", adapterType, Known())
	}
	return f(cfg)
}

func Known() []string {
	out := make([]string, 0, len(Registry))
	for k := range Registry {
		out = append(out, k)
	}
	return out
}
