package adapter

import "testing"

func TestWithDefaultConnectTimeout(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"postgres://u:p@db.example.com/k", "postgres://u:p@db.example.com/k?connect_timeout=30"},
		{"postgresql://u:p@db.example.com/k?sslmode=require", "postgresql://u:p@db.example.com/k?connect_timeout=30&sslmode=require"},
		{"postgres://db.example.com/k?connect_timeout=5", "postgres://db.example.com/k?connect_timeout=5"},
		// The timeout belongs in the query, never after a fragment.
		{"postgres://u:p@db.example.com/k#x", "postgres://u:p@db.example.com/k?connect_timeout=30#x"},
		{"host=db.example.com dbname=k", "host=db.example.com dbname=k connect_timeout=30"},
		{"host=db.example.com connect_timeout=5", "host=db.example.com connect_timeout=5"},
		// "://" inside a key=value value does not make it a URL.
		{"host=h user=u password=a://b dbname=k", "host=h user=u password=a://b dbname=k connect_timeout=30"},
		// Unparseable URLs are left for Connect to report.
		{"postgres://u:p%zz@db.example.com/k", "postgres://u:p%zz@db.example.com/k"},
	} {
		if got := withDefaultConnectTimeout(tc.in); got != tc.want {
			t.Errorf("withDefaultConnectTimeout(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestConfigKeysCoverEveryAdapter(t *testing.T) {
	for name := range Registry {
		if _, ok := ConfigKeys[name]; !ok {
			t.Errorf("adapter %q has no ConfigKeys entry", name)
		}
	}
}

func TestCouchbaseConnectTimeoutExceedsReadyTimeout(t *testing.T) {
	a := &couchbaseAdapter{readyTimeout: 2 * 60 * 1e9}
	if a.ConnectTimeout() <= a.readyTimeout {
		t.Fatalf("ConnectTimeout %s does not exceed ready_timeout %s", a.ConnectTimeout(), a.readyTimeout)
	}
}
