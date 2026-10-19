package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func loadConfigFile(t *testing.T, content string) *Config {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, DefaultConfigFile), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}
	return cfg
}

// The exclude/include format documented in wakatime-cli's USAGE.md.
func TestLoadMultilineExcludeInclude(t *testing.T) {
	cfg := loadConfigFile(t, `[settings]
api_url = https://hackatime.hackclub.com/api/hackatime/v1
exclude =
    ^COMMIT_EDITMSG$
    ^TAG_EDITMSG$
    ^/var/(?!www/).*
    ^/etc/
include =
    .*
api_key = 00000000-0000-4000-8000-000000000000
`)

	wantExclude := []string{"^COMMIT_EDITMSG$", "^TAG_EDITMSG$", "^/var/(?!www/).*", "^/etc/"}
	if !reflect.DeepEqual(cfg.Exclude, wantExclude) {
		t.Errorf("Exclude = %q, want %q", cfg.Exclude, wantExclude)
	}
	if !reflect.DeepEqual(cfg.Include, []string{".*"}) {
		t.Errorf("Include = %q, want [.*]", cfg.Include)
	}
	if cfg.APIKey != "00000000-0000-4000-8000-000000000000" {
		t.Errorf("APIKey after the lists = %q, want it loaded", cfg.APIKey)
	}
	if cfg.APIUrl != "https://hackatime.hackclub.com/api/hackatime/v1" {
		t.Errorf("APIUrl = %q", cfg.APIUrl)
	}
}

func TestLoadSingleLineExclude(t *testing.T) {
	cfg := loadConfigFile(t, "[settings]\nexclude = ^/tmp/\n")

	if !reflect.DeepEqual(cfg.Exclude, []string{"^/tmp/"}) {
		t.Errorf("Exclude = %q, want [^/tmp/]", cfg.Exclude)
	}
}

func TestLoadEmptyExclude(t *testing.T) {
	cfg := loadConfigFile(t, "[settings]\nexclude =\napi_key = abc\n")

	if len(cfg.Exclude) != 0 {
		t.Errorf("Exclude = %q, want empty", cfg.Exclude)
	}
	if cfg.APIKey != "abc" {
		t.Errorf("APIKey = %q, want abc", cfg.APIKey)
	}
}
