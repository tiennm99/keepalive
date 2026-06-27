package adapter

import (
	"fmt"
	"os"
)

func envOrFail(name string) (string, error) {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return "", fmt.Errorf("env %s is required", name)
	}
	return v, nil
}

func envOr(name, def string) string {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		return v
	}
	return def
}
