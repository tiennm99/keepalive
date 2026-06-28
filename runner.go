package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/tiennm99/keepalive/adapter"
)

type runningService struct {
	config  serviceConfig
	adapter adapter.Adapter
}

func runService(ctx context.Context, wg *sync.WaitGroup, svc runningService) {
	wg.Add(1)
	go func() {
		defer wg.Done()

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
	}()
}

func closeServices(services []runningService) {
	for _, svc := range services {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := svc.adapter.Close(shutdownCtx); err != nil {
			log.Printf("[%s] close: %v", svc.config.Name, err)
		}
		cancel()
	}
}
