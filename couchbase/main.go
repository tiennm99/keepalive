package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Uncomment following line to enable logging
	// gocb.SetLogger(gocb.VerboseStdioLogger())

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Update this to your cluster details
	connectionString, isExist := os.LookupEnv("COUCHBASE_CONNECTION_STRING")
	if !isExist {
		log.Fatal("Warning: COUCHBASE_CONNECTION_STRING not set!")
		return
	}
	username, isExist := os.LookupEnv("COUCHBASE_USERNAME")
	if !isExist {
		log.Fatal("Warning: COUCHBASE_USERNAME not set!")
		return
	}
	password, isExist := os.LookupEnv("COUCHBASE_PASSWORD")
	if !isExist {
		log.Fatal("Warning: COUCHBASE_PASSWORD not set!")
		return
	}
	bucketName, isExist := os.LookupEnv("COUCHBASE_BUCKET_NAME")
	if !isExist {
		log.Fatal("Warning: COUCHBASE_BUCKET_NAME not set!")
		return
	}
	scopeName, isExist := os.LookupEnv("COUCHBASE_SCOPE_NAME")
	if !isExist {
		log.Fatal("Warning: COUCHBASE_SCOPE_NAME not set!")
		return
	}
	collectionName, isExist := os.LookupEnv("COUCHBASE_COLLECTION_NAME")
	if !isExist {
		log.Fatal("Warning: COUCHBASE_COLLECTION_NAME not set!")
		return
	}

	options := gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
	}

	// Sets a pre-configured profile called "wan-development" to help avoid latency issues
	// when accessing Capella from a different Wide Area Network
	// or Availability Zone (e.g. your laptop).
	if err := options.ApplyProfile(gocb.ClusterConfigProfileWanDevelopment); err != nil {
		log.Fatal(err)
	}

	// Initialize the Connection
	cluster, err := gocb.Connect(connectionString, options)
	if err != nil {
		log.Fatal(err)
	}

	bucket := cluster.Bucket(bucketName)

	err = bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Get a reference to the default collection, required for older Couchbase server versions
	// col := bucket.DefaultCollection()

	col := bucket.Scope(scopeName).Collection(collectionName)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := incrementCounter(col); err != nil {
					log.Printf("Keepalive increment error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	defer func() {
		cancel()
		if err := cluster.Close(nil); err != nil {
			log.Printf("Error closing cluster: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func incrementCounter(col *gocb.Collection) error {
	counterDocId := "counter"
	// Increment by 1, creating doc if needed.
	// By using `Initial: 1` we set the starting count(non-negative) to 1 if the document needs to be created.
	// If it already exists, the count will increase by the amount provided in the Delta option(i.e 1).
	increment, err := col.Binary().Increment(counterDocId, &gocb.IncrementOptions{Initial: 1, Delta: 1})
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Printf("Counter : %d\n", increment.Content())
	return nil
}
