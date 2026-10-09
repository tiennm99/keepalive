package main

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/tiennm99/keepalive/adapter"
)

// normalizeServiceName returns a unique service name and records it in
// usedNames. Explicit names must be unique; generated names take the first
// free numeric suffix.
func normalizeServiceName(rawName, adapterType string, cfg adapter.Config, usedNames map[string]bool, servicePath string) (string, error) {
	if strings.TrimSpace(rawName) != "" {
		name := slugify(rawName)
		if name == "" {
			return "", fmt.Errorf("%s.name must contain at least one letter or number", servicePath)
		}
		if usedNames[name] {
			return "", fmt.Errorf("%s.name %q duplicates another service name", servicePath, name)
		}
		usedNames[name] = true
		return name, nil
	}

	base := generatedServiceName(adapterType, cfg)
	name := base
	for n := 2; usedNames[name]; n++ {
		name = fmt.Sprintf("%s-%d", base, n)
	}
	usedNames[name] = true
	return name, nil
}

func generatedServiceName(adapterType string, cfg adapter.Config) string {
	adapterPart := slugify(adapterType)
	if adapterPart == "" {
		adapterPart = "service"
	}
	host := serviceHost(cfg)
	if host == "" {
		return adapterPart
	}
	return adapterPart + "-" + slugify(host)
}

func serviceHost(cfg adapter.Config) string {
	for _, key := range []string{"url", "uri", "connection_string", "dsn"} {
		if host := hostFromEndpoint(cfg[key]); host != "" {
			return host
		}
	}
	return ""
}

func hostFromEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}

	// Never fall back to splitting arbitrary text: a malformed URL or a
	// key=value DSN can carry a password, and the name is logged on every line.
	if strings.Contains(endpoint, "://") {
		if u, err := url.Parse(endpoint); err == nil {
			return u.Hostname()
		}
		return ""
	}

	if host := hostFromMySQLDSN(endpoint); host != "" {
		return host
	}

	if strings.Contains(endpoint, "=") {
		return hostFromKeyValueDSN(endpoint)
	}

	if strings.ContainsAny(endpoint, " \t@/") {
		return ""
	}
	host, _, err := net.SplitHostPort(endpoint)
	if err == nil {
		return host
	}
	return ""
}

// hostFromKeyValueDSN reads host= from a PostgreSQL key=value DSN.
func hostFromKeyValueDSN(dsn string) string {
	for _, field := range strings.Fields(dsn) {
		if value, ok := strings.CutPrefix(field, "host="); ok {
			host := strings.Trim(value, "'")
			// host may list several hosts; the first names the service.
			host, _, _ = strings.Cut(host, ",")
			return host
		}
	}
	return ""
}

func hostFromMySQLDSN(dsn string) string {
	start := strings.Index(dsn, "@tcp(")
	if start == -1 {
		return ""
	}
	start += len("@tcp(")
	end := strings.Index(dsn[start:], ")")
	if end == -1 {
		return ""
	}
	address := dsn[start : start+end]
	host, _, err := net.SplitHostPort(address)
	if err == nil && host != "" {
		return host
	}
	return address
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false

	for _, r := range value {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(b.String(), "-")
}
