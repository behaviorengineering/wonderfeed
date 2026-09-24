package controlplane

import (
	"net"
	"os"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

const (
	// DefaultBind is the loopback-only HTTP listen address.
	DefaultBind = "127.0.0.1:8080"
	// DefaultProviderBaseURL is the local YT Zero UI/API.
	DefaultProviderBaseURL = "http://127.0.0.1:3001"
)

// Config holds control-plane runtime settings.
type Config struct {
	Bind                  string
	DatabaseURL           string
	ProviderBaseURL       string
	ProviderAuthMethod    string
	ProviderAuthPassword  string
	ProviderSessionCookie string
	ParentAuthKey         string
}

// LoadConfigFromEnv builds Config from environment variables with defaults.
func LoadConfigFromEnv() Config {
	cfg := Config{
		Bind:                  envOr("WONDERFEED_CONTROL_BIND", DefaultBind),
		DatabaseURL:           strings.TrimSpace(os.Getenv("DATABASE_URL")),
		ProviderBaseURL:       envOr("YTZERO_BASE_URL", DefaultProviderBaseURL),
		ProviderAuthMethod:    envOr("YTZERO_AUTH_METHOD", "none"),
		ProviderAuthPassword:  strings.TrimSpace(os.Getenv("YTZERO_AUTH_PASSWORD")),
		ProviderSessionCookie: strings.TrimSpace(os.Getenv("YTZERO_SESSION_COOKIE")),
		ParentAuthKey:         strings.TrimSpace(os.Getenv("WONDERFEED_PARENT_AUTH_KEY")),
	}
	return cfg
}

// Validate checks bind safety and required fields for serve.
func (c Config) Validate() error {
	const op = "controlplane.Config.Validate"
	if strings.TrimSpace(c.Bind) == "" {
		return apperr.New(apperr.CodeInvalid, op, "bind address is required")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return apperr.New(apperr.CodeInvalid, op, "DATABASE_URL is required")
	}
	if strings.TrimSpace(c.ProviderBaseURL) == "" {
		return apperr.New(apperr.CodeInvalid, op, "provider base URL is required")
	}
	loopback, err := isLoopbackBind(c.Bind)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeInvalid, op, "parse bind address").With("bind", c.Bind)
	}
	if !loopback && c.ParentAuthKey == "" {
		return apperr.New(apperr.CodeInvalid, op, "non-loopback bind requires WONDERFEED_PARENT_AUTH_KEY").
			With("bind", c.Bind)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func isLoopbackBind(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// Allow host-only forms by attempting default port split failure recovery.
		if strings.Count(addr, ":") == 0 {
			host = addr
		} else {
			return false, err
		}
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "localhost" {
		return true, nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false, nil
	}
	return ip.IsLoopback(), nil
}
