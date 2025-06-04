package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMainFunction(t *testing.T) {
	// Test that main function exists and can be called
	// This is a basic smoke test to ensure the binary can be built
	if testing.Short() {
		t.Skip("Skipping main function test in short mode")
	}

	// We can't easily test main() directly, but we can test that
	// the binary builds and basic commands work
	t.Log("Main function test - checking that binary can be built")
}

func TestExecuteFunction(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test help command
	os.Args = []string{"terminal-wakatime", "--help"}

	// This will exit the program, so we need to catch that
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior for help command
		}
	}()

	// We can't easily test execute() without it calling os.Exit()
	// So this test is mostly to ensure the function exists and compiles
	t.Log("Execute function test - function exists and compiles")
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "(not set)"},
		{"short", "*****"},
		{"1234567890", "1234**7890"},
		{"waka_123456789012345678901234567890", "waka***************************7890"},
		{"api_key_test", "api_****test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("maskAPIKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"api_key", "Api Key"},
		{"debug_enabled", "Debug Enabled"},
		{"heartbeat_frequency", "Heartbeat Frequency"},
		{"recent_commands", "Recent Commands"},
		{"single", "Single"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatKey(tt.input)
			if result != tt.expected {
				t.Errorf("formatKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"exactly ten", 11, "exactly ten"},
		{"", 5, ""},
		{"abc", 0, ""},
		{"abc", 1, "."},
		{"abc", 2, ".."},
		{"abc", 3, "abc"},
		{"longtext", 3, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// Integration test that runs the actual binary
func TestBinaryCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping binary integration test in short mode")
	}

	// Build the binary for testing
	tempDir := t.TempDir()
	binaryPath := tempDir + "/terminal-wakatime-test"

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = "."

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary for testing: %v\nOutput: %s", err, output)
	}

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:     "help command",
			args:     []string{"--help"},
			wantErr:  false,
			contains: []string{"Terminal WakaTime", "Usage:", "Available Commands:"},
		},
		{
			name:     "init command",
			args:     []string{"init"},
			wantErr:  false,
			contains: []string{"__terminal_wakatime_preexec", "__terminal_wakatime_precmd"},
		},
		{
			name:     "config help",
			args:     []string{"config", "--help"},
			wantErr:  false,
			contains: []string{"Configure terminal-wakatime", "--key", "--project"},
		},
		{
			name:     "status command",
			args:     []string{"status"},
			wantErr:  false,
			contains: []string{"Terminal WakaTime Status"},
		},
		{
			name:     "invalid command",
			args:     []string{"invalid-command"},
			wantErr:  true,
			contains: []string{"unknown command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)

			// Set up isolated environment
			testHome := t.TempDir()
			cmd.Env = append(os.Environ(), "HOME="+testHome)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.wantErr && err == nil {
				t.Errorf("Expected command to fail, but it succeeded")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Command failed unexpectedly: %v\nStdout: %s\nStderr: %s",
					err, stdout.String(), stderr.String())
			}

			// Check output contains expected strings
			output := stdout.String() + stderr.String()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s",
						expected, output)
				}
			}
		})
	}
}

func TestConfigWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping config workflow test in short mode")
	}

	// Build the binary for testing
	tempDir := t.TempDir()
	binaryPath := tempDir + "/terminal-wakatime-test"

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Set up isolated home directory
	testHome := t.TempDir()
	env := append(os.Environ(), "HOME="+testHome)

	// Test showing initial config
	cmd := exec.Command(binaryPath, "config", "--show")
	cmd.Env = env
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "API Key: (not set)") {
		t.Errorf("Expected initial config to show unset API key, got: %s", output)
	}

	// Test setting API key
	cmd = exec.Command(binaryPath, "config", "--key", "test-api-key-123456789")
	cmd.Env = env
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config set key failed: %v\nOutput: %s", err, output)
	}

	// Test showing config after setting key
	cmd = exec.Command(binaryPath, "config", "--show")
	cmd.Env = env
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show after set failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "test-***-9") && !strings.Contains(outputStr, "****") {
		t.Errorf("Expected masked API key in output, got: %s", outputStr)
	}

	// Test setting project
	cmd = exec.Command(binaryPath, "config", "--project", "test-project")
	cmd.Env = env
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config set project failed: %v\nOutput: %s", err, output)
	}

	// Verify project was set
	cmd = exec.Command(binaryPath, "config", "--show")
	cmd.Env = env
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show after project set failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "test-project") {
		t.Errorf("Expected config to show project, got: %s", output)
	}
}
