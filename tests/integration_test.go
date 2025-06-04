//go:build integration
// +build integration

package tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Integration tests that require a built binary and real shell environment
// Run with: go test -tags=integration ./tests/

const (
	testTimeout = 30 * time.Second
)

func TestConfigurationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binaryPath := buildTestBinary(t)
	testDir := t.TempDir()

	// Set up isolated environment
	os.Setenv("HOME", testDir)
	defer os.Unsetenv("HOME")

	// Test config show (should work even without API key)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "config", "--show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Config show failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Configuration file:") {
		t.Error("Expected config output to contain configuration file path")
	}

	// Test setting API key
	cmd = exec.CommandContext(ctx, binaryPath, "config", "--key", "test-api-key-12345")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Config set key failed: %v\nOutput: %s", err, output)
	}

	// Verify API key was saved
	cmd = exec.CommandContext(ctx, binaryPath, "config", "--show")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Config show after set failed: %v\nOutput: %s", err, output)
	}

	outputStr = string(output)
	if !strings.Contains(outputStr, "test-***-12345") && !strings.Contains(outputStr, "****") {
		t.Error("Expected config to show masked API key")
	}

	// Test setting project
	cmd = exec.CommandContext(ctx, binaryPath, "config", "--project", "test-project")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Config set project failed: %v\nOutput: %s", err, output)
	}

	// Verify project was saved
	cmd = exec.CommandContext(ctx, binaryPath, "config", "--show")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Config show after project set failed: %v\nOutput: %s", err, output)
	}

	outputStr = string(output)
	if !strings.Contains(outputStr, "test-project") {
		t.Error("Expected config to show project name")
	}
}

func TestCommandTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binaryPath := buildTestBinary(t)
	testDir := t.TempDir()

	// Set up isolated environment
	os.Setenv("HOME", testDir)
	defer os.Unsetenv("HOME")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test track command
	cmd := exec.CommandContext(ctx, binaryPath, "track", "--command", "git status", "--duration", "5", "--pwd", testDir)
	output, err := cmd.CombinedOutput()

	// This might fail due to missing wakatime-cli, but should not panic
	if err != nil {
		outputStr := string(output)
		// Acceptable errors in test environment
		if !strings.Contains(outputStr, "wakatime-cli") &&
			!strings.Contains(outputStr, "API key") &&
			!strings.Contains(err.Error(), "executable") {
			t.Errorf("Unexpected track command error: %v\nOutput: %s", err, output)
		}
	}

	// Test status command
	cmd = exec.CommandContext(ctx, binaryPath, "status")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Status command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Terminal WakaTime Status") {
		t.Error("Expected status output to contain header")
	}
}

func TestHelpCommands(t *testing.T) {
	binaryPath := buildTestBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test main help
	cmd := exec.CommandContext(ctx, binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Help command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	expectedSubcommands := []string{"init", "config", "heartbeat", "track", "status", "test", "deps"}
	for _, subcmd := range expectedSubcommands {
		if !strings.Contains(outputStr, subcmd) {
			t.Errorf("Expected help to mention subcommand '%s'", subcmd)
		}
	}

	// Test subcommand help
	cmd = exec.CommandContext(ctx, binaryPath, "config", "--help")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Config help failed: %v\nOutput: %s", err, output)
	}

	outputStr = string(output)
	if !strings.Contains(outputStr, "--key") {
		t.Error("Expected config help to mention --key flag")
	}
}

func TestInitCommand(t *testing.T) {
	binaryPath := buildTestBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test init command output
	cmd := exec.CommandContext(ctx, binaryPath, "init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Init command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should contain shell hook functions
	expectedElements := []string{
		"__terminal_wakatime_preexec",
		"__terminal_wakatime_postexec",
		binaryPath, // Should reference the binary path
	}

	for _, element := range expectedElements {
		if !strings.Contains(outputStr, element) {
			t.Errorf("Expected init output to contain '%s'", element)
		}
	}

	// Output should be valid shell code (basic check)
	if !strings.Contains(outputStr, "function") && !strings.Contains(outputStr, "()") {
		t.Error("Expected init output to contain shell function definitions")
	}
}

func TestDepsCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binaryPath := buildTestBinary(t)
	testDir := t.TempDir()

	// Set up isolated environment
	os.Setenv("HOME", testDir)
	defer os.Unsetenv("HOME")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Longer timeout for download
	defer cancel()

	// Test deps status (should show not installed initially)
	cmd := exec.CommandContext(ctx, binaryPath, "deps", "--status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Deps status failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "not installed") && !strings.Contains(outputStr, "installed") {
		t.Error("Expected deps status to show installation status")
	}
}

// Helper functions

func buildTestBinary(t *testing.T) string {
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "terminal-wakatime-test")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/terminal-wakatime")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	return binaryPath
}

func runShellScript(t *testing.T, script string, timeout time.Duration) error {
	return runScriptWithShell(t, script, "bash", timeout)
}

func runZshScript(t *testing.T, script string, timeout time.Duration) error {
	return runScriptWithShell(t, script, "zsh", timeout)
}

func runFishScript(t *testing.T, script string, timeout time.Duration) error {
	return runScriptWithShell(t, script, "fish", timeout)
}

func runScriptWithShell(t *testing.T, script, shell string, timeout time.Duration) error {
	tempFile := filepath.Join(t.TempDir(), "test_script")
	if err := os.WriteFile(tempFile, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write test script: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, tempFile)
	output, err := cmd.CombinedOutput()

	t.Logf("Script output: %s", output)

	if err != nil {
		return fmt.Errorf("script execution failed: %w\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "SUCCESS") {
		return fmt.Errorf("script did not report success\nOutput: %s", output)
	}

	return nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
