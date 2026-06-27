package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tiennm99/keepalive/adapter"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("note: .env not loaded, relying on process env")
	}

	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		log.Fatalf("DB_TYPE is required (known: %v)", adapter.Known())
	}

	interval := parseInterval(os.Getenv("INTERVAL"), time.Minute)

	a, err := adapter.New(dbType)
	if err != nil {
		log.Fatalf("init adapter: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := a.Connect(ctx); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		if err := a.Close(shutdownCtx); err != nil {
			log.Printf("close: %v", err)
		}
	}()

	log.Printf("keepalive: %s every %s", dbType, interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tickCtx, tcancel := context.WithTimeout(ctx, 3*time.Second)
				count, err := a.Increment(tickCtx)
				tcancel()
				if err != nil {
					log.Printf("increment: %v", err)
					continue
				}
				log.Printf("counter: %d", count)
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func parseInterval(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Second
	}
	return def
}
