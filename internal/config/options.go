package config

import (
	"strings"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
)

// AppName is the XDG config directory and keyring service id.
const AppName = "wonderfeed"

// ConfigEnv is the environment variable for an explicit config file path.
const ConfigEnv = "WONDERFEED_CONFIG"

// DefaultOptions returns operatorconfig discovery options for Wonderfeed.
func DefaultOptions(configFlagPath string) operatorconfig.Options {
	return operatorconfig.Options{
		App:            AppName,
		ConfigEnv:      ConfigEnv,
		ConfigFlagPath: strings.TrimSpace(configFlagPath),
		Filename:       "config.yaml",
		ExtraPaths:     []string{"config.yaml"},
	}
}

// UserConfigFilePath returns the live user config.yaml path.
func UserConfigFilePath() (string, error) {
	return operatorconfig.UserConfigPath(AppName, "config.yaml")
}
