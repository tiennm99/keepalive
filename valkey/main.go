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

	valkeyURI, isExist := os.LookupEnv("VALKEY_URI")
	if !isExist {
		log.Fatal("Warning: VALKEY_URI not set!")
		return
	}

	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{valkeyURI}})
	if err != nil {
		panic(err)
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
