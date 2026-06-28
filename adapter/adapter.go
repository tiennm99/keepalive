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
	"strconv"
	"strings"
	"time"
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

func (c Config) OptionalDuration(name string, def time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(c[name])
	if value == "" {
		return def, nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		if d <= 0 {
			return 0, fmt.Errorf("config %s must be greater than zero", name)
		}
		return d, nil
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("config %s must be a duration like 30s or an integer number of seconds", name)
	}
	d := time.Duration(seconds) * time.Second
	if d <= 0 {
		return 0, fmt.Errorf("config %s must be greater than zero", name)
	}
	return d, nil
}

func (c Config) OptionalUint64(name string, def uint64) (uint64, error) {
	value := strings.TrimSpace(c[name])
	if value == "" {
		return def, nil
	}
	out, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("config %s must be an unsigned integer", name)
	}
	return out, nil
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
