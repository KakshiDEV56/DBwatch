package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	if cfg.Monitor.Interval != 10*time.Second {
		t.Fatalf("default interval = %v, want 10s", cfg.Monitor.Interval)
	}
	if len(cfg.Databases) != 0 {
		t.Fatalf("default databases = %d, want 0", len(cfg.Databases))
	}
}

func TestLoadMultiDatabase(t *testing.T) {
	path := writeConfig(t, `
databases:
  - name: production
    region: us-east
    dsn: postgres://db/prod
  - name: staging
    dsn: postgres://db/stg
monitor:
  interval: 5s
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Monitor.Interval != 5*time.Second {
		t.Fatalf("interval = %v, want 5s", cfg.Monitor.Interval)
	}
	if len(cfg.Databases) != 2 {
		t.Fatalf("databases = %d, want 2", len(cfg.Databases))
	}
	if cfg.Databases[0].Region != "us-east" {
		t.Fatalf("production region = %q, want us-east", cfg.Databases[0].Region)
	}
	if cfg.Databases[1].Region != "local" {
		t.Fatalf("empty region should default to local, got %q", cfg.Databases[1].Region)
	}
}

func TestLoadLegacySingleDatabase(t *testing.T) {
	path := writeConfig(t, `
database:
  dsn: postgres://db/legacy
monitor:
  interval: 2s
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Databases) != 1 {
		t.Fatalf("legacy form should fold into Databases, got %d", len(cfg.Databases))
	}
	if cfg.Databases[0].Name != "default" {
		t.Fatalf("legacy name = %q, want default", cfg.Databases[0].Name)
	}
	if cfg.Databases[0].Region != "local" {
		t.Fatalf("legacy region = %q, want local", cfg.Databases[0].Region)
	}
	if cfg.Databases[0].DSN != "postgres://db/legacy" {
		t.Fatalf("legacy dsn = %q", cfg.Databases[0].DSN)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := writeConfig(t, "databases: [unterminated")
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error for invalid YAML")
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dbwatch.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
