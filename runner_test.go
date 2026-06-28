package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tiennm99/keepalive/adapter"
)

type retryConnectAdapter struct {
	connects  *atomic.Int32
	connected chan<- struct{}
	once      *sync.Once
}

func (a *retryConnectAdapter) Connect(context.Context) error {
	if a.connects.Add(1) == 1 {
		return errors.New("connect failed")
	}
	a.once.Do(func() { close(a.connected) })
	return nil
}

func (a *retryConnectAdapter) Increment(context.Context) (int64, error) {
	return 0, nil
}

func (a *retryConnectAdapter) Close(context.Context) error {
	return nil
}

func TestRunServiceRetriesConnectFailure(t *testing.T) {
	oldReconnectDelay := reconnectDelay
	reconnectDelay = 10 * time.Millisecond
	defer func() { reconnectDelay = oldReconnectDelay }()

	var connects atomic.Int32
	connected := make(chan struct{})
	var once sync.Once
	adapter.Registry["retry-test"] = func(adapter.Config) (adapter.Adapter, error) {
		return &retryConnectAdapter{connects: &connects, connected: connected, once: &once}, nil
	}
	defer delete(adapter.Registry, "retry-test")

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	runService(ctx, &wg, serviceConfig{
		Name:        "retry-test",
		AdapterType: "retry-test",
		Interval:    time.Hour,
		Config:      adapter.Config{},
	})

	select {
	case <-connected:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("service did not retry and connect")
	}

	cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("service did not stop after context cancellation")
	}

	if got := connects.Load(); got != 2 {
		t.Fatalf("connect attempts = %d, want 2", got)
	}
}
