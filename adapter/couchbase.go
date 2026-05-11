package adapter

import (
	"context"
	"time"

	"github.com/couchbase/gocb/v2"
)

func init() {
	Registry["couchbase"] = func() (Adapter, error) { return &couchbaseAdapter{}, nil }
}

type couchbaseAdapter struct {
	cluster *gocb.Cluster
	coll    *gocb.Collection
	docID   string
}

func (a *couchbaseAdapter) Connect(_ context.Context) error {
	conn, err := envOrFail("COUCHBASE_CONNECTION_STRING")
	if err != nil {
		return err
	}
	user, err := envOrFail("COUCHBASE_USERNAME")
	if err != nil {
		return err
	}
	pass, err := envOrFail("COUCHBASE_PASSWORD")
	if err != nil {
		return err
	}
	bucket, err := envOrFail("COUCHBASE_BUCKET_NAME")
	if err != nil {
		return err
	}
	scope, err := envOrFail("COUCHBASE_SCOPE_NAME")
	if err != nil {
		return err
	}
	collName, err := envOrFail("COUCHBASE_COLLECTION_NAME")
	if err != nil {
		return err
	}

	opts := gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{Username: user, Password: pass},
	}
	if err := opts.ApplyProfile(gocb.ClusterConfigProfileWanDevelopment); err != nil {
		return err
	}
	cluster, err := gocb.Connect(conn, opts)
	if err != nil {
		return err
	}
	b := cluster.Bucket(bucket)
	if err := b.WaitUntilReady(5*time.Second, nil); err != nil {
		return err
	}
	a.cluster = cluster
	a.coll = b.Scope(scope).Collection(collName)
	a.docID = envOr("COUNTER_KEY", "counter")
	return nil
}

func (a *couchbaseAdapter) Increment(_ context.Context) (int64, error) {
	docOut, err := a.coll.Get(a.docID, &gocb.GetOptions{})
	if err != nil {
		// On first run the doc may not exist; seed at 1.
		if _, upErr := a.coll.Upsert(a.docID, uint64(1), &gocb.UpsertOptions{}); upErr != nil {
			return 0, upErr
		}
		return 1, nil
	}
	var current uint64
	if err := docOut.Content(&current); err != nil {
		return 0, err
	}
	current++
	if _, err := a.coll.Upsert(a.docID, current, &gocb.UpsertOptions{}); err != nil {
		return 0, err
	}
	return int64(current), nil
}

func (a *couchbaseAdapter) Close(_ context.Context) error {
	return a.cluster.Close(nil)
}
