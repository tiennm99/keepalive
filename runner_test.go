package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
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
	oldRetryDelay := retryDelay
	retryDelay = 10 * time.Millisecond
	defer func() { retryDelay = oldRetryDelay }()

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

type failingIncrementAdapter struct {
	connects *atomic.Int32
	closes   *atomic.Int32
}

func (a *failingIncrementAdapter) Connect(context.Context) error {
	a.connects.Add(1)
	return nil
}

func (a *failingIncrementAdapter) Increment(context.Context) (int64, error) {
	return 0, errors.New("relation \"keepalive\" does not exist")
}

func (a *failingIncrementAdapter) Close(context.Context) error {
	a.closes.Add(1)
	return nil
}

func TestRunServiceReconnectsAfterIncrementFailure(t *testing.T) {
	oldRetryDelay := retryDelay
	retryDelay = 10 * time.Millisecond
	defer func() { retryDelay = oldRetryDelay }()

	var connects, closes atomic.Int32
	adapter.Registry["fail-tick-test"] = func(adapter.Config) (adapter.Adapter, error) {
		return &failingIncrementAdapter{connects: &connects, closes: &closes}, nil
	}
	defer delete(adapter.Registry, "fail-tick-test")

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	runService(ctx, &wg, serviceConfig{
		Name:        "fail-tick-test",
		AdapterType: "fail-tick-test",
		Interval:    5 * time.Millisecond,
		Config:      adapter.Config{},
	})

	deadline := time.After(time.Second)
	for connects.Load() < 3 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("connect attempts = %d, want at least 3", connects.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	wg.Wait()

	if closes.Load() < connects.Load()-1 {
		t.Fatalf("closes = %d for %d connects; each failed session must be closed", closes.Load(), connects.Load())
	}
}

func TestRedactURLErrorHidesPassword(t *testing.T) {
	for _, raw := range []string{
		"postgres://user:S3CR%ETpw@db.example.com/keepalive",
		"redis://user:S3CRETpw@cache.example.com:port/0",
	} {
		_, err := url.Parse(raw)
		if err == nil {
			t.Fatalf("%s: want parse error", raw)
		}
		got := redactURLError(fmt.Errorf("connect: %w", err)).Error()
		if strings.Contains(got, "S3CR") {
			t.Fatalf("redacted error leaks password: %s", got)
		}
		if !strings.Contains(got, "invalid connection URL") {
			t.Fatalf("redacted error = %q, want a URL hint", got)
		}
	}
}

func TestRedactURLErrorKeepsOtherErrors(t *testing.T) {
	err := errors.New("dial tcp: connection refused")
	if got := redactURLError(err); got != err {
		t.Fatalf("redactURLError changed a non-URL error: %v", got)
	}
}

type countingAdapter struct{ increments *atomic.Int32 }

func (a *countingAdapter) Connect(context.Context) error { return nil }
func (a *countingAdapter) Increment(context.Context) (int64, error) {
	return int64(a.increments.Add(1)), nil
}
func (a *countingAdapter) Close(context.Context) error { return nil }

func TestRunServiceIncrementsRightAfterConnect(t *testing.T) {
	var increments atomic.Int32
	adapter.Registry["first-tick-test"] = func(adapter.Config) (adapter.Adapter, error) {
		return &countingAdapter{increments: &increments}, nil
	}
	defer delete(adapter.Registry, "first-tick-test")

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	runService(ctx, &wg, serviceConfig{
		Name:        "first-tick-test",
		AdapterType: "first-tick-test",
		Interval:    time.Hour,
		Config:      adapter.Config{},
	})

	deadline := time.After(time.Second)
	for increments.Load() == 0 {
		select {
		case <-deadline:
			cancel()
			t.Fatal("no increment before the first interval")
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	wg.Wait()
}
