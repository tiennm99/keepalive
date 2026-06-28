package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/tiennm99/keepalive/adapter"
)

const defaultConfigFile = "keepalive.yaml"

func main() {
	services, err := loadConfigFile(defaultConfigFile)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	running := make([]runningService, 0, len(services))
	for _, svcConfig := range services {
		a, err := adapter.New(svcConfig.AdapterType, svcConfig.Config)
		if err != nil {
			cancel()
			closeServices(running)
			log.Fatalf("[%s] init adapter: %v", svcConfig.Name, err)
		}
		if err := a.Connect(ctx); err != nil {
			cancel()
			closeServices(running)
			log.Fatalf("[%s] connect: %v", svcConfig.Name, err)
		}
		running = append(running, runningService{config: svcConfig, adapter: a})
		log.Printf("[%s] keepalive: %s every %s", svcConfig.Name, svcConfig.AdapterType, svcConfig.Interval)
	}

	var wg sync.WaitGroup
	for _, svc := range running {
		runService(ctx, &wg, svc)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	cancel()
	wg.Wait()
	closeServices(running)
}
