package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// GitHub API URL for releases
	ReleasesAPI = "https://api.github.com/repos/hackclub/terminal-wakatime/releases/latest"
	
	// Update check frequency (24 hours)
	UpdateCheckInterval = 24 * time.Hour
	
	// File names for update tracking
	LastCheckFile = "last_update_check"
	UpdateInfoFile = "update_info"
	TempBinaryFile = "terminal-wakatime.new"
)

type Updater struct {
	currentVersion string
	wakatimeDir    string
	binaryPath     string
}

type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	PreRelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type UpdateInfo struct {
	FromVersion string    `json:"from_version"`
	ToVersion   string    `json:"to_version"`
	UpdateTime  time.Time `json:"update_time"`
}

func NewUpdater(currentVersion, wakatimeDir, binaryPath string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		wakatimeDir:    wakatimeDir,
		binaryPath:     binaryPath,
	}
}

// ShouldCheckForUpdate returns true if it's time to check for updates
func (u *Updater) ShouldCheckForUpdate() bool {
	lastCheckFile := filepath.Join(u.wakatimeDir, LastCheckFile)
	
	data, err := os.ReadFile(lastCheckFile)
	if err != nil {
		// File doesn't exist, we should check
		return true
	}
	
	timestamp, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		// Invalid timestamp, check again
		return true
	}
	
	lastCheck := time.Unix(timestamp, 0)
	return time.Since(lastCheck) >= UpdateCheckInterval
}

// UpdateLastCheckTime records the current time as the last update check
func (u *Updater) UpdateLastCheckTime() error {
	lastCheckFile := filepath.Join(u.wakatimeDir, LastCheckFile)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	return os.WriteFile(lastCheckFile, []byte(timestamp), 0644)
}

// CheckForUpdate checks GitHub for a newer version
func (u *Updater) CheckForUpdate() (*GitHubRelease, bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	
	resp, err := client.Get(ReleasesAPI)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, fmt.Errorf("failed to decode release info: %w", err)
	}
	
	// Skip pre-releases
	if release.PreRelease {
		return nil, false, nil
	}
	
	// Compare versions
	isNewer, err := u.isVersionNewer(release.TagName)
	if err != nil {
		return nil, false, fmt.Errorf("failed to compare versions: %w", err)
	}
	
	return &release, isNewer, nil
}

// isVersionNewer compares semantic versions (simple implementation)
func (u *Updater) isVersionNewer(newVersion string) (bool, error) {
	current := strings.TrimPrefix(u.currentVersion, "v")
	new := strings.TrimPrefix(newVersion, "v")
	
	// Handle development version - always consider any release newer than "dev"
	if current == "dev" || current == "" {
		return true, nil
	}
	
	// Simple version comparison (works for semver like "0.0.4")
	currentParts := strings.Split(current, ".")
	newParts := strings.Split(new, ".")
	
	// Ensure we have at least 3 parts for comparison
	for len(currentParts) < 3 {
		currentParts = append(currentParts, "0")
	}
	for len(newParts) < 3 {
		newParts = append(newParts, "0")
	}
	
	for i := 0; i < 3; i++ {
		currentNum, err := strconv.Atoi(currentParts[i])
		if err != nil {
			return false, fmt.Errorf("invalid current version format: %s", current)
		}
		
		newNum, err := strconv.Atoi(newParts[i])
		if err != nil {
			return false, fmt.Errorf("invalid new version format: %s", new)
		}
		
		if newNum > currentNum {
			return true, nil
		} else if newNum < currentNum {
			return false, nil
		}
		// Continue to next part if equal
	}
	
	return false, nil // Versions are equal
}

// GetAssetURL returns the download URL for the current platform
func (u *Updater) GetAssetURL(release *GitHubRelease) (string, error) {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		platform += ".exe"
	}
	
	expectedName := fmt.Sprintf("terminal-wakatime-%s", platform)
	
	for _, asset := range release.Assets {
		if asset.Name == expectedName {
			return asset.BrowserDownloadURL, nil
		}
	}
	
	return "", fmt.Errorf("no asset found for platform %s", platform)
}

// DownloadUpdate downloads the new binary to a temporary location
func (u *Updater) DownloadUpdate(downloadURL string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	
	tempFile := filepath.Join(u.wakatimeDir, TempBinaryFile)
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()
	
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write update: %w", err)
	}
	
	// Make executable
	if err := os.Chmod(tempFile, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}
	
	return nil
}

// InstallUpdate atomically replaces the current binary with the new one
func (u *Updater) InstallUpdate(newVersion string) error {
	tempFile := filepath.Join(u.wakatimeDir, TempBinaryFile)
	
	// Verify temp file exists and is executable
	if _, err := os.Stat(tempFile); err != nil {
		return fmt.Errorf("temp file not found: %w", err)
	}
	
	// Atomic replace
	if err := os.Rename(tempFile, u.binaryPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	
	// Record update info for notification
	updateInfo := UpdateInfo{
		FromVersion: u.currentVersion,
		ToVersion:   newVersion,
		UpdateTime:  time.Now(),
	}
	
	return u.SaveUpdateInfo(updateInfo)
}

// SaveUpdateInfo saves update information for later notification
func (u *Updater) SaveUpdateInfo(info UpdateInfo) error {
	updateInfoFile := filepath.Join(u.wakatimeDir, UpdateInfoFile)
	
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal update info: %w", err)
	}
	
	return os.WriteFile(updateInfoFile, data, 0644)
}

// GetPendingUpdateInfo returns update info if there's a pending notification
func (u *Updater) GetPendingUpdateInfo() (*UpdateInfo, error) {
	updateInfoFile := filepath.Join(u.wakatimeDir, UpdateInfoFile)
	
	data, err := os.ReadFile(updateInfoFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No pending update
		}
		return nil, fmt.Errorf("failed to read update info: %w", err)
	}
	
	var info UpdateInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update info: %w", err)
	}
	
	return &info, nil
}

// ClearPendingUpdateInfo removes the update notification file
func (u *Updater) ClearPendingUpdateInfo() error {
	updateInfoFile := filepath.Join(u.wakatimeDir, UpdateInfoFile)
	err := os.Remove(updateInfoFile)
	if os.IsNotExist(err) {
		return nil // Already removed
	}
	return err
}

// PerformUpdateCheck checks for updates and downloads them in the background
func (u *Updater) PerformUpdateCheck() {
	// Always update the last check time first
	u.UpdateLastCheckTime()
	
	// Check for updates
	release, isNewer, err := u.CheckForUpdate()
	if err != nil || !isNewer {
		return // Silently fail or no update needed
	}
	
	// Get download URL
	downloadURL, err := u.GetAssetURL(release)
	if err != nil {
		return // Silently fail
	}
	
	// Download and install update
	if err := u.DownloadUpdate(downloadURL); err != nil {
		return // Silently fail
	}
	
	if err := u.InstallUpdate(release.TagName); err != nil {
		return // Silently fail
	}
}

// CheckAndUpdate performs a complete update check and update if needed
// This runs in the background and doesn't block the user
func (u *Updater) CheckAndUpdate() {
	if !u.ShouldCheckForUpdate() {
		return
	}
	
	// Run the actual update check in a goroutine to avoid blocking
	go u.PerformUpdateCheck()
}
