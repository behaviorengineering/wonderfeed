package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/config/template"
	"github.com/behaviorengineering/wonderfeed/internal/controlplane"
	"github.com/behaviorengineering/wonderfeed/internal/secret"
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
	dir, err := UserConfigDir()
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, "config.Init", "resolve user config directory")
	}
	configPath, err := UserConfigFilePath()
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, "config.Init", "resolve user config file")
	}
	examplePath, err := UserConfigExampleFilePath()
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, "config.Init", "resolve user config example")
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, "config.Init", "create user config directory").With("path", dir)
	}

	result := &InitResult{ConfigDir: dir}

	if err := os.WriteFile(examplePath, template.ConfigExampleYAML, 0o600); err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, "config.Init", "write config example").With("path", examplePath)
	}
	result.Created = append(result.Created, examplePath)

	if _, err := os.Stat(configPath); err == nil && !opts.Force {
		result.Skipped = append(result.Skipped, configPath)
		return result, nil
	}
	if err := os.WriteFile(configPath, template.ConfigYAML, 0o600); err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, "config.Init", "write config").With("path", configPath)
	}
	result.Created = append(result.Created, configPath)
	return result, nil
}

// FileConfig is the on-disk operator YAML schema.
type FileConfig struct {
	DatabaseURL   string              `yaml:"database_url"`
	ParentAuthKey string              `yaml:"parent_auth_key"`
	ControlPlane  ControlPlaneSection `yaml:"control_plane"`
	Provider      ProviderSection     `yaml:"provider"`
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

// Load reads operator config from override, env, user path, then cwd config.yaml.
func Load(override string) (*FileConfig, string, error) {
	path, err := resolveConfigPath(override)
	if err != nil {
		return nil, "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", apperr.Wrap(err, apperr.CodeUnavailable, "config.Load", "read config").With("path", path)
	}
	cfg := &FileConfig{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, "", apperr.Wrap(err, apperr.CodeInvalid, "config.Load", "parse config").With("path", path)
	}
	applyDefaults(cfg)
	if err := expandConfigPlaceholders(cfg); err != nil {
		return nil, "", err
	}
	if err := resolveFileSecrets(cfg); err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}

// LoadControlPlaneConfig loads YAML when present, otherwise environment defaults.
func LoadControlPlaneConfig(override string) (controlplane.Config, error) {
	cfg, _, err := Load(override)
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

func resolveConfigPath(override string) (string, error) {
	if p := strings.TrimSpace(override); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", apperr.Wrap(err, apperr.CodeNotFound, "config.resolvePath", "config override missing").With("path", p)
		}
		return p, nil
	}
	if env := strings.TrimSpace(os.Getenv(EnvConfig)); env != "" {
		if _, err := os.Stat(env); err != nil {
			return "", apperr.Wrap(err, apperr.CodeNotFound, "config.resolvePath", "WONDERFEED_CONFIG path missing").With("path", env)
		}
		return env, nil
	}
	if userPath, err := UserConfigFilePath(); err == nil {
		if _, err := os.Stat(userPath); err == nil {
			return userPath, nil
		}
	}
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml", nil
	}
	return "", apperr.New(apperr.CodeNotFound, "config.resolvePath", "config.yaml not found (run: wonderfeed init)")
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

func resolveFileSecrets(cfg *FileConfig) error {
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		if v, err := secret.Resolve("DATABASE_URL"); err == nil {
			cfg.DatabaseURL = v
		} else if !secret.IsNotFound(err) {
			return apperr.Wrap(err, apperr.CodeUnavailable, "config.resolveSecrets", "resolve DATABASE_URL")
		}
	}
	if strings.TrimSpace(cfg.ParentAuthKey) == "" {
		if v, err := secret.Resolve("WONDERFEED_PARENT_AUTH_KEY"); err == nil {
			cfg.ParentAuthKey = v
		} else if !secret.IsNotFound(err) {
			return apperr.Wrap(err, apperr.CodeUnavailable, "config.resolveSecrets", "resolve WONDERFEED_PARENT_AUTH_KEY")
		}
	}
	if strings.TrimSpace(cfg.Provider.YTZero.SessionCookie) == "" {
		if v, err := secret.Resolve("YTZERO_SESSION_COOKIE"); err == nil {
			cfg.Provider.YTZero.SessionCookie = v
		} else if !secret.IsNotFound(err) {
			return apperr.Wrap(err, apperr.CodeUnavailable, "config.resolveSecrets", "resolve YTZERO_SESSION_COOKIE")
		}
	}
	if strings.TrimSpace(cfg.Provider.YTZero.AuthPassword) == "" {
		if v, err := secret.Resolve("YTZERO_AUTH_PASSWORD"); err == nil {
			cfg.Provider.YTZero.AuthPassword = v
		} else if !secret.IsNotFound(err) {
			return apperr.Wrap(err, apperr.CodeUnavailable, "config.resolveSecrets", "resolve YTZERO_AUTH_PASSWORD")
		}
	}
	return nil
}

func isConfigNotFound(err error) bool {
	var ae *apperr.Error
	if err == nil {
		return false
	}
	// apperr.Wrap preserves code in Error type
	if apperrAs(err, &ae) {
		return ae.Code == apperr.CodeNotFound
	}
	return false
}

func apperrAs(err error, target **apperr.Error) bool {
	ae, ok := err.(*apperr.Error)
	if !ok {
		return false
	}
	*target = ae
	return true
}
