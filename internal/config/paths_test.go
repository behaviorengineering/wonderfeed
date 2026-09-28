package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
)

func TestResolveConfigPathPrefersFlagOverride(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("database_url: postgres://x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions(cfgPath)
	got, err := operatorconfig.ResolveConfigPath(opts)
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if got != cfgPath {
		t.Fatalf("got %q want %q", got, cfgPath)
	}
}

func TestUserConfigFilePathUsesXDG(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	got, err := UserConfigFilePath()
	if err != nil {
		t.Fatalf("UserConfigFilePath: %v", err)
	}
	want := filepath.Join(base, AppName, "config.yaml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestInitUserConfigFilesCreates0600(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	result, err := InitUserConfigFiles(InitOptions{})
	if err != nil {
		t.Fatalf("InitUserConfigFiles: %v", err)
	}
	if len(result.Created) < 2 {
		t.Fatalf("expected created files, got %#v", result)
	}
	cfgPath, err := UserConfigFilePath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode %v want 0600", info.Mode().Perm())
	}
}

func TestInitUserConfigFilesSkipsExistingLiveConfig(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	if _, err := InitUserConfigFiles(InitOptions{}); err != nil {
		t.Fatal(err)
	}
	result, err := InitUserConfigFiles(InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("expected skipped live config, got %#v", result)
	}
}
