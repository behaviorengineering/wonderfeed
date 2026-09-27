package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
)

const testConfigWithSecrets = `secrets:
  - DATABASE_URL
database_url: "${DATABASE_URL}"
control_plane:
  bind: "127.0.0.1:8080"
provider:
  ytzero:
    base_url: "http://127.0.0.1:3001"
`

func TestLoadExpandsDatabaseURLFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(testConfigWithSecrets), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:5432/db?sslmode=disable")
	cfg, loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != path {
		t.Fatalf("path %q", loaded)
	}
	if cfg.DatabaseURL != "postgres://u:p@127.0.0.1:5432/db?sslmode=disable" {
		t.Fatalf("database_url %q", cfg.DatabaseURL)
	}
}

func TestLoadExpandsDatabaseURLFromKeyringAfterResolve(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(testConfigWithSecrets), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("DATABASE_URL")
	mem := operatorconfig.NewMemKeyring()
	const dsn = "postgres://keyring@127.0.0.1:5432/db?sslmode=disable"
	if err := mem.Set(AppName, "DATABASE_URL", dsn); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadWith(path, mem)
	if err != nil {
		t.Fatalf("LoadWith: %v", err)
	}
	if cfg.DatabaseURL != dsn {
		t.Fatalf("database_url %q want %q", cfg.DatabaseURL, dsn)
	}
}

func TestLoadControlPlaneConfigFallsBackToEnvWhenNoFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_ = os.Remove("config.yaml")
	t.Setenv("WONDERFEED_CONFIG", "")
	t.Setenv("DATABASE_URL", "postgres://env-only")
	got, err := LoadControlPlaneConfig("")
	if err != nil {
		t.Fatalf("LoadControlPlaneConfig: %v", err)
	}
	if got.DatabaseURL != "postgres://env-only" {
		t.Fatalf("got %q", got.DatabaseURL)
	}
}

func TestLoadMissingExplicitOverrideFailsClosed(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-config.yaml")
	_, _, err := Load(missing)
	if err == nil || !isConfigNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestLoadWithoutSecretsListStillExpandsFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	raw := "database_url: \"${DATABASE_URL}\"\ncontrol_plane:\n  bind: \"127.0.0.1:8080\"\nprovider:\n  ytzero:\n    base_url: \"http://127.0.0.1:3001\"\n"
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "postgres://env-only")
	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseURL != "postgres://env-only" {
		t.Fatalf("got %q", cfg.DatabaseURL)
	}
}
