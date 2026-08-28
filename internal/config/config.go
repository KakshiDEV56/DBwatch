// Package config loads dbwatch's YAML configuration file.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Database is the legacy single-database form, still accepted for
	// backwards compatibility. Load() folds it into Databases.
	Database Database `yaml:"database"`
	// Databases is the multi-database form. dbwatch polls every entry
	// concurrently and lets the user switch between them in the sidebar.
	Databases []Database `yaml:"databases"`
	Monitor   Monitor    `yaml:"monitor"`
}

type Database struct {
	Name   string `yaml:"name"`
	Region string `yaml:"region"`
	Type   string `yaml:"type"`
	DSN    string `yaml:"dsn"`
}

type Monitor struct {
	Interval time.Duration `yaml:"interval"`
}

// Default returns a Config with sane defaults for fields a file or flag
// might not set.
func Default() Config {
	return Config{
		Monitor: Monitor{Interval: 10 * time.Second},
	}
}

// Load reads and parses a YAML config file at path. A missing file is not
// an error — the caller is expected to fill in required fields (like DSN)
// from flags or environment variables instead.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.Database.DSN != "" {
		if cfg.Database.Name == "" {
			cfg.Database.Name = "default"
		}
		if cfg.Database.Region == "" {
			cfg.Database.Region = "local"
		}
		cfg.Databases = append(cfg.Databases, cfg.Database)
	}

	for i := range cfg.Databases {
		if cfg.Databases[i].Region == "" {
			cfg.Databases[i].Region = "local"
		}
	}

	return cfg, nil
}
