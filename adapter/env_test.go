package adapter

import (
	"strings"
	"testing"
)

func TestEnvOrFailReturnsConfiguredValue(t *testing.T) {
	t.Setenv("KEEPALIVE_TEST_REQUIRED", "configured")

	got, err := envOrFail("KEEPALIVE_TEST_REQUIRED")
	if err != nil {
		t.Fatalf("envOrFail returned error: %v", err)
	}
	if got != "configured" {
		t.Fatalf("envOrFail() = %q, want configured", got)
	}
}

func TestEnvOrFailReportsMissingName(t *testing.T) {
	_, err := envOrFail("KEEPALIVE_TEST_MISSING_REQUIRED")
	if err == nil {
		t.Fatal("envOrFail returned nil error")
	}
	if !strings.Contains(err.Error(), "KEEPALIVE_TEST_MISSING_REQUIRED") {
		t.Fatalf("error %q does not include env name", err)
	}
}

func TestEnvOrReturnsConfiguredValue(t *testing.T) {
	t.Setenv("KEEPALIVE_TEST_OPTIONAL", "configured")

	got := envOr("KEEPALIVE_TEST_OPTIONAL", "default")
	if got != "configured" {
		t.Fatalf("envOr() = %q, want configured", got)
	}
}

func TestEnvOrUsesDefaultWhenNoNamesAreSet(t *testing.T) {
	got := envOr("KEEPALIVE_TEST_MISSING_DEFAULT", "default")
	if got != "default" {
		t.Fatalf("envOr() = %q, want default", got)
	}
}
