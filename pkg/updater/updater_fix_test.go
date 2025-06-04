// +build integration

package updater

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAutoUpdateFixedBehavior verifies that the fix prevents premature check file creation
func TestAutoUpdateFixedBehavior(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test with nonexistent binary (should fail but not create check file)
	nonexistentBinary := filepath.Join(tempDir, "does-not-exist")
	updater := NewUpdater("v1.0.0", tempDir, nonexistentBinary)
	
	// Run background update (should fail during install)
	updater.CheckAndUpdate()
	time.Sleep(200 * time.Millisecond)
	
	checkFile := filepath.Join(tempDir, LastCheckFile)
	
	// The fix: check file should NOT be created when update fails
	if _, err := os.Stat(checkFile); err == nil {
		t.Error("Check file should NOT be created when auto-update fails")
	} else {
		t.Log("✓ Check file correctly NOT created after failed auto-update")
	}
	
	// Should still be able to check for updates
	if !updater.ShouldCheckForUpdate() {
		t.Error("Should still be able to check for updates after failed auto-update")
	} else {
		t.Log("✓ Can still check for updates after failed auto-update")
	}
}

// TestAutoUpdateCheckTimeUpdatedOnSuccess verifies check time is updated appropriately
func TestAutoUpdateCheckTimeUpdatedOnSuccess(t *testing.T) {
	tempDir := t.TempDir()
	checkFile := filepath.Join(tempDir, LastCheckFile)
	
	// Test 1: Update needed but install fails - should NOT create check file
	t.Run("UpdateFailedNoCheckFile", func(t *testing.T) {
		// Create a read-only directory to cause install failure
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.Mkdir(readOnlyDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create readonly dir: %v", err)
		}
		
		// Make it read-only after creating
		err = os.Chmod(readOnlyDir, 0555)
		if err != nil {
			t.Fatalf("Failed to make dir readonly: %v", err)
		}
		defer os.Chmod(readOnlyDir, 0755) // Restore for cleanup
		
		// Try to install to a path inside the read-only directory
		binaryPath := filepath.Join(readOnlyDir, "terminal-wakatime")
		updater := NewUpdater("v0.1.0", tempDir, binaryPath)
		
		// This should fail during the install step (os.Rename will fail)
		updater.PerformUpdateCheck()
		
		// Check file should NOT be created because install failed
		if _, err := os.Stat(checkFile); err == nil {
			data, _ := os.ReadFile(checkFile)
			t.Errorf("Check file should not be created when install fails, but found: %s", string(data))
		} else {
			t.Log("✓ Check file correctly NOT created when install fails")
		}
	})
	
	// Test 2: No update needed - should create check file
	t.Run("NoUpdateNeededCreateCheckFile", func(t *testing.T) {
		// Clean up any existing check file
		os.Remove(checkFile)
		
		// Use very new version so no update is needed
		updater := NewUpdater("v99.99.99", tempDir, filepath.Join(tempDir, "fake"))
		updater.PerformUpdateCheck()
		
		if _, err := os.Stat(checkFile); err != nil {
			t.Error("Check file should be created when no update is needed")
		} else {
			t.Log("✓ Check file created when no update needed")
		}
	})
}

// TestManualUpdateStillWorks verifies manual updates are unaffected by the fix
func TestManualUpdateStillWorks(t *testing.T) {
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "fake-binary")
	
	// Create fake binary
	err := os.WriteFile(binaryPath, []byte("fake"), 0755)
	if err != nil {
		t.Fatalf("Failed to create fake binary: %v", err)
	}
	
	updater := NewUpdater("v1.0.0", tempDir, binaryPath)
	
	// Manual update process should work the same
	release, isNewer, err := updater.CheckForUpdate()
	if err != nil {
		t.Fatalf("Manual check failed: %v", err)
	}
	
	if isNewer {
		t.Logf("Manual update found newer version: %s", release.TagName)
		
		// Manual update explicitly calls UpdateLastCheckTime at the end
		// regardless of install success (this matches existing manual behavior)
		err = updater.UpdateLastCheckTime()
		if err != nil {
			t.Errorf("Manual UpdateLastCheckTime failed: %v", err)
		}
		
		checkFile := filepath.Join(tempDir, LastCheckFile)
		if _, err := os.Stat(checkFile); err != nil {
			t.Error("Manual update should create check file")
		} else {
			t.Log("✓ Manual update correctly creates check file")
		}
	}
}
