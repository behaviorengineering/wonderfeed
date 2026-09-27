package config

import "os"

func expandFileConfigPlaceholders(cfg *FileConfig) {
	if cfg == nil {
		return
	}
	expand := func(s string) string {
		return os.Expand(s, os.Getenv)
	}
	cfg.DatabaseURL = expand(cfg.DatabaseURL)
	cfg.ParentAuthKey = expand(cfg.ParentAuthKey)
	cfg.ControlPlane.Bind = expand(cfg.ControlPlane.Bind)
	cfg.Provider.YTZero.BaseURL = expand(cfg.Provider.YTZero.BaseURL)
	cfg.Provider.YTZero.AuthMethod = expand(cfg.Provider.YTZero.AuthMethod)
	cfg.Provider.YTZero.AuthPassword = expand(cfg.Provider.YTZero.AuthPassword)
	cfg.Provider.YTZero.SessionCookie = expand(cfg.Provider.YTZero.SessionCookie)
}
