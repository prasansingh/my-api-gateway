package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
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

type DatabaseConfig struct {
	Host            string   `yaml:"host"`
	Port            int      `yaml:"port"`
	User            string   `yaml:"user"`
	Password        string   `yaml:"password"`
	DBName          string   `yaml:"dbname"`
	SSLMode         string   `yaml:"sslmode"`
	MaxOpenConns    int      `yaml:"max-open-conns"`
	MaxIdleConns    int      `yaml:"max-idle-conns"`
	ConnMaxLifetime Duration `yaml:"conn-max-lifetime"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Routes   []RouteConfig  `yaml:"routes"`
	Health   HealthConfig   `yaml:"health"`
	Database DatabaseConfig `yaml:"database"`
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

	resolveEnvPassword(&cfg)

	return &cfg, nil
}

func resolveEnvPassword(cfg *Config) {
	p := cfg.Database.Password
	if strings.HasPrefix(p, "${") && strings.HasSuffix(p, "}") {
		envVar := p[2 : len(p)-1]
		val := os.Getenv(envVar)
		if val == "" {
			slog.Warn("database password env var is empty", "var", envVar)
		}
		cfg.Database.Password = val
	}
}
