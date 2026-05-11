package adapter

import "os"

func lookupEnv(name string) (string, bool) {
	return os.LookupEnv(name)
}

func envOr(name, def string) string {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		return v
	}
	return def
}
