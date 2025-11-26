package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	redisUrl, isExist := os.LookupEnv("REDIS_URL")
	if !isExist {
		log.Fatal("Warning: REDIS_URL not set!")
		return
	}

	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := incrementCounter(ctx, rdb); err != nil {
					log.Printf("Keepalive increment error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	defer func() {
		cancel()
		if err := rdb.Close(); err != nil {
			log.Printf("Error closing rdb: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func incrementCounter(ctx context.Context, rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	counterKey := "counter"
	increment, err := rdb.Incr(ctx, counterKey).Result()
	if err != nil {
		return err
	}
	log.Printf("Counter : %d\n", increment)
	return nil
}
