package config

import (
	"strings"
	"testing"
)

func TestLoadValid(t *testing.T) {
	t.Setenv(EnvURL, "http://localhost:3000")
	t.Setenv(EnvToken, "sk-test-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.URL != "http://localhost:3000" {
		t.Errorf("URL = %q, want %q", cfg.URL, "http://localhost:3000")
	}
	if cfg.Token != "sk-test-key" {
		t.Errorf("Token = %q, want %q", cfg.Token, "sk-test-key")
	}
}

func TestLoadValidHTTPS(t *testing.T) {
	t.Setenv(EnvURL, "https://open-webui.example.com")
	t.Setenv(EnvToken, "sk-test-key")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() with https URL: %v", err)
	}
}

func TestLoadMissingURL(t *testing.T) {
	// EnvURL deliberately absent.
	t.Setenv(EnvToken, "sk-test-key")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error for missing OWUI_URL, got nil")
	}
	if !strings.Contains(err.Error(), EnvURL) {
		t.Errorf("error %q should mention %s", err, EnvURL)
	}
}

func TestLoadMissingToken(t *testing.T) {
	t.Setenv(EnvURL, "http://localhost:3000")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error for missing OWUI_TOKEN, got nil")
	}
	if !strings.Contains(err.Error(), EnvToken) {
		t.Errorf("error %q should mention %s", err, EnvToken)
	}
}

func TestLoadUnparsableURL(t *testing.T) {
	t.Setenv(EnvURL, "://not-a-url")
	t.Setenv(EnvToken, "sk-test-key")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for an unparsable URL, got nil")
	}
}

func TestLoadNonHTTPSScheme(t *testing.T) {
	t.Setenv(EnvURL, "ftp://localhost:3000")
	t.Setenv(EnvToken, "sk-test-key")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error for a non-http scheme, got nil")
	}
	if !strings.Contains(err.Error(), "http") {
		t.Errorf("error %q should mention http(s) guidance", err)
	}
}

func TestLoadMissingHost(t *testing.T) {
	t.Setenv(EnvURL, "http://")
	t.Setenv(EnvToken, "sk-test-key")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for a URL with no host, got nil")
	}
}

func TestLoadEmptyHostname(t *testing.T) {
	// http://:3000 has a non-empty Host (":3000") but an empty Hostname.
	t.Setenv(EnvURL, "http://:3000")
	t.Setenv(EnvToken, "sk-test-key")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for a URL with no hostname, got nil")
	}
}

func TestLoadURLUserInfo(t *testing.T) {
	// D6: credentials embedded in the URL must be rejected.
	t.Setenv(EnvURL, "http://user:pass@example.com")
	t.Setenv(EnvToken, "sk-test-key")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error for URL userinfo, got nil")
	}
	if !strings.Contains(err.Error(), EnvToken) {
		t.Errorf("error %q should point the user at %s", err, EnvToken)
	}
}

func TestLoadWhitespaceOnlyToken(t *testing.T) {
	t.Setenv(EnvURL, "http://localhost:3000")
	t.Setenv(EnvToken, "   \t\n")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error for a whitespace-only token, got nil")
	}
	if !strings.Contains(err.Error(), EnvToken) {
		t.Errorf("error %q should mention %s", err, EnvToken)
	}
}

func TestLoadUppercaseSchemeNormalized(t *testing.T) {
	// net/url lowercases the scheme, so an uppercase HTTP:// must be accepted
	// and normalized to a lowercase http:// URL.
	t.Setenv(EnvURL, "HTTP://localhost:3000")
	t.Setenv(EnvToken, "sk-test-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() rejected a valid uppercase-scheme URL: %v", err)
	}
	if cfg.URL != "http://localhost:3000" {
		t.Errorf("URL = %q, want normalized %q", cfg.URL, "http://localhost:3000")
	}
}
