package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A typical ~/.wakatime.cfg written by the Hackatime setup script plus
// settings other WakaTime plugins rely on.
const sharedConfig = `# keep this comment
[settings]
api_url = https://hackatime.hackclub.com/api/hackatime/v1
api_key = 00000000-0000-4000-8000-000000000000
heartbeat_rate_limit_seconds = 30
hide_branch_names = true

[projectmap]
~/work/foo = Foo Project

[git]
submodules_disabled = true
`

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, DefaultConfigFile)
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func readConfig(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSaveKeepsOtherSettings(t *testing.T) {
	path := writeConfig(t, sharedConfig)

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}
	cfg.APIKey = "11111111-1111-4111-8111-111111111111"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	want := strings.Replace(sharedConfig,
		"api_key = 00000000-0000-4000-8000-000000000000",
		"api_key = 11111111-1111-4111-8111-111111111111", 1)
	if got := readConfig(t, path); got != want {
		t.Errorf("Save() changed more than api_key.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestSaveAddsNewKeyInsideSettings(t *testing.T) {
	path := writeConfig(t, sharedConfig)

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}
	cfg.Project = "cloudfall"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	want := strings.Replace(sharedConfig,
		"hide_branch_names = true\n",
		"hide_branch_names = true\nproject = cloudfall\n", 1)
	if got := readConfig(t, path); got != want {
		t.Errorf("project not added to [settings] cleanly.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestSaveWithoutChangesLeavesFileUntouched(t *testing.T) {
	path := writeConfig(t, sharedConfig)

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	if got := readConfig(t, path); got != sharedConfig {
		t.Errorf("Save() with no changes modified the file:\n%s", got)
	}
}

func TestSaveReplacesMultilineValue(t *testing.T) {
	path := writeConfig(t, "")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}
	cfg.Exclude = []string{"*.log", "*.tmp"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	cfg.Exclude = []string{"^/private/"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	got := readConfig(t, path)
	if got != "[settings]\nexclude = ^/private/\n" {
		t.Errorf("multi-line exclude not replaced cleanly:\n%s", got)
	}

	cfg2, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}
	if len(cfg2.Exclude) != 1 || cfg2.Exclude[0] != "^/private/" {
		t.Errorf("Exclude = %q, want [^/private/]", cfg2.Exclude)
	}
}

func TestNewConfigFileIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions only")
	}
	path := writeConfig(t, "")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}
	cfg.APIKey = "abc"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file mode = %o, want 600 since it holds the API key", perm)
	}
}
