package monitor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
)

func TestNewMonitor(t *testing.T) {
	cfg := &config.Config{}
	monitor := NewMonitor(cfg)

	if monitor == nil {
		t.Error("Expected monitor to be created")
	}

	if monitor.config != cfg {
		t.Error("Expected monitor to store config reference")
	}
}

func TestProcessCommand(t *testing.T) {
	cfg := &config.Config{
		MinCommandTime: 1 * time.Second,
		Debug:          false, // Disable logging for tests
	}
	monitor := NewMonitor(cfg)

	// Test command that meets minimum duration
	err := monitor.ProcessCommand("ls -la", 2*time.Second, "/tmp")
	if err == nil {
		t.Log("ProcessCommand succeeded (expected in test environment)")
	} else {
		// Expected to fail in test environment due to missing wakatime-cli
		if !strings.Contains(err.Error(), "wakatime-cli") && !strings.Contains(err.Error(), "executable") {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Test command that doesn't meet minimum duration
	err = monitor.ProcessCommand("ls", 500*time.Millisecond, "/tmp")
	if err != nil {
		t.Errorf("Expected short command to be skipped, got error: %v", err)
	}
}

func TestProcessFileEdit(t *testing.T) {
	cfg := &config.Config{}
	monitor := NewMonitor(cfg)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	// Create test file
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := monitor.ProcessFileEdit(testFile, false)
	if err == nil {
		t.Log("ProcessFileEdit succeeded (expected in test environment)")
	} else {
		// Expected to fail in test environment due to missing wakatime-cli
		if !strings.Contains(err.Error(), "wakatime-cli") && !strings.Contains(err.Error(), "executable") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestParseLogLine(t *testing.T) {
	cfg := &config.Config{}
	monitor := NewMonitor(cfg)

	// Test valid log line
	timestamp := time.Now().Format(time.RFC3339)
	logLine := timestamp + "\t/home/user\t2s\tgit status"

	event, err := monitor.parseLogLine(logLine)
	if err != nil {
		t.Errorf("parseLogLine failed: %v", err)
	}

	if event.Command != "git status" {
		t.Errorf("Expected command 'git status', got '%s'", event.Command)
	}

	if event.WorkingDir != "/home/user" {
		t.Errorf("Expected working dir '/home/user', got '%s'", event.WorkingDir)
	}

	if event.Duration != 2*time.Second {
		t.Errorf("Expected duration 2s, got %v", event.Duration)
	}

	// Test invalid log line
	_, err = monitor.parseLogLine("invalid\tline")
	if err == nil {
		t.Error("Expected parseLogLine to fail with invalid line")
	}
}

func TestParseTrackCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *CommandEvent
		hasError bool
	}{
		{
			name: "valid command",
			args: []string{"--command", "git status", "--duration", "5", "--pwd", "/home/user"},
			expected: &CommandEvent{
				Command:    "git status",
				Duration:   5 * time.Second,
				WorkingDir: "/home/user",
			},
			hasError: false,
		},
		{
			name:     "missing command",
			args:     []string{"--duration", "5"},
			expected: nil,
			hasError: true,
		},
		{
			name: "command without optional args",
			args: []string{"--command", "ls"},
			expected: &CommandEvent{
				Command:  "ls",
				Duration: 0,
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseTrackCommand(tt.args)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if event.Command != tt.expected.Command {
				t.Errorf("Expected command '%s', got '%s'", tt.expected.Command, event.Command)
			}

			if event.Duration != tt.expected.Duration {
				t.Errorf("Expected duration %v, got %v", tt.expected.Duration, event.Duration)
			}

			if tt.expected.WorkingDir != "" && event.WorkingDir != tt.expected.WorkingDir {
				t.Errorf("Expected working dir '%s', got '%s'", tt.expected.WorkingDir, event.WorkingDir)
			}
		})
	}
}

func TestGetStatus(t *testing.T) {
	cfg := &config.Config{
		APIKey:             "test-key",
		Debug:              true,
		HeartbeatFrequency: 2 * time.Minute,
	}
	monitor := NewMonitor(cfg)

	status, err := monitor.GetStatus()
	if err != nil {
		t.Errorf("GetStatus failed: %v", err)
	}

	// Check expected status fields
	expectedFields := []string{
		"api_key_configured",
		"debug_enabled",
		"heartbeat_frequency",
	}

	for _, field := range expectedFields {
		if _, exists := status[field]; !exists {
			t.Errorf("Expected status field '%s' to be present", field)
		}
	}

	// Check values
	if status["api_key_configured"] != true {
		t.Error("Expected api_key_configured to be true")
	}

	if status["debug_enabled"] != true {
		t.Error("Expected debug_enabled to be true")
	}

	if status["heartbeat_frequency"] != "2m0s" {
		t.Errorf("Expected heartbeat_frequency to be '2m0s', got '%v'", status["heartbeat_frequency"])
	}
}

func TestLogCommand(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		Debug: true,
	}

	// Override wakaTimeDir to use temp directory
	cfg = &config.Config{
		Debug: true,
	}

	monitor := &Monitor{
		config:  cfg,
		logFile: filepath.Join(tempDir, "commands.log"),
	}

	// Test logging
	monitor.logCommand("git status", 2*time.Second, "/home/user")

	// Check if log file was created
	if _, err := os.Stat(monitor.logFile); os.IsNotExist(err) {
		t.Error("Expected log file to be created")
	}

	// Read log file content
	content, err := os.ReadFile(monitor.logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "git status") {
		t.Error("Expected log to contain command")
	}

	if !strings.Contains(logContent, "/home/user") {
		t.Error("Expected log to contain working directory")
	}

	if !strings.Contains(logContent, "2s") {
		t.Error("Expected log to contain duration")
	}
}
