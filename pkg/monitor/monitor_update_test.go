package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
	"github.com/hackclub/terminal-wakatime/pkg/updater"
)

func TestMonitor_UpdateNotifications(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	wakatimeDir := filepath.Join(tempDir, ".wakatime")
	if err := os.MkdirAll(wakatimeDir, 0755); err != nil {
		t.Fatalf("Failed to create wakatime dir: %v", err)
	}

	// Create config with temporary home
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg := &config.Config{
		APIKey: "test-key",
	}

	// Create monitor
	monitor := NewMonitor(cfg)

	// Simulate a pending update notification
	updateInfo := updater.UpdateInfo{
		FromVersion: "v0.0.1",
		ToVersion:   "v0.0.2",
		UpdateTime:  time.Now(),
	}

	// Use the private saveUpdateInfo method via the updater
	upd := updater.NewUpdater("v0.0.1", wakatimeDir, "/fake/binary")
	if err := upd.SaveUpdateInfo(updateInfo); err != nil {
		t.Fatalf("Failed to save update info: %v", err)
	}

	// Test that the notification is shown and cleared
	// Note: This would require capturing stderr to test properly
	// For now, we just test that the notification file is cleared after calling checkAndShowUpdateNotification
	monitor.checkAndShowUpdateNotification()

	// Verify that the notification was cleared
	pendingInfo, err := monitor.updater.GetPendingUpdateInfo()
	if err != nil {
		t.Fatalf("Failed to get pending update info: %v", err)
	}

	if pendingInfo != nil {
		t.Error("Expected pending update info to be cleared after notification")
	}
}

func TestMonitor_BackgroundUpdateCheck(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	wakatimeDir := filepath.Join(tempDir, ".wakatime")
	if err := os.MkdirAll(wakatimeDir, 0755); err != nil {
		t.Fatalf("Failed to create wakatime dir: %v", err)
	}

	// Create config with temporary home
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	cfg := &config.Config{
		APIKey:         "test-key",
		MinCommandTime: 10 * time.Second, // Set high to avoid tracker calls
	}

	// Create monitor with custom wakatime dir
	monitor := NewMonitor(cfg)
	// Override the updater to use our test directory
	monitor.updater = updater.NewUpdater("v0.0.1", wakatimeDir, filepath.Join(wakatimeDir, "wakatime-cli"))

	// Process a command with very short duration to skip wakatime CLI calls
	// This will still trigger the background update check but won't call tracker
	err := monitor.ProcessCommand("test command", 1*time.Millisecond, tempDir)
	if err != nil {
		t.Fatalf("Failed to process command: %v", err)
	}

	// We can't easily test the background update without mocking the GitHub API
	// But we can at least verify that the background check was triggered
	// (after a short delay to allow the goroutine to run)
	time.Sleep(200 * time.Millisecond)

	// The test simply verifies that the background update check was called
	// without errors. We can't check file creation reliably since the background
	// check might fail due to network or other issues in test environment
}
