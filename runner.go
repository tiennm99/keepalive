package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tiennm99/keepalive/adapter"
)

// retryDelay is how long a service waits after any failure (connect or tick)
// before reconnecting, so a broken service logs at most once per delay.
var retryDelay = time.Minute

// connectTimeout bounds one connect attempt, including adapter
// initialization. It exceeds Couchbase's default ready_timeout of 30s.
var connectTimeout = time.Minute

type runningService struct {
	config  serviceConfig
	adapter adapter.Adapter
}

func runService(ctx context.Context, wg *sync.WaitGroup, config serviceConfig) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			a, err := adapter.New(config.AdapterType, config.Config)
			if err != nil {
				log.Printf("[%s] init adapter: %v", config.Name, redactURLError(err))
				return
			}

			connectCtx, cancel := context.WithTimeout(ctx, connectTimeoutFor(a))
			err = a.Connect(connectCtx)
			cancel()
			if err != nil {
				closeService(ctx, config.Name, a)
				if ctx.Err() != nil {
					return
				}
				log.Printf("[%s] connect: %v; retrying in %s", config.Name, redactURLError(err), retryDelay)
			} else {
				log.Printf("[%s] keepalive: %s every %s", config.Name, config.AdapterType, config.Interval)
				err := runConnectedService(ctx, runningService{config: config, adapter: a})
				closeService(ctx, config.Name, a)
				if ctx.Err() != nil {
					return
				}
				// Reconnecting re-runs each adapter's initialization, which
				// recreates a dropped table, row, or collection.
				log.Printf("[%s] increment: %v; reconnecting in %s", config.Name, redactURLError(err), retryDelay)
			}
			if !waitContext(ctx, retryDelay) {
				return
			}
		}
	}()
}

// runConnectedService increments once right away, so every (re)start writes
// even when the interval outlasts the process, then once per interval. It
// returns nil when the context ends, or the first increment error.
func runConnectedService(ctx context.Context, svc runningService) error {
	if err := incrementOnce(ctx, svc); err != nil {
		return err
	}

	ticker := time.NewTicker(svc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := incrementOnce(ctx, svc); err != nil {
				return err
			}
		}
	}
}

func incrementOnce(ctx context.Context, svc runningService) error {
	tickCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	count, err := svc.adapter.Increment(tickCtx)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	log.Printf("[%s] counter: %d", svc.config.Name, count)
	return nil
}

func connectTimeoutFor(a adapter.Adapter) time.Duration {
	if t, ok := a.(adapter.ConnectTimeouter); ok && t.ConnectTimeout() > connectTimeout {
		return t.ConnectTimeout()
	}
	return connectTimeout
}

func closeService(_ context.Context, name string, a adapter.Adapter) {
	if a == nil {
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Close(shutdownCtx); err != nil {
		log.Printf("[%s] close: %v", name, err)
	}
}

// redactURLError hides connection-string parse errors that quote part of
// the string, password included: *url.Error repeats the whole URL, an escape
// error quotes the bad escape, and lib/pq quotes the token after a stray space
// in a key=value DSN.
func redactURLError(err error) error {
	var escapeErr url.EscapeError
	if errors.As(err, &escapeErr) {
		return errors.New("invalid connection URL: invalid percent-escape; percent-encode special characters in the user name and password")
	}
	if strings.Contains(err.Error(), `missing "=" after`) {
		return errors.New("invalid key=value DSN: quote values that contain spaces, like password='a b'")
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("invalid connection URL: %w", urlErr.Err)
	}
	return err
}

func waitContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
