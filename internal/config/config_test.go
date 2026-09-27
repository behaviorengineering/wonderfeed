package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExpandsDatabaseURLFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	raw := "database_url: \"${DATABASE_URL}\"\ncontrol_plane:\n  bind: \"127.0.0.1:8080\"\nprovider:\n  ytzero:\n    base_url: \"http://127.0.0.1:3001\"\n"
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
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

func TestLoadControlPlaneConfigFallsBackToEnvWhenNoFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	os.Remove("config.yaml")
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
