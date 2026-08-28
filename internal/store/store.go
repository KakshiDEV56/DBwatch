// Package store persists the list of databases dbwatch is watching, so
// the tool works with zero config files: add a database once (from the
// welcome screen or the "a" key), and it's there on every future launch
// with no flags, no YAML to hand-write.
package store

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"dbwatch/internal/config"
)

// Dir resolves the OS-appropriate per-user config directory for dbwatch
// (via os.UserConfigDir(): ~/.config/dbwatch on Linux, ~/Library/Application
// Support/dbwatch on macOS, %AppData%\dbwatch on Windows).
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "dbwatch"), nil
}

// Path is the full path to the persisted database list.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "databases.yaml"), nil
}

type file struct {
	Databases []config.Database `yaml:"databases"`
}

// Load reads the persisted database list. A store that doesn't exist yet
// (first run) is not an error -- it just means an empty list.
func Load() ([]config.Database, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return f.Databases, nil
}

// Save writes the full database list, replacing whatever was there.
// Written atomically (temp file + rename) so a crash or concurrent write
// never leaves a truncated/corrupt store on disk.
func Save(dbs []config.Database) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	path, err := Path()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(file{Databases: dbs})
	if err != nil {
		return fmt.Errorf("encode database list: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
