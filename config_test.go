package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeConfigGeneratesNamesFromAdapterAndHost(t *testing.T) {
	services, err := normalizeConfig(appConfig{
		Services: []serviceFileConfig{
			{Adapter: "redis", Config: map[string]string{"url": "redis://default@cache.example.com:6379"}},
			{Adapter: "mongodb", Config: map[string]string{"uri": "mongodb+srv://user:pass@cluster.mongodb.net", "database": "keepalive", "collection": "counter"}},
			{Adapter: "couchbase", Config: map[string]string{"connection_string": "couchbases://cb.example.net", "username": "u", "password": "p", "bucket_name": "b", "scope_name": "_default", "collection_name": "_default"}},
		},
	})
	if err != nil {
		t.Fatalf("normalizeConfig returned error: %v", err)
	}

	want := []string{"redis-cache-example-com", "mongodb-cluster-mongodb-net", "couchbase-cb-example-net"}
	for i, service := range services {
		if service.Name != want[i] {
			t.Fatalf("services[%d].Name = %q, want %q", i, service.Name, want[i])
		}
	}
}

func TestConfigExampleYMLParses(t *testing.T) {
	services, err := loadConfigFile("config.example.yml")
	if err != nil {
		t.Fatalf("loadConfigFile returned error: %v", err)
	}
	wantAdapters := []string{"redis", "valkey", "postgresql", "mysql", "mongodb", "couchbase"}
	if len(services) != len(wantAdapters) {
		t.Fatalf("len(services) = %d, want %d", len(services), len(wantAdapters))
	}
	for i, want := range wantAdapters {
		if services[i].AdapterType != want {
			t.Fatalf("services[%d].AdapterType = %q, want %q", i, services[i].AdapterType, want)
		}
	}
	if services[0].Interval != time.Minute {
		t.Fatalf("services[0].Interval = %s, want 1m", services[0].Interval)
	}
	if services[1].Interval != 30*time.Second {
		t.Fatalf("services[1].Interval = %s, want 30s", services[1].Interval)
	}
	if services[2].Interval != time.Minute {
		t.Fatalf("services[2].Interval = %s, want 1m", services[2].Interval)
	}
	if services[0].Config["counter_key"] != "counter" {
		t.Fatalf("services[0] counter_key = %q, want counter", services[0].Config["counter_key"])
	}
	if services[1].Config["counter_key"] != "valkey-counter" {
		t.Fatalf("services[1] counter_key = %q, want valkey-counter", services[1].Config["counter_key"])
	}
	if services[2].Config["counter_key"] != "counter" {
		t.Fatalf("services[2] counter_key = %q, want counter", services[2].Config["counter_key"])
	}
}

func TestFirstExistingConfigFileFindsYMLFallback(t *testing.T) {
	dir := t.TempDir()
	ymlPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(ymlPath, []byte("services: []\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := firstExistingConfigFile([]string{
		filepath.Join(dir, "config.yaml"),
		ymlPath,
	})
	if err != nil {
		t.Fatalf("firstExistingConfigFile returned error: %v", err)
	}
	if got != ymlPath {
		t.Fatalf("firstExistingConfigFile() = %q, want %q", got, ymlPath)
	}
}

func TestFirstExistingConfigFileReportsCandidates(t *testing.T) {
	dir := t.TempDir()
	_, err := firstExistingConfigFile([]string{
		filepath.Join(dir, "config.yaml"),
		filepath.Join(dir, "config.yml"),
	})
	if err == nil {
		t.Fatal("firstExistingConfigFile returned nil error")
	}
	if !strings.Contains(err.Error(), "config.yaml") || !strings.Contains(err.Error(), "config.yml") {
		t.Fatalf("error %q does not list expected candidates", err)
	}
}

