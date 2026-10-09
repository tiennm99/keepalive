package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	configPath, err := defaultConfigFile()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	services, err := loadConfigFile(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, svcConfig := range services {
		runService(ctx, &wg, svcConfig)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	cancel()
	waitShutdown(&wg, shutdownTimeout)
}

// shutdownTimeout stays under Docker's default 10s stop grace period. A
// driver stuck in a handshake that ignores cancellation (lib/pq) would
// otherwise hold the process until Docker sends SIGKILL.
const shutdownTimeout = 7 * time.Second

func waitShutdown(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		log.Printf("shutdown: services still stopping after %s; exiting", timeout)
		return false
	}
}
