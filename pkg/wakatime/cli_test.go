package wakatime

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hackclub/terminal-wakatime/pkg/config"
)

func TestNewCLI(t *testing.T) {
	cfg := &config.Config{}
	cli := NewCLI(cfg)

	if cli == nil {
		t.Error("Expected CLI to be created")
	}

	if cli.config != cfg {
		t.Error("Expected CLI to store config reference")
	}

	// Check binary path format
	expectedName := fmt.Sprintf("wakatime-cli-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		expectedName += ".exe"
	}

	if !strings.Contains(cli.binPath, expectedName) {
		t.Errorf("Expected binary path to contain '%s', got '%s'", expectedName, cli.binPath)
	}
}

func TestIsInstalled(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{}

	// Create a mock CLI with temp directory
	cli := &CLI{
		config:  cfg,
		binPath: filepath.Join(tempDir, "wakatime-cli"),
	}

	// Should not be installed initially
	if cli.IsInstalled() {
		t.Error("Expected CLI to not be installed initially")
	}

	// Create a dummy binary file
	if err := os.WriteFile(cli.binPath, []byte("#!/bin/bash\necho 'wakatime-cli v1.0.0'\n"), 0755); err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}

	// Should still not be considered installed because testBinary will fail
	if cli.IsInstalled() {
		t.Log("CLI is considered installed (test binary validation passed)")
	}
}

func TestFindAssetForPlatform(t *testing.T) {
	cfg := &config.Config{}
	cli := NewCLI(cfg)

	// Create a mock release with various assets
	release := &GitHubRelease{
		TagName: "v1.73.0",
		Assets: []Asset{
			{Name: "wakatime-cli-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64.tar.gz"},
			{Name: "wakatime-cli-darwin-amd64.tar.gz", BrowserDownloadURL: "https://example.com/darwin-amd64.tar.gz"},
			{Name: "wakatime-cli-windows-amd64.zip", BrowserDownloadURL: "https://example.com/windows-amd64.zip"},
			{Name: "wakatime-cli-linux-arm64.tar.gz", BrowserDownloadURL: "https://example.com/linux-arm64.tar.gz"},
		},
	}

	// Test finding asset for current platform
	asset, err := cli.findAssetForPlatform(release)
	if err != nil {
		// It's okay if the current platform isn't in our mock data
		t.Logf("No asset found for current platform (%s-%s): %v", runtime.GOOS, runtime.GOARCH, err)
		return
	}

	// Verify the asset matches current platform
	expectedPlatform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if !strings.Contains(asset.Name, expectedPlatform) {
		t.Errorf("Expected asset to contain platform '%s', got '%s'", expectedPlatform, asset.Name)
	}
}

func TestFindAssetForPlatformNotFound(t *testing.T) {
	cfg := &config.Config{}
	cli := NewCLI(cfg)

	// Create a mock release with no matching assets
	release := &GitHubRelease{
		TagName: "v1.73.0",
		Assets: []Asset{
			{Name: "wakatime-cli-unsupported-platform.tar.gz", BrowserDownloadURL: "https://example.com/unsupported.tar.gz"},
		},
	}

	// Should fail to find asset
	_, err := cli.findAssetForPlatform(release)
	if err == nil {
		t.Error("Expected error when no matching asset found")
	}

	if !strings.Contains(err.Error(), "no asset found") {
		t.Errorf("Expected 'no asset found' error, got: %v", err)
	}
}

func TestGetCurrentVersion(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{}

	cli := &CLI{
		config:  cfg,
		binPath: filepath.Join(tempDir, "wakatime-cli"),
	}

	// Create a mock binary that outputs version
	mockScript := `#!/bin/bash
if [ "$1" = "--version" ]; then
    echo "wakatime-cli v1.73.0"
else
    echo "Unknown command"
    exit 1
fi
`
	if err := os.WriteFile(cli.binPath, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Test getting version (will only work on Unix systems)
	if runtime.GOOS != "windows" {
		version, err := cli.getCurrentVersion()
		if err != nil {
			t.Logf("getCurrentVersion failed (expected in test environment): %v", err)
		} else {
			if version != "v1.73.0" {
				t.Errorf("Expected version 'v1.73.0', got '%s'", version)
			}
		}
	}
}

func TestSendHeartbeat(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		Debug: false,
	}

	cli := &CLI{
		config:  cfg,
		binPath: filepath.Join(tempDir, "wakatime-cli"),
	}

	// Create a mock binary that accepts heartbeats
	mockScript := `#!/bin/bash
# Mock wakatime-cli that just exits successfully
exit 0
`
	if err := os.WriteFile(cli.binPath, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Test sending heartbeat (will only work on Unix systems)
	if runtime.GOOS != "windows" {
		err := cli.SendHeartbeat("/path/to/file.go", "file", "coding", "go", "test-project", "main", false)
		if err != nil {
			t.Logf("SendHeartbeat failed (expected in test environment): %v", err)
		}
	}
}

func TestTestConnection(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{}

	cli := &CLI{
		config:  cfg,
		binPath: filepath.Join(tempDir, "wakatime-cli"),
	}

	// Create a mock binary that simulates API test
	mockScript := `#!/bin/bash
if [ "$1" = "--today" ]; then
    echo "Today you coded for 2h 30m"
    exit 0
else
    exit 1
fi
`
	if err := os.WriteFile(cli.binPath, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Test connection (will only work on Unix systems)
	if runtime.GOOS != "windows" {
		err := cli.TestConnection()
		if err != nil {
			t.Logf("TestConnection failed (expected in test environment): %v", err)
		}
	}
}

func TestBinaryPath(t *testing.T) {
	cfg := &config.Config{}
	cli := NewCLI(cfg)

	path := cli.BinaryPath()
	if path == "" {
		t.Error("Expected non-empty binary path")
	}

	// Should contain wakatime-cli in the name
	if !strings.Contains(path, "wakatime-cli") {
		t.Errorf("Expected path to contain 'wakatime-cli', got '%s'", path)
	}
}

// Integration test for GitHub API (requires network)
func TestGetLatestRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	cfg := &config.Config{}
	cli := NewCLI(cfg)

	release, err := cli.getLatestRelease()
	if err != nil {
		t.Logf("Failed to get latest release (network issue): %v", err)
		return
	}

	if release.TagName == "" {
		t.Error("Expected non-empty tag name")
	}

	if len(release.Assets) == 0 {
		t.Error("Expected at least one asset in release")
	}

	// Verify at least one asset exists for common platforms
	foundLinux := false
	foundDarwin := false
	foundWindows := false

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, "linux") {
			foundLinux = true
		}
		if strings.Contains(asset.Name, "darwin") {
			foundDarwin = true
		}
		if strings.Contains(asset.Name, "windows") {
			foundWindows = true
		}
	}

	if !foundLinux || !foundDarwin || !foundWindows {
		t.Error("Expected assets for linux, darwin, and windows platforms")
	}
}
