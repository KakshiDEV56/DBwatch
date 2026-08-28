// Package testkit is dbwatch's chaos/workload test harness: it generates
// realistic PostgreSQL activity and failure conditions against a
// dedicated test environment so dbwatch's detection can be verified
// against real server behavior instead of assumptions.
//
// Every operation here is gated by Guard (see safety.go) before it
// touches a database. This package must never run against anything but
// the databases explicitly listed in the test config.
package testkit

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Safety     SafetyConfig     `yaml:"safety"`
	Databases  []DatabaseTarget `yaml:"databases"`
	Workload   WorkloadConfig   `yaml:"workload"`
	Thresholds Thresholds       `yaml:"thresholds"`
}

type SafetyConfig struct {
	// RequireEnv names an environment variable that must be set to
	// "true" before any scenario will run.
	RequireEnv string `yaml:"require_env"`
	// AllowedDatabases lists the exact `current_database()` names this
	// harness is permitted to touch. Anything else is refused.
	AllowedDatabases []string `yaml:"allowed_databases"`
	// AllowedHosts lists the DSN hosts this harness is permitted to
	// target -- a second, independent guard against ever pointing this
	// at a remote/real server by accident.
	AllowedHosts []string `yaml:"allowed_hosts"`
}

type DatabaseTarget struct {
	Name string `yaml:"name"`
	DSN  string `yaml:"dsn"`
}

type WorkloadConfig struct {
	Workers int `yaml:"workers"`
	QPS     int `yaml:"qps"`
}

type Thresholds struct {
	SlowQuery          time.Duration `yaml:"slow_query"`
	LongTransaction    time.Duration `yaml:"long_transaction"`
	ConnectionWarning  float64       `yaml:"connection_warning"`
	ConnectionCritical float64       `yaml:"connection_critical"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read test config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse test config %s: %w", path, err)
	}
	return cfg, nil
}

// Database looks up a configured target by name.
func (c Config) Database(name string) (DatabaseTarget, error) {
	for _, d := range c.Databases {
		if d.Name == name {
			return d, nil
		}
	}
	return DatabaseTarget{}, fmt.Errorf("no database named %q in test config (have: %s)", name, c.databaseNames())
}

func (c Config) databaseNames() string {
	names := make([]string, len(c.Databases))
	for i, d := range c.Databases {
		names[i] = d.Name
	}
	return strings.Join(names, ", ")
}
