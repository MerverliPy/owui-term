// Package config loads owui-term runtime configuration from the environment
// (locked constraint D6: env vars for CI/overrides; tokens never committed).
package config

import (
	"fmt"
	"net/url"
	"os"
)

// Environment variables owui-term reads. OWUI_URL must be the base URL of the
// Open-WebUI instance; OWUI_TOKEN is a Bearer token (API key or JWT).
const (
	EnvURL   = "OWUI_URL"
	EnvToken = "OWUI_TOKEN"
)

// Config holds the validated runtime configuration.
type Config struct {
	// URL is the base URL of the Open-WebUI instance, e.g. http://localhost:3000.
	URL string
	// Token is the Bearer token used to authenticate. It is used, never stored.
	Token string
}

// Load reads configuration from the environment and validates it. On failure it
// returns a descriptive, actionable error explaining exactly which variable is
// missing or invalid and how to fix it.
func Load() (Config, error) {
	cfg := Config{
		URL:   os.Getenv(EnvURL),
		Token: os.Getenv(EnvToken),
	}

	if cfg.URL == "" {
		return cfg, fmt.Errorf(
			"%s is not set — point it at your Open-WebUI instance, e.g. "+
				`export %s=http://localhost:3000`, EnvURL, EnvURL)
	}

	u, err := url.Parse(cfg.URL)
	if err != nil {
		return cfg, fmt.Errorf("%s=%q is not a valid URL: %w", EnvURL, cfg.URL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return cfg, fmt.Errorf(
			"%s=%q must be an http:// or https:// URL (got scheme %q)",
			EnvURL, cfg.URL, u.Scheme)
	}
	if u.Host == "" {
		return cfg, fmt.Errorf("%s=%q is missing a host, e.g. http://localhost:3000", EnvURL, cfg.URL)
	}

	if cfg.Token == "" {
		return cfg, fmt.Errorf(
			"%s is not set — use an Open-WebUI API key (sk-…) or a sign-in JWT, e.g. "+
				`export %s=sk-…`, EnvToken, EnvToken)
	}

	return cfg, nil
}
