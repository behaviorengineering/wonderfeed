package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// AppDir is the XDG config subdirectory (~/.config/wonderfeed).
const AppDir = "wonderfeed"

// EnvConfig is the override env var for an explicit config file path.
const EnvConfig = "WONDERFEED_CONFIG"

// UserConfigDir returns ~/.config/wonderfeed or $XDG_CONFIG_HOME/wonderfeed.
func UserConfigDir() (string, error) {
	base, err := userConfigBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppDir), nil
}

// UserConfigFilePath returns the live user config.yaml path.
func UserConfigFilePath() (string, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// UserConfigExampleFilePath returns the user config.yaml.example path.
func UserConfigExampleFilePath() (string, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml.example"), nil
}

func userConfigBaseDir() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperr.Wrap(err, apperr.CodeUnavailable, "config.userConfigBaseDir", "resolve user home directory")
	}
	return filepath.Join(home, ".config"), nil
}
