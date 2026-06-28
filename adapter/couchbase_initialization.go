package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

const defaultCouchbaseReadyTimeout = 30 * time.Second

func (a *couchbaseAdapter) bucketReadyError(err error) error {
	return fmt.Errorf(
		"bucket %q was not ready after %s: %w; check bucket_name, connection_string, database user permissions, and Capella allowed IP/network access; increase ready_timeout if the cluster or bucket was just created",
		a.bucket,
		a.readyTimeout,
		err,
	)
}

func (a *couchbaseAdapter) ensureBucket(ctx context.Context, cluster *gocb.Cluster) error {
	if a.bucketRAMQuotaMB == 0 {
		return nil
	}
	err := cluster.Buckets().CreateBucket(gocb.CreateBucketSettings{
		BucketSettings: gocb.BucketSettings{
			Name:       a.bucket,
			RAMQuotaMB: a.bucketRAMQuotaMB,
			BucketType: gocb.CouchbaseBucketType,
		},
	}, &gocb.CreateBucketOptions{Context: ctx, Timeout: a.readyTimeout})
	if err != nil && !errors.Is(err, gocb.ErrBucketExists) {
		return fmt.Errorf("create bucket %q: %w", a.bucket, err)
	}
	return nil
}

func (a *couchbaseAdapter) ensureScopeAndCollection(ctx context.Context, bucket *gocb.Bucket) error {
	manager := bucket.Collections()
	if a.scope != "_default" {
		err := manager.CreateScope(a.scope, &gocb.CreateScopeOptions{Context: ctx, Timeout: a.readyTimeout})
		if err != nil && !errors.Is(err, gocb.ErrScopeExists) {
			return fmt.Errorf("create scope %q: %w", a.scope, err)
		}
	}
	if a.scope == "_default" && a.collName == "_default" {
		return nil
	}
	err := manager.CreateCollection(gocb.CollectionSpec{Name: a.collName, ScopeName: a.scope}, &gocb.CreateCollectionOptions{Context: ctx, Timeout: a.readyTimeout})
	if err != nil && !errors.Is(err, gocb.ErrCollectionExists) {
		return fmt.Errorf("create collection %q.%q: %w", a.scope, a.collName, err)
	}
	return nil
}

func (a *couchbaseAdapter) ensureDocument(ctx context.Context) error {
	deadline := time.Now().Add(a.readyTimeout)
	for {
		_, err := a.coll.Insert(a.docID, uint64(0), &gocb.InsertOptions{Context: ctx, Timeout: a.readyTimeout})
		if err == nil || errors.Is(err, gocb.ErrDocumentExists) {
			return nil
		}
		if !errors.Is(err, gocb.ErrScopeNotFound) && !errors.Is(err, gocb.ErrCollectionNotFound) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("wait for collection %q.%q: %w", a.scope, a.collName, err)
		}
		if !sleepContext(ctx, time.Second) {
			return ctx.Err()
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
