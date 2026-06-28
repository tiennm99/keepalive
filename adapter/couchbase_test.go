package adapter

import (
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
