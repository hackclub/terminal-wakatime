package wakatime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
	"github.com/hackclub/terminal-wakatime/pkg/shell"
)

const (
	WakaTimeCLIRepo     = "wakatime/wakatime-cli"
	GitHubReleasesURL   = "https://api.github.com/repos/wakatime/wakatime-cli/releases/latest"
	CheckUpdateInterval = 24 * time.Hour
)

type CLI struct {
	config  *config.Config
	binPath string
}

type GitHubRelease struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func NewCLI(cfg *config.Config) *CLI {
	binName := fmt.Sprintf("wakatime-cli-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	binPath := filepath.Join(cfg.WakaTimeDir(), binName)

	return &CLI{
		config:  cfg,
		binPath: binPath,
	}
}

func (c *CLI) EnsureInstalled() error {
	if c.IsInstalled() {
		return c.checkForUpdates()
	}

	return c.install()
}

func (c *CLI) IsInstalled() bool {
	if _, err := os.Stat(c.binPath); os.IsNotExist(err) {
		return false
	}

	// Check if binary is executable
	return c.testBinary()
}

func (c *CLI) testBinary() bool {
	cmd := exec.Command(c.binPath, "--version")
	return cmd.Run() == nil
}

func (c *CLI) install() error {
	if err := os.MkdirAll(c.config.WakaTimeDir(), 0755); err != nil {
		return fmt.Errorf("failed to create wakatime directory: %w", err)
	}

	release, err := c.getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	asset, err := c.findAssetForPlatform(release)
	if err != nil {
		return fmt.Errorf("failed to find asset for platform: %w", err)
	}

	if err := c.downloadAndExtract(asset); err != nil {
		return fmt.Errorf("failed to download and extract: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(c.binPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Save installation timestamp
	c.saveLastUpdateCheck()

	return nil
}

func (c *CLI) checkForUpdates() error {
	lastCheck := c.getLastUpdateCheck()
	if time.Since(lastCheck) < CheckUpdateInterval {
		return nil
	}

	release, err := c.getLatestRelease()
	if err != nil {
		return nil // Silently fail on update checks
	}

	currentVersion, err := c.getCurrentVersion()
	if err != nil {
		return nil
	}

	if currentVersion != release.TagName {
		c.install() // Silently update
	}

	c.saveLastUpdateCheck()
	return nil
}

func (c *CLI) getCurrentVersion() (string, error) {
	cmd := exec.Command(c.binPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse version from output like "wakatime-cli v1.73.0"
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1], nil
	}

	return "", fmt.Errorf("unable to parse version")
}

func (c *CLI) getLatestRelease() (*GitHubRelease, error) {
	resp, err := http.Get(GitHubReleasesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch releases: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *CLI) findAssetForPlatform(release *GitHubRelease) (*Asset, error) {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, platform) {
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("no asset found for platform %s", platform)
}

func (c *CLI) downloadAndExtract(asset *Asset) error {
	resp, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download asset: %s", resp.Status)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "wakatime-cli-*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Download to temp file
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return err
	}

	// Extract based on file extension
	if strings.HasSuffix(asset.Name, ".tar.gz") {
		return c.extractTarGz(tempFile.Name())
	} else if strings.HasSuffix(asset.Name, ".zip") {
		return c.extractZip(tempFile.Name())
	}

	return fmt.Errorf("unsupported archive format")
}

func (c *CLI) extractTarGz(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.Contains(header.Name, "wakatime-cli") && header.Typeflag == tar.TypeReg {
			outFile, err := os.Create(c.binPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, tr)
			return err
		}
	}

	return fmt.Errorf("wakatime-cli binary not found in archive")
}

func (c *CLI) extractZip(archivePath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.Contains(f.Name, "wakatime-cli") && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			outFile, err := os.Create(c.binPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			return err
		}
	}

	return fmt.Errorf("wakatime-cli binary not found in archive")
}

func (c *CLI) getLastUpdateCheck() time.Time {
	timestampFile := filepath.Join(c.config.WakaTimeDir(), "last_update_check")
	data, err := os.ReadFile(timestampFile)
	if err != nil {
		return time.Time{}
	}

	timestamp, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return time.Time{}
	}

	return timestamp
}

func (c *CLI) saveLastUpdateCheck() {
	timestampFile := filepath.Join(c.config.WakaTimeDir(), "last_update_check")
	timestamp := time.Now().Format(time.RFC3339)
	os.WriteFile(timestampFile, []byte(timestamp), 0644)
}

func (c *CLI) SendHeartbeat(entity, entityType, category, language, project, branch string, isWrite bool, lines, lineNo, cursorPos, lineAdditions, lineDeletions *int) error {
	// Format plugin string according to WakaTime spec: "shell/version terminal-wakatime/version"
	pluginString := shell.FormatPluginString(config.PluginName, config.PluginVersion)

	args := []string{
		"--entity", entity,
		"--plugin", pluginString,
	}

	if entityType != "" {
		args = append(args, "--entity-type", entityType)
	}

	if category != "" {
		args = append(args, "--category", category)
	}

	if language != "" {
		args = append(args, "--language", language)
	}

	if project != "" {
		args = append(args, "--project", project)
	}

	if branch != "" {
		args = append(args, "--alternate-project", branch)
	}

	if isWrite {
		args = append(args, "--write")
	}

	if lines != nil {
		args = append(args, "--lines-in-file", fmt.Sprintf("%d", *lines))
	}

	if lineNo != nil {
		args = append(args, "--lineno", fmt.Sprintf("%d", *lineNo))
	}

	if cursorPos != nil {
		args = append(args, "--cursorpos", fmt.Sprintf("%d", *cursorPos))
	}

	if lineAdditions != nil {
		args = append(args, "--line-additions", fmt.Sprintf("%d", *lineAdditions))
	}

	if lineDeletions != nil {
		args = append(args, "--line-deletions", fmt.Sprintf("%d", *lineDeletions))
	}

	if c.config.Debug {
		args = append(args, "--verbose")
	}

	cmd := exec.Command(c.binPath, args...)

	if c.config.Debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

func (c *CLI) TestConnection() error {
	cmd := exec.Command(c.binPath, "--today")
	return cmd.Run()
}

func (c *CLI) BinaryPath() string {
	return c.binPath
}
