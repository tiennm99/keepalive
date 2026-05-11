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

type Factory func() (Adapter, error)

var Registry = map[string]Factory{}

func New(dbType string) (Adapter, error) {
	f, ok := Registry[dbType]
	if !ok {
		return nil, fmt.Errorf("unknown DB_TYPE %q (known: %v)", dbType, Known())
	}
	return f()
}

func Known() []string {
	out := make([]string, 0, len(Registry))
	for k := range Registry {
		out = append(out, k)
	}
	return out
}

func envOrFail(name string) (string, error) {
	v, ok := lookupEnv(name)
	if !ok || v == "" {
		return "", fmt.Errorf("env %s is required", name)
	}
	return v, nil
}
