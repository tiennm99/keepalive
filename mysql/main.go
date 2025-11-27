package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	dataSourceName, isExist := os.LookupEnv("DATA_SOURCE_NAME")
	if !isExist {
		log.Fatal("Warning: DATA_SOURCE_NAME not set!")
		return
	}

	rootCertPool := x509.NewCertPool()
	pem, err := os.ReadFile("ca.pem")
	if err != nil {
		log.Fatal(err)
	}
	rootCertPool.AppendCertsFromPEM(pem)

	err = mysql.RegisterTLSConfig("custom", &tls.Config{
		RootCAs: rootCertPool,
	})
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		panic(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

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

	_, err = tx.ExecContext(ctx,
		"UPDATE `keepalive` SET `value` = `value` + 1 WHERE `key` = 'counter'",
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	var value int64
	err = tx.QueryRowContext(ctx,
		"SELECT `value` FROM `keepalive` WHERE `key` = 'counter'",
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
