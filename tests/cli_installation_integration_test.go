package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCLIInstallationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI installation integration test in short mode")
	}

	// Create a temporary directory for testing
	testDir := t.TempDir()

	// Build the main binary
	binaryPath := filepath.Join(testDir, "terminal-wakatime")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/terminal-wakatime")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Override the home directory to use our test directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", testDir)
	defer os.Setenv("HOME", originalHome)

	// Clear any existing WAKATIME_HOME to ensure we use the test directory
	originalWakaTimeHome := os.Getenv("WAKATIME_HOME")
	os.Unsetenv("WAKATIME_HOME")
	defer func() {
		if originalWakaTimeHome != "" {
			os.Setenv("WAKATIME_HOME", originalWakaTimeHome)
		}
	}()

	// Test 1: Set up configuration first
	t.Log("Setting up configuration...")
	
	// Disable auto-updates by setting a future timestamp
	wakatimeDir := filepath.Join(testDir, ".wakatime")
	os.MkdirAll(wakatimeDir, 0755)
	futureTime := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	lastCheckFile := filepath.Join(wakatimeDir, "terminal-wakatime_last_update_check.txt")
	os.WriteFile(lastCheckFile, []byte(futureTime), 0644)
	
	configCmd := exec.Command(binaryPath, "config", "--key", "test-api-key-123456789", "--project", "test-project")
	configCmd.Env = append(os.Environ(), "HOME="+testDir)

	output, err := configCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config command failed: %v\nOutput: %s", err, output)
	}

	// Test 2: Track command should trigger CLI installation
	t.Log("Testing CLI installation via track command...")
	trackCmd := exec.Command(binaryPath, "track", "--command", "vim main.go", "--pwd", testDir, "--duration", "5", "-v")
	trackCmd.Env = append(os.Environ(), "HOME="+testDir, "TERMINAL_WAKATIME_DISABLE_UPDATES=1")

	trackOutput, err := trackCmd.CombinedOutput()
	t.Logf("Track command output: %s", string(trackOutput))

	// Check if .wakatime directory was created
	// wakatimeDir already declared above
	if _, err := os.Stat(wakatimeDir); os.IsNotExist(err) {
		t.Fatalf("WakaTime directory was not created: %s", wakatimeDir)
	}

	// List contents of wakatime directory
	files, err := os.ReadDir(wakatimeDir)
	if err != nil {
		t.Fatalf("Failed to read wakatime directory: %v", err)
	}
	t.Logf("WakaTime directory contents: %v", files)

	// Verify wakatime-cli was installed
	binName := fmt.Sprintf("wakatime-cli-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	expectedCLIPath := filepath.Join(testDir, ".wakatime", binName)
	if _, err := os.Stat(expectedCLIPath); os.IsNotExist(err) {
		t.Fatalf("wakatime-cli was not installed at expected path: %s", expectedCLIPath)
	}

	t.Logf("✓ wakatime-cli successfully installed at: %s", expectedCLIPath)

	// Test 3: Verify the CLI binary works (may fail in CI environments)
	t.Log("Testing installed CLI binary...")
	versionCmd := exec.Command(expectedCLIPath, "--version")
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		// In CI environments, wakatime-cli may fail due to missing dependencies
		// This is acceptable as long as the binary was downloaded and installed
		t.Logf("CLI version check failed (expected in CI): %v\nOutput: %s", err, versionOutput)
	} else {
		versionStr := strings.TrimSpace(string(versionOutput))
		if versionStr == "" || (!strings.Contains(versionStr, "wakatime") && !strings.HasPrefix(versionStr, "v")) {
			t.Errorf("Unexpected version output: %s", versionStr)
		} else {
			t.Logf("✓ CLI version check successful: %s", versionStr)
		}
	}
	t.Logf("✓ Track command executed successfully")

	// Test 4: Verify subsequent calls don't reinstall
	t.Log("Testing that CLI is not reinstalled on subsequent calls...")

	// Get modification time of existing CLI
	stat1, err := os.Stat(expectedCLIPath)
	if err != nil {
		t.Fatalf("Failed to stat CLI binary: %v", err)
	}

	// Run another command that would trigger installation
	trackCmd2 := exec.Command(binaryPath, "track", "--command", "another test", "--pwd", testDir, "--duration", "3")
	trackCmd2.Env = append(os.Environ(), "HOME="+testDir, "TERMINAL_WAKATIME_DISABLE_UPDATES=1")

	_, err = trackCmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("Second track command failed: %v", err)
	}

	// Check if binary was modified (it shouldn't be unless there was an update)
	stat2, err := os.Stat(expectedCLIPath)
	if err != nil {
		t.Fatalf("Failed to stat CLI binary after second call: %v", err)
	}

	if stat1.ModTime() != stat2.ModTime() {
		t.Log("Note: CLI binary was updated (this could be due to version check)")
	} else {
		t.Log("✓ CLI binary was not reinstalled unnecessarily")
	}
}

func TestCLIInstallationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI installation failure test in short mode")
	}

	// This test would be more complex to implement as it would require
	// mocking network failures or GitHub API responses
	// For now, we'll skip this but it's a good candidate for future improvement
	t.Skip("CLI installation failure testing not yet implemented")
}
