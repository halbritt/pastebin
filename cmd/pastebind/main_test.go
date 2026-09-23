package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pastebin/internal/paste"
)

func TestParseConfigRejectsInvalidEnvironment(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{name: "max bytes", key: "PASTEBIN_MAX_BYTES", val: "not-bytes"},
		{name: "default ttl", key: "PASTEBIN_DEFAULT_TTL", val: "not-duration"},
		{name: "max ttl", key: "PASTEBIN_MAX_TTL", val: "not-duration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.val)
			_, err := parseConfig(nil)
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("parseConfig() error = %v, want error mentioning %s", err, tt.key)
			}
		})
	}
}

func TestParseConfigUsesEnvironmentAndFlags(t *testing.T) {
	t.Setenv("PASTEBIN_BASE_URL", "https://paste.example.ts.net/")
	t.Setenv("PASTEBIN_PUBLIC_HOST", "pastebin.harm.org")
	t.Setenv("PASTEBIN_LISTEN", "127.0.0.1:9090")
	t.Setenv("PASTEBIN_DB", "/tmp/pastebin.db")
	t.Setenv("PASTEBIN_MAX_BYTES", "64")
	t.Setenv("PASTEBIN_DEFAULT_TTL", "1h")
	t.Setenv("PASTEBIN_MAX_TTL", "24h")

	cfg, err := parseConfig([]string{"--listen", "127.0.0.1:8081"})
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}
	if cfg.BaseURL != "https://paste.example.ts.net" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.PublicHost != "pastebin.harm.org" {
		t.Fatalf("PublicHost = %q", cfg.PublicHost)
	}
	if cfg.Listen != "127.0.0.1:8081" {
		t.Fatalf("Listen = %q", cfg.Listen)
	}
	if cfg.DBPath != "/tmp/pastebin.db" || cfg.MaxBytes != 64 || cfg.DefaultTTL != time.Hour || cfg.MaxTTL != 24*time.Hour {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestParseConfigReadsPublishTokenFile(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "publish-token")
	publishToken := "AQIDBAUGBwgJCgsMDQ4PEA"
	if err := os.WriteFile(tokenFile, []byte(publishToken+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PASTEBIN_PUBLIC_HOST", "pastebin.harm.org")
	t.Setenv("PASTEBIN_PUBLISH_TOKEN_FILE", tokenFile)

	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}
	if cfg.PublishToken != publishToken {
		t.Fatal("parseConfig() did not load the publishing token")
	}
}

func TestParseConfigRejectsShortPublishToken(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "publish-token")
	if err := os.WriteFile(tokenFile, []byte("too-short\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PASTEBIN_PUBLIC_HOST", "pastebin.harm.org")
	t.Setenv("PASTEBIN_PUBLISH_TOKEN_FILE", tokenFile)

	if _, err := parseConfig(nil); err == nil || !strings.Contains(err.Error(), "22 base64url characters") {
		t.Fatalf("parseConfig() error = %v, want publish token format error", err)
	}
}

func TestParseConfigRejectsPublishTokenWithoutPublicHost(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "publish-token")
	if err := os.WriteFile(tokenFile, []byte(strings.Repeat("c", 64)), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PASTEBIN_PUBLISH_TOKEN_FILE", tokenFile)

	if _, err := parseConfig(nil); err == nil || !strings.Contains(err.Error(), "public host") {
		t.Fatalf("parseConfig() error = %v, want public host error", err)
	}
}

func TestParseConfigRejectsInvalidLimits(t *testing.T) {
	if _, err := parseConfig([]string{"--max-bytes", "0"}); err == nil {
		t.Fatal("parseConfig(max bytes 0) error = nil, want error")
	}
	if _, err := parseConfig([]string{"--max-ttl", "0"}); err == nil {
		t.Fatal("parseConfig(max ttl 0) error = nil, want error")
	}
	if _, err := parseConfig([]string{"--default-ttl", (paste.MaxTTL + time.Hour).String()}); err == nil {
		t.Fatal("parseConfig(default ttl over max) error = nil, want error")
	}
}

func TestParseConfigRejectsPublicHostWithSchemePathOrPort(t *testing.T) {
	for _, publicHost := range []string{
		"https://pastebin.harm.org",
		"pastebin.harm.org/public",
		"pastebin.harm.org:443",
	} {
		t.Run(publicHost, func(t *testing.T) {
			if _, err := parseConfig([]string{"--public-host", publicHost}); err == nil {
				t.Fatalf("parseConfig(public host %q) error = nil, want error", publicHost)
			}
		})
	}
}
