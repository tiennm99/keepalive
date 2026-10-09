package main

import (
	"strings"
	"testing"
)

func TestHostFromEndpointNeverLeaksDSNSecrets(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"postgres://u:p@db.example.com:5432/k", "db.example.com"},
		{"host=db.example.com user=u password=SeCrEt:x dbname=k", "db.example.com"},
		{"host='db.example.com,db2.example.com' password=SeCrEt:x", "db.example.com"},
		{"user=u password=SeCrEt:x dbname=k", ""},
		{"postgres://u:SeCrEt%zz@db.example.com:5432/k", ""},
		{"u:SeCrEt@tcp(db.example.com:3306)/k", "db.example.com"},
		{"cache.example.com:6379", "cache.example.com"},
	} {
		got := hostFromEndpoint(tc.in)
		if got != tc.want || strings.Contains(strings.ToLower(got), "secret") {
			t.Errorf("hostFromEndpoint(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
