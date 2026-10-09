package adapter

import "testing"

func TestWithDefaultConnectTimeout(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"postgres://u:p@db.example.com/k", "postgres://u:p@db.example.com/k?connect_timeout=30"},
		{"postgres://u:p@db.example.com/k?sslmode=require", "postgres://u:p@db.example.com/k?sslmode=require&connect_timeout=30"},
		{"postgres://db.example.com/k?connect_timeout=5", "postgres://db.example.com/k?connect_timeout=5"},
		{"host=db.example.com dbname=k", "host=db.example.com dbname=k connect_timeout=30"},
	} {
		if got := withDefaultConnectTimeout(tc.in); got != tc.want {
			t.Fatalf("withDefaultConnectTimeout(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