func TestNormalizeConfigSuffixesDuplicateGeneratedNames(t *testing.T) {
	services, err := normalizeConfig(appConfig{
		Services: []serviceFileConfig{
			{Adapter: "redis", Config: map[string]string{"url": "redis://cache.example.com:6379"}},
			{Adapter: "redis", Config: map[string]string{"url": "redis://cache.example.com:6379"}},
		},
	})
	if err != nil {
		t.Fatalf("normalizeConfig returned error: %v", err)
	}

	if services[0].Name != "redis-cache-example-com" {
		t.Fatalf("services[0].Name = %q", services[0].Name)
	}
	if services[1].Name != "redis-cache-example-com-2" {
		t.Fatalf("services[1].Name = %q", services[1].Name)
	}
}

func TestNormalizeConfigRejectsDuplicateExplicitNames(t *testing.T) {
	_, err := normalizeConfig(appConfig{
		Services: []serviceFileConfig{
			{Name: "cache", Adapter: "redis", Config: map[string]string{"url": "redis://one.example.com:6379"}},
			{Name: "cache", Adapter: "redis", Config: map[string]string{"url": "redis://two.example.com:6379"}},
		},
	})
	if err == nil {
		t.Fatal("normalizeConfig returned nil error")
	}
}

func TestNormalizeConfigRejectsExplicitNameThatDuplicatesGeneratedName(t *testing.T) {
	_, err := normalizeConfig(appConfig{
		Services: []serviceFileConfig{
			{Adapter: "redis", Config: map[string]string{"url": "redis://cache.example.com:6379"}},
			{Name: "redis-cache-example-com", Adapter: "redis", Config: map[string]string{"url": "redis://other.example.com:6379"}},
		},
	})
	if err == nil {
		t.Fatal("normalizeConfig returned nil error")
	}
}

func TestNormalizeConfigAppliesGlobalAndServiceDefaults(t *testing.T) {
	services, err := normalizeConfig(appConfig{
		Interval:   "2m",
		CounterKey: "global-counter",
		Services: []serviceFileConfig{
			{Adapter: "redis", Config: map[string]string{"url": "redis://cache.example.com:6379"}},
			{Adapter: "redis", Interval: "30s", CounterKey: "local-counter", Config: map[string]string{"url": "redis://other.example.com:6379"}},
		},
	})
	if err != nil {
		t.Fatalf("normalizeConfig returned error: %v", err)
	}

	if services[0].Interval != 2*time.Minute {
		t.Fatalf("services[0].Interval = %s, want 2m", services[0].Interval)
	}
	if services[0].Config["counter_key"] != "global-counter" {
		t.Fatalf("services[0] counter_key = %q", services[0].Config["counter_key"])
	}
	if services[1].Interval != 30*time.Second {
		t.Fatalf("services[1].Interval = %s, want 30s", services[1].Interval)
	}
	if services[1].Config["counter_key"] != "local-counter" {
		t.Fatalf("services[1] counter_key = %q", services[1].Config["counter_key"])
	}
}

func TestNormalizeConfigParsesCompoundAndFractionalIntervals(t *testing.T) {
	services, err := normalizeConfig(appConfig{
		Interval: "1h30m",
		Services: []serviceFileConfig{
			{Adapter: "redis", Config: map[string]string{"url": "redis://cache.example.com:6379"}},
			{Adapter: "redis", Interval: "1.5h", Config: map[string]string{"url": "redis://other.example.com:6379"}},
		},
	})
	if err != nil {
		t.Fatalf("normalizeConfig returned error: %v", err)
	}

	if services[0].Interval != 90*time.Minute {
		t.Fatalf("services[0].Interval = %s, want 1h30m", services[0].Interval)
	}
	if services[1].Interval != 90*time.Minute {
		t.Fatalf("services[1].Interval = %s, want 1.5h", services[1].Interval)
	}
}

func TestNormalizeConfigRejectsNonPositiveInterval(t *testing.T) {
	_, err := normalizeConfig(appConfig{
		Interval: "0s",
		Services: []serviceFileConfig{
			{Adapter: "redis", Config: map[string]string{"url": "redis://cache.example.com:6379"}},
		},
	})
	if err == nil {
		t.Fatal("normalizeConfig returned nil error")
	}
}
