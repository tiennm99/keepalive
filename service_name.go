package main

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/tiennm99/keepalive/adapter"
)

func normalizeServiceName(rawName, adapterType string, cfg adapter.Config, usedNames map[string]int, servicePath string) (string, error) {
	if strings.TrimSpace(rawName) != "" {
		name := slugify(rawName)
		if name == "" {
			return "", fmt.Errorf("%s.name must contain at least one letter or number", servicePath)
		}
		if usedNames[name] > 0 {
			return "", fmt.Errorf("%s.name %q duplicates another service name", servicePath, name)
		}
		usedNames[name]++
		return name, nil
	}

	base := generatedServiceName(adapterType, cfg)
	count := usedNames[base] + 1
	usedNames[base] = count
	if count == 1 {
		return base, nil
	}
	return fmt.Sprintf("%s-%d", base, count), nil
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

	if strings.Contains(endpoint, "://") {
		if u, err := url.Parse(endpoint); err == nil {
			if host := u.Hostname(); host != "" {
				return host
			}
		}
	}

	if host := hostFromMySQLDSN(endpoint); host != "" {
		return host
	}

	host, _, err := net.SplitHostPort(endpoint)
	if err == nil && host != "" {
		return host
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
