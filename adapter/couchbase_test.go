package adapter

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCouchbaseFactoryParsesInitializationOptions(t *testing.T) {
	a, err := New("couchbase", Config{
		"connection_string":   "couchbases://cb.example.com",
		"username":            "user",
		"password":            "pass",
		"bucket_name":         "keepalive",
		"scope_name":          "scope",
		"collection_name":     "collection",
		"ready_timeout":       "45s",
		"bucket_ram_quota_mb": "128",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got := a.(*couchbaseAdapter)
	if got.readyTimeout != 45*time.Second {
		t.Fatalf("readyTimeout = %s, want 45s", got.readyTimeout)
	}
	if got.bucketRAMQuotaMB != 128 {
		t.Fatalf("bucketRAMQuotaMB = %d, want 128", got.bucketRAMQuotaMB)
	}
}

func TestCouchbaseFactoryUsesHostedReadyTimeoutDefault(t *testing.T) {
	a, err := New("couchbase", Config{
		"connection_string": "couchbases://cb.example.com",
		"username":          "user",
		"password":          "pass",
		"bucket_name":       "keepalive",
		"scope_name":        "scope",
		"collection_name":   "collection",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got := a.(*couchbaseAdapter)
	if got.readyTimeout != 2*time.Minute {
		t.Fatalf("readyTimeout = %s, want 2m", got.readyTimeout)
	}
}

func TestCouchbaseFactoryRejectsInvalidReadyTimeout(t *testing.T) {
	_, err := New("couchbase", Config{
		"connection_string": "couchbases://cb.example.com",
		"username":          "user",
		"password":          "pass",
		"bucket_name":       "keepalive",
		"scope_name":        "scope",
		"collection_name":   "collection",
		"ready_timeout":     "0s",
	})
	if err == nil {
		t.Fatal("New returned nil error")
	}
}

func TestCouchbaseBucketReadyErrorAddsActionableContext(t *testing.T) {
	cause := errors.New("unambiguous timeout")
	a := couchbaseAdapter{bucket: "keepalive", readyTimeout: 2 * time.Minute}

	err := a.bucketReadyError(cause)
	if !errors.Is(err, cause) {
		t.Fatalf("bucketReadyError does not wrap cause: %v", err)
	}
	for _, want := range []string{"keepalive", "2m0s", "Capella allowed IP", "ready_timeout"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("bucketReadyError() = %q, want it to contain %q", err, want)
		}
	}
}
