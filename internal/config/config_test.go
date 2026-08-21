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
