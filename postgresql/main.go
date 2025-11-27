package main

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	serviceURI, isExist := os.LookupEnv("SERVICE_URI")
	if !isExist {
		log.Fatal("Warning: SERVICE_URI not set!")
		return
	}
	conn, _ := url.Parse(serviceURI)
	conn.RawQuery = "sslmode=verify-ca;sslrootcert=ca.pem"

	db, err := sql.Open("postgres", conn.String())
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
				if err := incrementCounter(ctx, db); err != nil {
					log.Printf("Keepalive increment error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	defer func() {
		cancel()
		if err := db.Close(); err != nil {
			log.Printf("Close error: %v", err)
			return
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func incrementCounter(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}

	var value int64
	err = tx.QueryRowContext(ctx,
		"UPDATE keepalive SET value = value + 1 WHERE key = 'counter' RETURNING value",
	).Scan(&value)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("Counter: %d\n", value)
	return nil
}
