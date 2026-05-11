package adapter

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func init() {
	Registry["mongodb"] = func() (Adapter, error) { return &mongoAdapter{}, nil }
	Registry["mongo"] = Registry["mongodb"]
}

type mongoAdapter struct {
	client *mongo.Client
	coll   *mongo.Collection
	docID  string
}

func (a *mongoAdapter) Connect(ctx context.Context) error {
	uri, err := envOrFail("MONGODB_URI")
	if err != nil {
		return err
	}
	dbName, err := envOrFail("MONGODB_DATABASE")
	if err != nil {
		return err
	}
	collName, err := envOrFail("MONGODB_COLLECTION")
	if err != nil {
		return err
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	a.client = client
	a.coll = client.Database(dbName).Collection(collName)
	a.docID = envOr("COUNTER_KEY", "counter")
	return client.Ping(ctx, nil)
}

func (a *mongoAdapter) Increment(ctx context.Context) (int64, error) {
	update := bson.M{"$inc": bson.M{"count": 1}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var out struct {
		Count int64 `bson:"count"`
	}
	err := a.coll.FindOneAndUpdate(ctx, bson.M{"_id": a.docID}, update, opts).Decode(&out)
	if err != nil {
		return 0, err
	}
	return out.Count, nil
}

func (a *mongoAdapter) Close(ctx context.Context) error {
	return a.client.Disconnect(ctx)
}
