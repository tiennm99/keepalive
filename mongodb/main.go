package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	uri, isExist := os.LookupEnv("MONGODB_URI")
	if !isExist {
		log.Fatal("Warning: MONGODB_URI not set!")
		return
	}
	databaseName, isExist := os.LookupEnv("MONGODB_DATABASE")
	if !isExist {
		log.Fatal("Warning: MONGODB_DATABASE not set!")
		return
	}
	collectionName, isExist := os.LookupEnv("MONGODB_COLLECTION")
	if !isExist {
		log.Fatal("Warning: MONGODB_COLLECTION not set!")
		return
	}

	client, err := mongo.Connect(options.Client().
		ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	coll := client.Database(databaseName).Collection(collectionName)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := incrementCounter(ctx, coll); err != nil {
					log.Printf("Keepalive increment error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	defer func() {
		cancel()
		if err := client.Disconnect(context.TODO()); err != nil {
			log.Printf("Error closing client: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func incrementCounter(ctx context.Context, coll *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	counterDocID := "counter"

	update := bson.M{
		"$inc": bson.M{"count": 1},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)
	var out struct {
		Count int64 `bson:"count"`
	}
	err := coll.FindOneAndUpdate(ctx, bson.M{"_id": counterDocID}, update, opts).Decode(&out)

	if err != nil {
		return err
	}
	log.Printf("Counter : %d\n", out.Count)
	return nil
}
