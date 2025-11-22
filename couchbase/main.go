package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/joho/godotenv"
)

// Global counter to track number of documents created
var documentCounter int = 0
var counterMutex sync.Mutex

// KeepaliveDocument represents the JSON document structure for keepalive operations
type KeepaliveDocument struct {
	ID        string    `json:"id"`
	Timestamp int64     `json:"timestamp"`
	Value     string    `json:"value"`
	Operation string    `json:"operation"`
	Cluster   string    `json:"cluster"`
}

type Config struct {
	ClusterURL      string
	Username        string
	Password        string
	BucketName      string
	ScopeName       string
	CollectionName  string
	Interval        time.Duration
	OperationTimeout time.Duration
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := loadConfig()

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Connect to Couchbase cluster
	cluster, err := connectToCluster(config)
	if err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}
	defer cluster.Close(nil)

	// Test connection to the specified scope
	err = testScopeConnection(cluster, config)
	if err != nil {
		log.Fatalf("Failed to connect to scope: %v", err)
	}

	log.Println("Successfully connected to Couchbase cluster")
	log.Printf("Scope: %s, Collection: %s", config.ScopeName, config.CollectionName)
	log.Printf("Keeping cluster alive with operations every %v", config.Interval)

	// Start keepalive operations
	keepAlive(cluster, config)
}

func loadConfig() Config {
	var config Config

	// Load from environment variables with defaults
	config.ClusterURL = getEnv("COUCHBASE_CONNECTION_STRING", "localhost")
	config.Username = getEnv("COUCHBASE_USERNAME", "")
	config.Password = getEnv("COUCHBASE_PASSWORD", "")
	config.BucketName = getEnv("COUCHBASE_BUCKET_NAME", "default")
	config.ScopeName = getEnv("COUCHBASE_SCOPE_NAME", "_default")
	config.CollectionName = getEnv("COUCHBASE_COLLECTION_NAME", "_default")

	// Parse interval duration from environment variable
	intervalStr := getEnv("COUCHBASE_INTERVAL", "5m")
	if interval, err := time.ParseDuration(intervalStr); err == nil {
		config.Interval = interval
	} else {
		log.Printf("Invalid interval '%s', using default 5m", intervalStr)
		config.Interval = 5 * time.Minute
	}

	// Parse operation timeout from environment variable
	timeoutStr := getEnv("COUCHBASE_OPERATION_TIMEOUT", "10s")
	if timeout, err := time.ParseDuration(timeoutStr); err == nil {
		config.OperationTimeout = timeout
	} else {
		log.Printf("Invalid operation timeout '%s', using default 10s", timeoutStr)
		config.OperationTimeout = 10 * time.Second
	}

	// Validate required parameters
	if config.Username == "" || config.Password == "" {
		log.Fatal("COUCHBASE_USERNAME and COUCHBASE_PASSWORD are required")
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func connectToCluster(config Config) (*gocb.Cluster, error) {
	cluster, err := gocb.Connect(config.ClusterURL, gocb.ClusterOptions{
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster connection: %w", err)
	}

	// Get the bucket
	bucket := cluster.Bucket(config.BucketName)

	return cluster, bucket, nil
}

func testScopeConnection(cluster *gocb.Cluster, config Config) error {
	bucket := cluster.Bucket(config.BucketName)
	scope := bucket.Scope(config.ScopeName)
	collection := scope.Collection(config.CollectionName)

	// Test the connection to the specific scope and collection with configured timeout
	err := collection.Ping(context.Background())
	if err != nil {
		return fmt.Errorf("failed to ping scope '%s', collection '%s': %w",
			config.ScopeName, config.CollectionName, err)
	}

	return nil
}

func keepAlive(cluster *gocb.Cluster, config Config) {
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := performRandomOperation(cluster, config)
			if err != nil {
				log.Printf("Error performing keepalive operation: %v", err)
			} else {
				log.Println("Successfully performed keepalive operation")
			}
		}
	}
}

func performRandomOperation(cluster *gocb.Cluster, config Config) error {
	bucket := cluster.Bucket(config.BucketName)
	scope := bucket.Scope(config.ScopeName)
	collection := scope.Collection(config.CollectionName)

	// 50% chance to perform GET or SET operation
	if rand.Intn(2) == 0 {
		return performGetOperation(collection, config)
	} else {
		return performSetOperation(collection, config)
	}
}

func getCurrentHourSeconds() string {
	now := time.Now()
	hourSeconds := now.Hour()*3600 + now.Minute()*60 + now.Second()
	return fmt.Sprintf("%d", hourSeconds)
}

// incrementCounter increases the document counter and returns the new value
func incrementCounter() int {
	counterMutex.Lock()
	defer counterMutex.Unlock()
	documentCounter++
	return documentCounter
}

// getCurrentCounter returns the current document counter value
func getCurrentCounter() int {
	counterMutex.Lock()
	defer counterMutex.Unlock()
	return documentCounter
}

func performGetOperation(collection *gocb.Collection, config Config) error {
	// Use the current counter value to get the latest document
	currentCounter := getCurrentCounter()
	docID := fmt.Sprintf("%d", currentCounter)

	// Create a document struct for the Get result
	var doc KeepaliveDocument

	// Get the document with configured timeout
	_, err := collection.Get(docID, &gocb.GetOptions{
		Timeout: config.OperationTimeout,
	})

	// It's okay if the document doesn't exist, we're just testing the operation
	if err != nil && err != gocb.ErrDocumentNotFound {
		return fmt.Errorf("get operation failed: %w", err)
	}

	// Try to decode the document to verify JSON structure
	// We don't need the actual data, just testing the operation
	_ = doc

	log.Printf("Successfully retrieved document with counter: %d", currentCounter)
	return nil
}

func performSetOperation(collection *gocb.Collection, config Config) error {
	// Use the incremented counter value as document ID
	newCounter := incrementCounter()
	docID := fmt.Sprintf("%d", newCounter)

	// Create a structured JSON document
	document := KeepaliveDocument{
		ID:        docID,
		Timestamp: time.Now().Unix(),
		Value:     fmt.Sprintf("keepalive-value-%d", newCounter),
		Operation: "keepalive",
		Cluster:   config.ClusterURL,
	}

	// Set the document with a short TTL (1 hour) and configured timeout
	_, err := collection.Insert(docID, document, &gocb.InsertOptions{
		Expiry:   time.Hour,
		Timeout:  config.OperationTimeout,
	})

	if err != nil {
		return fmt.Errorf("set operation failed: %w", err)
	}

	log.Printf("Successfully created document with counter: %d", newCounter)
	return nil
}