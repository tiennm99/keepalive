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
		readyTimeout, err := cfg.OptionalDuration("ready_timeout", defaultCouchbaseReadyTimeout)
		if err != nil {
			return nil, err
		}
		bucketRAMQuotaMB, err := cfg.OptionalUint64("bucket_ram_quota_mb", 0)
		if err != nil {
			return nil, err
		}
		return &couchbaseAdapter{
			conn:             conn,
			user:             user,
			pass:             pass,
			bucket:           bucket,
			scope:            scope,
			collName:         collName,
			docID:            cfg.Optional("counter_key", "counter"),
			readyTimeout:     readyTimeout,
			bucketRAMQuotaMB: bucketRAMQuotaMB,
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

	readyTimeout     time.Duration
	bucketRAMQuotaMB uint64
}

func (a *couchbaseAdapter) Connect(ctx context.Context) error {
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
	connected := false
	defer func() {
		if !connected {
			cluster.Close(nil)
		}
	}()

	if err := a.ensureBucket(ctx, cluster); err != nil {
		return err
	}
	b := cluster.Bucket(a.bucket)
	if err := b.WaitUntilReady(a.readyTimeout, &gocb.WaitUntilReadyOptions{Context: ctx}); err != nil {
		return a.bucketReadyError(err)
	}
	if err := a.ensureScopeAndCollection(ctx, b); err != nil {
		return err
	}
	a.coll = b.Scope(a.scope).Collection(a.collName)
	if err := a.ensureDocument(ctx); err != nil {
		return err
	}
	// Set only on success: the deferred cleanup closes a failed cluster, and
	// Close must not close it again.
	a.cluster = cluster
	connected = true
	return nil
}

func (a *couchbaseAdapter) Increment(ctx context.Context) (int64, error) {
	// One atomic server-side increment; ensureDocument seeded the counter.
	res, err := a.coll.Binary().Increment(a.docID, &gocb.IncrementOptions{Delta: 1, Initial: 1, Context: ctx})
	if err != nil {
		return 0, err
	}
	return int64(res.Content()), nil
}

func (a *couchbaseAdapter) Close(_ context.Context) error {
	if a.cluster == nil {
		return nil
	}
	return a.cluster.Close(nil)
}
