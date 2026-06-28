package adapter

import (
	"strings"
	"testing"
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
