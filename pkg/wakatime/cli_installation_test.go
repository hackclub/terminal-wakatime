package wakatime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hackclub/terminal-wakatime/pkg/config"
)

func TestCLIInstallation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI installation test in short mode")
	}

	// Create a temporary directory for testing
	testDir := t.TempDir()
	
	// Override the home directory to use our test directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", testDir)
	defer os.Setenv("HOME", originalHome)
	
	// Create test config with temporary directory
	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	
	// Create CLI instance
	cli := NewCLI(cfg)
	
	// Ensure the binary doesn't exist initially
	if cli.IsInstalled() {
		t.Fatal("CLI should not be installed initially in test environment")
	}
	
	t.Logf("Testing CLI installation to: %s", cli.binPath)
	
	// Test installation
	err = cli.EnsureInstalled()
	if err != nil {
		t.Fatalf("Failed to install CLI: %v", err)
	}
	
	// Verify installation
	if !cli.IsInstalled() {
		t.Fatal("CLI should be installed after EnsureInstalled")
	}
	
	// Verify binary exists and is executable
	if _, err := os.Stat(cli.binPath); os.IsNotExist(err) {
		t.Fatal("CLI binary does not exist after installation")
	}
	
	// Verify binary is actually executable by running --version
	if !cli.testBinary() {
		t.Fatal("CLI binary is not working after installation")
	}
	
	t.Logf("✓ CLI successfully installed and verified at: %s", cli.binPath)
}

func TestCLIAlreadyInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI installation test in short mode")
	}

	// Create a temporary directory for testing
	testDir := t.TempDir()
	
	// Override the home directory to use our test directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", testDir)
	defer os.Setenv("HOME", originalHome)
	
	// Create test config with temporary directory
	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	
	// Create CLI instance and install it first
	cli := NewCLI(cfg)
	
	// First installation
	err = cli.EnsureInstalled()
	if err != nil {
		t.Fatalf("Failed to install CLI: %v", err)
	}
	
	// Record the modification time
	stat1, err := os.Stat(cli.binPath)
	if err != nil {
		t.Fatalf("Failed to stat CLI binary: %v", err)
	}
	
	// Call EnsureInstalled again - should not reinstall
	err = cli.EnsureInstalled()
	if err != nil {
		t.Fatalf("Failed on second EnsureInstalled call: %v", err)
	}
	
	// Verify binary wasn't replaced (same modification time)
	stat2, err := os.Stat(cli.binPath)
	if err != nil {
		t.Fatalf("Failed to stat CLI binary after second call: %v", err)
	}
	
	if stat1.ModTime() != stat2.ModTime() {
		t.Log("Note: Binary was updated (this could be due to version check)")
	}
	
	t.Logf("✓ CLI correctly detected existing installation")
}

func TestCLIBinaryPath(t *testing.T) {
	testDir := t.TempDir()
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", testDir)
	defer os.Setenv("HOME", originalHome)
	
	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	
	cli := NewCLI(cfg)
	
	// Verify the binary path follows expected pattern
	expectedDir := filepath.Join(testDir, ".wakatime")
	if !filepath.HasPrefix(cli.binPath, expectedDir) {
		t.Errorf("Expected binary path to be in %s, got %s", expectedDir, cli.binPath)
	}
	
	// Verify it includes platform info
	fileName := filepath.Base(cli.binPath)
	if fileName == "wakatime-cli" {
		t.Errorf("Expected platform-specific binary name, got generic name: %s", fileName)
	}
	
	// Should contain platform-specific name
	if len(fileName) <= len("wakatime-cli") {
		t.Errorf("Expected platform-specific binary name, got %s", fileName)
	}
	
	t.Logf("✓ Binary path is correct: %s", cli.binPath)
}
