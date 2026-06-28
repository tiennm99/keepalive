package adapter

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func init() {
	Registry["mongodb"] = func(cfg Config) (Adapter, error) {
		uri, err := cfg.Required("uri")
		if err != nil {
			return nil, err
		}
		dbName, err := cfg.Required("database")
		if err != nil {
			return nil, err
		}
		collName, err := cfg.Required("collection")
		if err != nil {
			return nil, err
		}
		return &mongoAdapter{
			uri:      uri,
			dbName:   dbName,
			collName: collName,
			docID:    cfg.Optional("counter_key", "counter"),
		}, nil
	}
	Registry["mongo"] = Registry["mongodb"]
}

type mongoAdapter struct {
	client   *mongo.Client
	coll     *mongo.Collection
	uri      string
	dbName   string
	collName string
	docID    string
}

func (a *mongoAdapter) Connect(ctx context.Context) error {
	client, err := mongo.Connect(options.Client().ApplyURI(a.uri))
	if err != nil {
		return err
	}
	a.client = client
	a.coll = client.Database(a.dbName).Collection(a.collName)
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
