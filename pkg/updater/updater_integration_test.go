//go:build integration
// +build integration

package updater

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAutoUpdateFailureBehavior reproduces the auto-update failure scenario
// where background updates fail silently and don't create check files
func TestAutoUpdateFailureBehavior(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create a fake binary path that doesn't exist
	binaryPath := filepath.Join(tempDir, "nonexistent-binary")

	// Test with a known old version that should trigger update
	updater := NewUpdater("v1.0.0", tempDir, binaryPath)

	// Verify no check file exists initially
	checkFile := filepath.Join(tempDir, LastCheckFile)
	if _, err := os.Stat(checkFile); err == nil {
		t.Fatal("Check file should not exist initially")
	}

	// Simulate what happens in monitor.go line 51: go m.updater.CheckAndUpdate()
	// This should fail silently but may or may not create the check file
	updater.CheckAndUpdate()

	// Give the goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	// Check if a check file was created
	checkFileExists := false
	if _, err := os.Stat(checkFile); err == nil {
		checkFileExists = true
		t.Logf("Check file was created during failed auto-update")
	} else {
		t.Logf("Check file was NOT created during failed auto-update")
	}

	// Now test manual update with same setup
	release, isNewer, err := updater.CheckForUpdate()
	if err != nil {
		t.Logf("Manual update check failed: %v", err)
	} else if !isNewer {
		t.Logf("Manual update check says no newer version available")
	} else {
		t.Logf("Manual update check found version: %s", release.TagName)

		// Try to get download URL (this might fail due to architecture mismatch)
		downloadURL, err := updater.GetAssetURL(release)
		if err != nil {
			t.Logf("Getting download URL failed: %v", err)
		} else {
			t.Logf("Manual update would download from: %s", downloadURL)
		}
	}

	// The key issue: if UpdateLastCheckTime() was called during the background update
	// but the update failed, we're stuck until the next 24-hour window
	if checkFileExists {
		// Read the timestamp
		data, err := os.ReadFile(checkFile)
		if err != nil {
			t.Fatalf("Failed to read check file: %v", err)
		}
		t.Logf("Check file contains timestamp: %s", string(data))

		// Verify that ShouldCheckForUpdate now returns false
		if updater.ShouldCheckForUpdate() {
			t.Error("ShouldCheckForUpdate should return false after check file creation")
		}
	}
}

// TestAutoUpdateWithPermissionIssues tests what happens when we can't write to wakatime dir
func TestAutoUpdateWithPermissionIssues(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Make directory read-only to simulate permission issues
	err := os.Chmod(tempDir, 0444)
	if err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}

	// Restore permissions for cleanup
	defer os.Chmod(tempDir, 0755)

	binaryPath := filepath.Join(tempDir, "binary")
	updater := NewUpdater("v1.0.0", tempDir, binaryPath)

	// This should fail to write the check file
	err = updater.UpdateLastCheckTime()
	if err == nil {
		t.Error("UpdateLastCheckTime should fail with read-only directory")
	} else {
		t.Logf("UpdateLastCheckTime failed as expected: %v", err)
	}

	// Background update should handle this gracefully
	updater.CheckAndUpdate()
	time.Sleep(100 * time.Millisecond)

	// Should still return true since no check file could be written
	if !updater.ShouldCheckForUpdate() {
		t.Error("ShouldCheckForUpdate should return true when check file can't be written")
	}
}

// TestAutoUpdateBinaryNotFound tests what happens when the binary path doesn't exist
func TestAutoUpdateBinaryNotFound(t *testing.T) {
	tempDir := t.TempDir()
	nonexistentBinary := filepath.Join(tempDir, "does-not-exist")

	updater := NewUpdater("v1.0.0", tempDir, nonexistentBinary)

	// Background update with nonexistent binary
	updater.CheckAndUpdate()
	time.Sleep(100 * time.Millisecond)

	// Check if check file was created despite binary not existing
	checkFile := filepath.Join(tempDir, LastCheckFile)
	checkFileExists := false
	if _, err := os.Stat(checkFile); err == nil {
		checkFileExists = true
	}

	t.Logf("Check file created despite missing binary: %v", checkFileExists)

	// If check file was created, future updates will be blocked for 24 hours
	// even though the update couldn't possibly succeed
	if checkFileExists && !updater.ShouldCheckForUpdate() {
		t.Error("Auto-update is blocked due to premature check file creation")
	}
}

// TestManualVsAutoUpdateBehavior compares manual and auto update behavior
func TestManualVsAutoUpdateBehavior(t *testing.T) {
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "fake-binary")

	// Create a fake binary file
	err := os.WriteFile(binaryPath, []byte("fake binary"), 0755)
	if err != nil {
		t.Fatalf("Failed to create fake binary: %v", err)
	}

	updater := NewUpdater("v0.9.0", tempDir, binaryPath) // Very old version to ensure update needed

	// Test auto-update behavior
	t.Run("AutoUpdate", func(t *testing.T) {
		// Simulate background auto-update
		updater.CheckAndUpdate()
		time.Sleep(200 * time.Millisecond) // Give more time for network request

		checkFile := filepath.Join(tempDir, LastCheckFile)
		if _, err := os.Stat(checkFile); err == nil {
			t.Log("Auto-update created check file")
		} else {
			t.Log("Auto-update did NOT create check file")
		}

		// Check if update succeeded by looking for update info file
		updateInfoFile := filepath.Join(tempDir, UpdateInfoFile)
		if _, err := os.Stat(updateInfoFile); err == nil {
			t.Log("Auto-update created update info file - update likely succeeded")
		} else {
			t.Log("Auto-update did NOT create update info file - update likely failed")
		}
	})

	// Test manual update behavior for comparison
	t.Run("ManualUpdate", func(t *testing.T) {
		// Clear any existing files
		os.Remove(filepath.Join(tempDir, LastCheckFile))
		os.Remove(filepath.Join(tempDir, UpdateInfoFile))

		// Manual update process (what --force does)
		release, isNewer, err := updater.CheckForUpdate()
		t.Logf("Manual check: newer=%v, err=%v", isNewer, err)

		if err == nil && isNewer {
			downloadURL, err := updater.GetAssetURL(release)
			t.Logf("Manual download URL: err=%v", err)

			if err == nil {
				err = updater.DownloadUpdate(downloadURL)
				t.Logf("Manual download: err=%v", err)

				if err == nil {
					err = updater.InstallUpdate(release.TagName)
					t.Logf("Manual install: err=%v", err)
				}
			}
		}

		// Update check time only after successful completion (this is what manual update does)
		updater.UpdateLastCheckTime()
	})
}
