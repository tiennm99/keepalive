package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tiennm99/keepalive/adapter"
	"gopkg.in/yaml.v3"
)

const (
	defaultInterval   = time.Minute
	defaultCounterKey = "counter"
)

type appConfig struct {
	Interval   string              `json:"interval" yaml:"interval"`
	CounterKey string              `json:"counter_key" yaml:"counter_key"`
	Services   []serviceFileConfig `json:"services" yaml:"services"`
}

type serviceFileConfig struct {
	Name       string            `json:"name" yaml:"name"`
	Adapter    string            `json:"adapter" yaml:"adapter"`
	Interval   string            `json:"interval" yaml:"interval"`
	CounterKey string            `json:"counter_key" yaml:"counter_key"`
	Config     map[string]string `json:"config" yaml:"config"`
}

type serviceConfig struct {
	Name        string
	AdapterType string
	Interval    time.Duration
	Config      adapter.Config
}

func loadConfigFile(path string) ([]serviceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw appConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return normalizeConfig(raw)
}

func normalizeConfig(raw appConfig) ([]serviceConfig, error) {
	if len(raw.Services) == 0 {
		return nil, fmt.Errorf("services must contain at least one service")
	}

	globalInterval, err := parseConfigInterval("interval", raw.Interval, defaultInterval)
	if err != nil {
		return nil, err
	}
	globalCounterKey := valueOrDefault(raw.CounterKey, defaultCounterKey)

	services := make([]serviceConfig, 0, len(raw.Services))
	usedNames := map[string]int{}

	for i, rawService := range raw.Services {
		servicePath := fmt.Sprintf("services[%d]", i)
		adapterType := strings.TrimSpace(rawService.Adapter)
		if adapterType == "" {
			return nil, fmt.Errorf("%s.adapter is required", servicePath)
		}

		interval, err := parseConfigInterval(servicePath+".interval", rawService.Interval, globalInterval)
		if err != nil {
			return nil, err
		}

		cfg := adapter.Config{}
		for key, value := range rawService.Config {
			cfg[key] = value
		}
		cfg["counter_key"] = valueOrDefault(rawService.CounterKey, globalCounterKey)

		name, err := normalizeServiceName(rawService.Name, adapterType, cfg, usedNames, servicePath)
		if err != nil {
			return nil, err
		}

		services = append(services, serviceConfig{
			Name:        name,
			AdapterType: adapterType,
			Interval:    interval,
			Config:      cfg,
		})
	}

	return services, nil
}

func parseConfigInterval(field, value string, def time.Duration) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return def, nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		if d <= 0 {
			return 0, fmt.Errorf("%s must be greater than zero", field)
		}
		return d, nil
	}
	if n, err := strconv.Atoi(value); err == nil {
		d := time.Duration(n) * time.Second
		if d <= 0 {
			return 0, fmt.Errorf("%s must be greater than zero", field)
		}
		return d, nil
	}
	return 0, fmt.Errorf("%s must be a duration like 30s or an integer number of seconds", field)
}

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

func valueOrDefault(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}
