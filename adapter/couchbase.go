package adapter

import (
	"context"
	"time"

	"github.com/couchbase/gocb/v2"
)

func init() {
	Registry["couchbase"] = func(cfg Config) (Adapter, error) {
		conn, err := cfg.Required("connection_string")
		if err != nil {
			return nil, err
		}
		user, err := cfg.Required("username")
		if err != nil {
			return nil, err
		}
		pass, err := cfg.Required("password")
		if err != nil {
			return nil, err
		}
		bucket, err := cfg.Required("bucket_name")
		if err != nil {
			return nil, err
		}
		scope, err := cfg.Required("scope_name")
		if err != nil {
			return nil, err
		}
		collName, err := cfg.Required("collection_name")
		if err != nil {
			return nil, err
		}
		return &couchbaseAdapter{
			conn:     conn,
			user:     user,
			pass:     pass,
			bucket:   bucket,
			scope:    scope,
			collName: collName,
			docID:    cfg.Optional("counter_key", "counter"),
		}, nil
	}
}

type couchbaseAdapter struct {
	cluster  *gocb.Cluster
	coll     *gocb.Collection
	conn     string
	user     string
	pass     string
	bucket   string
	scope    string
	collName string
	docID    string
}

func (a *couchbaseAdapter) Connect(_ context.Context) error {
	opts := gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{Username: a.user, Password: a.pass},
	}
	if err := opts.ApplyProfile(gocb.ClusterConfigProfileWanDevelopment); err != nil {
		return err
	}
	cluster, err := gocb.Connect(a.conn, opts)
	if err != nil {
		return err
	}
	b := cluster.Bucket(a.bucket)
	if err := b.WaitUntilReady(5*time.Second, nil); err != nil {
		return err
	}
	a.cluster = cluster
	a.coll = b.Scope(a.scope).Collection(a.collName)
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
