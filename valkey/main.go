package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/valkey-io/valkey-go"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	valkeyUrl, isExist := os.LookupEnv("VALKEY_URL")
	if !isExist {
		log.Fatal("Warning: VALKEY_URL not set!")
		return
	}

	opt, err := valkey.ParseURL(valkeyUrl)
	if err != nil {
		log.Fatal(err)
	}

	client, err := valkey.NewClient(opt)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := incrementCounter(ctx, client); err != nil {
					log.Printf("Keepalive increment error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	defer func() {
		cancel()
		client.Close()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func incrementCounter(ctx context.Context, client valkey.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	increment, err := client.Do(ctx, client.B().Incr().Key("counter").Build()).AsInt64()
	if err != nil {
		return err
	}
	log.Printf("Counter : %d\n", increment)
	return nil
}
