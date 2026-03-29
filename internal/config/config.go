package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (d Duration) Std() time.Duration {
	return time.Duration(d)
}

type ServerConfig struct {
	Port         int      `yaml:"port"`
	ReadTimeout  Duration `yaml:"read-timeout"`
	WriteTimeout Duration `yaml:"write-timeout"`
	Shutdown     Duration `yaml:"shutdown"`
	DrainWindow  Duration `yaml:"drain-window"`
}

type RouteConfig struct {
	Path     string   `yaml:"path"`
	Rewrite  string   `yaml:"rewrite"`
	Upstream string   `yaml:"upstream"`
	Timeout  Duration `yaml:"timeout"`
}

type HealthConfig struct {
	CheckInterval Duration `yaml:"check_interval"`
	CheckTimeout  Duration `yaml:"check_timeout"`
}

type Config struct {
	Server ServerConfig  `yaml:"server"`
	Routes []RouteConfig `yaml:"routes"`
	Health HealthConfig  `yaml:"health"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
