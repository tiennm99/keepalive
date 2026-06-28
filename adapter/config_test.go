package adapter

import (
	"strings"
	"testing"
	"time"
)

func TestConfigRequiredReturnsConfiguredValue(t *testing.T) {
	cfg := Config{"url": "redis://127.0.0.1:6379"}

	got, err := cfg.Required("url")
	if err != nil {
		t.Fatalf("Required returned error: %v", err)
	}
	if got != "redis://127.0.0.1:6379" {
		t.Fatalf("Required() = %q, want configured value", got)
	}
}

func TestConfigRequiredReportsMissingName(t *testing.T) {
	cfg := Config{}

	_, err := cfg.Required("url")
	if err == nil {
		t.Fatal("Required returned nil error")
	}
	if !strings.Contains(err.Error(), "url") {
		t.Fatalf("error %q does not include config key", err)
	}
}

func TestConfigOptionalReturnsConfiguredValue(t *testing.T) {
	cfg := Config{"counter_key": "custom"}

	got := cfg.Optional("counter_key", "counter")
	if got != "custom" {
		t.Fatalf("Optional() = %q, want custom", got)
	}
}

func TestConfigOptionalUsesDefault(t *testing.T) {
	cfg := Config{}

	got := cfg.Optional("counter_key", "counter")
	if got != "counter" {
		t.Fatalf("Optional() = %q, want counter", got)
	}
}

func TestConfigOptionalDurationParsesDuration(t *testing.T) {
	cfg := Config{"ready_timeout": "45s"}

	got, err := cfg.OptionalDuration("ready_timeout", time.Second)
	if err != nil {
		t.Fatalf("OptionalDuration returned error: %v", err)
	}
	if got != 45*time.Second {
		t.Fatalf("OptionalDuration() = %s, want 45s", got)
	}
}

func TestConfigOptionalDurationParsesSeconds(t *testing.T) {
	cfg := Config{"ready_timeout": "30"}

	got, err := cfg.OptionalDuration("ready_timeout", time.Second)
	if err != nil {
		t.Fatalf("OptionalDuration returned error: %v", err)
	}
	if got != 30*time.Second {
		t.Fatalf("OptionalDuration() = %s, want 30s", got)
	}
}

func TestConfigOptionalDurationRejectsNonPositive(t *testing.T) {
	cfg := Config{"ready_timeout": "0s"}

	if _, err := cfg.OptionalDuration("ready_timeout", time.Second); err == nil {
		t.Fatal("OptionalDuration returned nil error")
	}
}

func TestConfigOptionalUint64ParsesValue(t *testing.T) {
	cfg := Config{"bucket_ram_quota_mb": "128"}

	got, err := cfg.OptionalUint64("bucket_ram_quota_mb", 0)
	if err != nil {
		t.Fatalf("OptionalUint64 returned error: %v", err)
	}
	if got != 128 {
		t.Fatalf("OptionalUint64() = %d, want 128", got)
	}
}

func TestConfigOptionalUint64RejectsInvalidValue(t *testing.T) {
	cfg := Config{"bucket_ram_quota_mb": "-1"}

	if _, err := cfg.OptionalUint64("bucket_ram_quota_mb", 0); err == nil {
		t.Fatal("OptionalUint64 returned nil error")
	}
}
