package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/tiennm99/keepalive/adapter"
)

// retryDelay is how long a service waits after any failure (connect or tick)
// before reconnecting, so a broken service logs at most once per delay.
var retryDelay = time.Minute

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

			if err := a.Connect(ctx); err != nil {
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

// runConnectedService ticks until the context ends (returning nil) or an
// increment fails (returning that error).
func runConnectedService(ctx context.Context, svc runningService) error {
	ticker := time.NewTicker(svc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			tickCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			count, err := svc.adapter.Increment(tickCtx)
			cancel()
			if err != nil {
				return err
			}
			log.Printf("[%s] counter: %d", svc.config.Name, count)
		}
	}
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

// redactURLError drops the raw URL from a *url.Error. Drivers return one when
// a connection URL fails to parse, and its message repeats the URL, password
// included.
func redactURLError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return err
	}
	var escapeErr url.EscapeError
	if errors.As(urlErr.Err, &escapeErr) {
		// The message quotes the bad escape, which may be part of a password.
		return errors.New("invalid connection URL: invalid percent-escape; percent-encode special characters in the user name and password")
	}
	return fmt.Errorf("invalid connection URL: %w", urlErr.Err)
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
