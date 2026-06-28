package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/tiennm99/keepalive/adapter"
)

var reconnectDelay = 10 * time.Second

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
				log.Printf("[%s] init adapter: %v", config.Name, err)
				return
			}

			if err := a.Connect(ctx); err != nil {
				if ctx.Err() != nil {
					closeService(ctx, config.Name, a)
					return
				}
				log.Printf("[%s] connect: %v", config.Name, err)
				closeService(ctx, config.Name, a)
				if !waitContext(ctx, reconnectDelay) {
					return
				}
				continue
			}

			log.Printf("[%s] keepalive: %s every %s", config.Name, config.AdapterType, config.Interval)
			runConnectedService(ctx, runningService{config: config, adapter: a})
			closeService(ctx, config.Name, a)
			return
		}
	}()
}

func runConnectedService(ctx context.Context, svc runningService) {
	ticker := time.NewTicker(svc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			count, err := svc.adapter.Increment(tickCtx)
			cancel()
			if err != nil {
				log.Printf("[%s] increment: %v", svc.config.Name, err)
				continue
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
