package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	// Create a temporary home directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}

	// Test default values
	if cfg.APIUrl != DefaultAPIURL {
		t.Errorf("Expected API URL %s, got %s", DefaultAPIURL, cfg.APIUrl)
	}

	if cfg.HeartbeatFrequency != DefaultHeartbeatFrequency {
		t.Errorf("Expected heartbeat frequency %v, got %v", DefaultHeartbeatFrequency, cfg.HeartbeatFrequency)
	}

	if cfg.MinCommandTime != DefaultMinCommandTime {
		t.Errorf("Expected min command time %v, got %v", DefaultMinCommandTime, cfg.MinCommandTime)
	}

	expectedConfigFile := filepath.Join(tempDir, DefaultConfigFile)
	if cfg.ConfigFile() != expectedConfigFile {
		t.Errorf("Expected config file %s, got %s", expectedConfigFile, cfg.ConfigFile())
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}

	// Set some test values
	cfg.APIKey = "test-api-key"
	cfg.Debug = true
	cfg.Project = "test-project"
	cfg.HideFilenames = true

	// Save configuration
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Create a new config instance and load
	cfg2, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}

	// Verify values were loaded correctly
	if cfg2.APIKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", cfg2.APIKey)
	}

	if !cfg2.Debug {
		t.Errorf("Expected debug to be true")
	}

	if cfg2.Project != "test-project" {
		t.Errorf("Expected project 'test-project', got '%s'", cfg2.Project)
	}

	if !cfg2.HideFilenames {
		t.Errorf("Expected hide filenames to be true")
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := &Config{}

	// Test validation with empty API key
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail with empty API key")
	}

	// Test validation with API key but empty URL
	cfg.APIKey = "test-key"
	cfg.APIUrl = ""
	err = cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail with empty API URL")
	}

	// Test successful validation
	cfg.APIUrl = "https://api.wakatime.com"
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}
}

func TestConfigEnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Clear environment variables first
	os.Unsetenv("TERMINAL_WAKATIME_HEARTBEAT_FREQUENCY")
	os.Unsetenv("TERMINAL_WAKATIME_MIN_COMMAND_TIME")
	os.Unsetenv("TERMINAL_WAKATIME_DISABLE_EDITOR_SUGGESTIONS")

	// Set environment variables
	os.Setenv("TERMINAL_WAKATIME_HEARTBEAT_FREQUENCY", "300")
	os.Setenv("TERMINAL_WAKATIME_MIN_COMMAND_TIME", "5")
	os.Setenv("TERMINAL_WAKATIME_DISABLE_EDITOR_SUGGESTIONS", "true")

	defer func() {
		os.Unsetenv("TERMINAL_WAKATIME_HEARTBEAT_FREQUENCY")
		os.Unsetenv("TERMINAL_WAKATIME_MIN_COMMAND_TIME")
		os.Unsetenv("TERMINAL_WAKATIME_DISABLE_EDITOR_SUGGESTIONS")
	}()

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}

	// Reload config to pick up environment variables
	if err := cfg.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	expectedFreq := 300 * time.Second
	if cfg.HeartbeatFrequency != expectedFreq {
		t.Errorf("Expected heartbeat frequency %v, got %v", expectedFreq, cfg.HeartbeatFrequency)
	}

	expectedMinTime := 5 * time.Second
	if cfg.MinCommandTime != expectedMinTime {
		t.Errorf("Expected min command time %v, got %v", expectedMinTime, cfg.MinCommandTime)
	}

	if !cfg.DisableEditorSuggestions {
		t.Errorf("Expected editor suggestions to be disabled, got %v", cfg.DisableEditorSuggestions)
	}
}

func TestConfigExcludeInclude(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}

	// Set exclude and include patterns
	cfg.Exclude = []string{"*.log", "*.tmp"}
	cfg.Include = []string{"*.go", "*.js"}

	// Save and reload
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	cfg2, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() failed: %v", err)
	}

	if len(cfg2.Exclude) != 2 || cfg2.Exclude[0] != "*.log" || cfg2.Exclude[1] != "*.tmp" {
		t.Errorf("Expected exclude patterns [*.log, *.tmp], got %v", cfg2.Exclude)
	}

	if len(cfg2.Include) != 2 || cfg2.Include[0] != "*.go" || cfg2.Include[1] != "*.js" {
		t.Errorf("Expected include patterns [*.go, *.js], got %v", cfg2.Include)
	}
}
