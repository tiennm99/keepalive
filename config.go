package main

import (
	"errors"
	"fmt"
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
	return normalizeConfig(raw)
}

func defaultConfigFile() (string, error) {
	return firstExistingConfigFile(defaultConfigFiles)
}

func firstExistingConfigFile(paths []string) (string, error) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
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

func valueOrDefault(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}
