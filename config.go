package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"slices"
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

var defaultConfigFiles = []string{
	"config.yml",
	"config.yaml",
	"/config.yml",
	"/config.yaml",
}

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
	for _, warning := range schemaWarnings(data, raw) {
		log.Printf("warning: %s: %s; ignoring it", path, warning)
	}
	return normalizeConfig(raw)
}

// schemaWarnings lists every key the config schema does not define: keys
// appConfig and serviceFileConfig lack (such as a misspelled interval), and
// keys under a service's config map that its adapter does not read (such as
// a misspelled namespace). Unknown keys are ignored, not fatal.
func schemaWarnings(data []byte, raw appConfig) []string {
	var warnings []string

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var strict appConfig
	if err := decoder.Decode(&strict); err != nil && !errors.Is(err, io.EOF) {
		var typeErr *yaml.TypeError
		if errors.As(err, &typeErr) {
			warnings = append(warnings, typeErr.Errors...)
		} else {
			warnings = append(warnings, err.Error())
		}
	}

	for i, service := range raw.Services {
		adapterType := strings.TrimSpace(service.Adapter)
		known, ok := adapter.ConfigKeys[adapterType]
		if !ok {
			// normalizeConfig rejects the unknown adapter itself.
			continue
		}
		for _, key := range slices.Sorted(maps.Keys(service.Config)) {
			if key == "counter_key" || slices.Contains(known, key) {
				// normalizeConfig rejects config.counter_key with its own error.
				continue
			}
			warnings = append(warnings, fmt.Sprintf("services[%d].config: unknown key %q for adapter %s (known: %s)",
				i, key, adapterType, strings.Join(known, ", ")))
		}
	}
	return warnings
}

func defaultConfigFile() (string, error) {
	return firstExistingConfigFile(defaultConfigFiles)
}

func firstExistingConfigFile(paths []string) (string, error) {
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				// Docker creates a missing bind-mount source as a directory.
				return "", fmt.Errorf("%s is a directory, not a config file; create the config file before starting the container", path)
			}
			return path, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("config file not found (looked for: %s)", strings.Join(paths, ", "))
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
	usedNames := map[string]bool{}

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

		if _, ok := rawService.Config["counter_key"]; ok {
			return nil, fmt.Errorf("%s.config.counter_key is not supported; set counter_key on the service or at the root", servicePath)
		}
		cfg := adapter.Config{}
		for key, value := range rawService.Config {
			cfg[key] = value
		}
		cfg["counter_key"] = valueOrDefault(rawService.CounterKey, globalCounterKey)

		// Factories do no I/O, so building one here catches an unknown adapter
		// or a missing config key at startup instead of silently idling.
		if _, err := adapter.New(adapterType, cfg); err != nil {
			return nil, fmt.Errorf("%s: %w", servicePath, err)
		}

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
	return 0, fmt.Errorf("%s must be a duration like 30s, 5m, or 1h30m, or an integer number of seconds", field)
}

func valueOrDefault(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}
