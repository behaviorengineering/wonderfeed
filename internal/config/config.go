package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/behaviorengineering/operatorconfig/pkg/operatorconfig"
	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/config/template"
	"github.com/behaviorengineering/wonderfeed/internal/controlplane"
)

// InitOptions controls wonderfeed init behavior.
type InitOptions struct {
	Force bool
}

// InitResult reports which files were created or skipped.
type InitResult struct {
	ConfigDir string
	Created   []string
	Skipped   []string
}

// InitUserConfigFiles writes config.yaml and config.yaml.example under the user config directory.
func InitUserConfigFiles(opts InitOptions) (*InitResult, error) {
	op := DefaultOptions("")
	written, err := operatorconfig.InitUserConfig(op, template.ConfigYAML, opts.Force)
	if err != nil {
		return nil, wrapConfigErr(err, "config.Init", "initialize user config")
	}
	cfgPath, err := UserConfigFilePath()
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, "config.Init", "resolve user config file")
	}
	result := &InitResult{ConfigDir: filepath.Dir(cfgPath)}
	examplePath := cfgPath + ".example"
	result.Created = append(result.Created, examplePath)
	if written {
		result.Created = append(result.Created, cfgPath)
	} else {
		result.Skipped = append(result.Skipped, cfgPath)
	}
	return result, nil
}

// FileConfig is the on-disk operator YAML schema.
type FileConfig struct {
	Secrets       []operatorconfig.Secret `yaml:"secrets"`
	DatabaseURL   string                  `yaml:"database_url"`
	ParentAuthKey string                  `yaml:"parent_auth_key"`
	ControlPlane  ControlPlaneSection     `yaml:"control_plane"`
	Provider      ProviderSection           `yaml:"provider"`
}

// ControlPlaneSection holds HTTP API settings.
type ControlPlaneSection struct {
	Bind string `yaml:"bind"`
}

// ProviderSection holds default media provider settings.
type ProviderSection struct {
	YTZero YTZeroSection `yaml:"ytzero"`
}

// YTZeroSection is the first bundled provider adapter configuration.
type YTZeroSection struct {
	BaseURL       string `yaml:"base_url"`
	AuthMethod    string `yaml:"auth_method"`
	AuthPassword  string `yaml:"auth_password"`
	SessionCookie string `yaml:"session_cookie"`
}

// Load reads operator config using DefaultOptions and the default keyring.
func Load(configFlagPath string) (*FileConfig, string, error) {
	return LoadWith(configFlagPath, nil)
}

// LoadWith reads operator config with an injectable keyring (tests).
func LoadWith(configFlagPath string, kr operatorconfig.Keyring) (*FileConfig, string, error) {
	opts := DefaultOptions(configFlagPath)
	if kr != nil {
		opts.Keyring = kr
	}
	path, err := operatorconfig.ResolveConfigPath(opts)
	if err != nil {
		return nil, "", wrapConfigErr(err, "config.Load", "resolve config path")
	}
	if path == "" {
		return nil, "", apperr.New(apperr.CodeNotFound, "config.Load", "config.yaml not found (run: wonderfeed init)")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", apperr.Wrap(err, apperr.CodeUnavailable, "config.Load", "read config").With("path", path)
	}
	cfg := &FileConfig{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, "", apperr.Wrap(err, apperr.CodeInvalid, "config.Load", "parse config").With("path", path)
	}
	resolveOpts := opts
	resolveOpts.Secrets = cfg.Secrets
	if len(resolveOpts.Secrets) > 0 {
		if err := operatorconfig.ResolveSecrets(resolveOpts, nil); err != nil {
			return nil, "", wrapConfigErr(err, "config.Load", "resolve secrets")
		}
	}
	expandFileConfigPlaceholders(cfg)
	applyDefaults(cfg)
	return cfg, path, nil
}

// LoadControlPlaneConfig loads YAML when present, otherwise environment defaults.
func LoadControlPlaneConfig(configFlagPath string) (controlplane.Config, error) {
	cfg, _, err := Load(configFlagPath)
	if err != nil {
		if isConfigNotFound(err) {
			return controlplane.LoadConfigFromEnv(), nil
		}
		return controlplane.Config{}, err
	}
	return cfg.ToControlPlaneConfig(), nil
}

func (f *FileConfig) ToControlPlaneConfig() controlplane.Config {
	bind := strings.TrimSpace(f.ControlPlane.Bind)
	if bind == "" {
		bind = controlplane.DefaultBind
	}
	baseURL := strings.TrimSpace(f.Provider.YTZero.BaseURL)
	if baseURL == "" {
		baseURL = controlplane.DefaultProviderBaseURL
	}
	authMethod := strings.TrimSpace(f.Provider.YTZero.AuthMethod)
	if authMethod == "" {
		authMethod = "none"
	}
	return controlplane.Config{
		Bind:                  bind,
		DatabaseURL:           strings.TrimSpace(f.DatabaseURL),
		ProviderBaseURL:       baseURL,
		ProviderAuthMethod:    authMethod,
		ProviderAuthPassword:  strings.TrimSpace(f.Provider.YTZero.AuthPassword),
		ProviderSessionCookie: strings.TrimSpace(f.Provider.YTZero.SessionCookie),
		ParentAuthKey:         strings.TrimSpace(f.ParentAuthKey),
	}
}

func applyDefaults(cfg *FileConfig) {
	if strings.TrimSpace(cfg.ControlPlane.Bind) == "" {
		cfg.ControlPlane.Bind = controlplane.DefaultBind
	}
	if strings.TrimSpace(cfg.Provider.YTZero.BaseURL) == "" {
		cfg.Provider.YTZero.BaseURL = controlplane.DefaultProviderBaseURL
	}
	if strings.TrimSpace(cfg.Provider.YTZero.AuthMethod) == "" {
		cfg.Provider.YTZero.AuthMethod = "none"
	}
}

func wrapConfigErr(err error, op, message string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "not found") {
		return apperr.Wrap(err, apperr.CodeNotFound, op, message)
	}
	return apperr.Wrap(err, apperr.CodeInvalid, op, message)
}

func isConfigNotFound(err error) bool {
	ae, ok := err.(*apperr.Error)
	return ok && ae.Code == apperr.CodeNotFound
}
