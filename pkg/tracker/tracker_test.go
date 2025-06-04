package tracker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
)

func TestNewTracker(t *testing.T) {
	cfg := &config.Config{}
	tracker := NewTracker(cfg)

	if tracker == nil {
		t.Error("Expected tracker to be created")
	}

	if tracker.config != cfg {
		t.Error("Expected tracker to store config reference")
	}
}

func TestParseCommandToSingleActivity(t *testing.T) {
	cfg := &config.Config{
		Project: "test-project",
	}
	tracker := NewTracker(cfg)

	tests := []struct {
		name       string
		command    string
		workingDir string
		expected   bool // whether activity should be created
	}{
		{
			name:       "vim with file",
			command:    "vim test.go",
			workingDir: "/tmp",
			expected:   true,
		},
		{
			name:       "git command",
			command:    "git status",
			workingDir: "/tmp",
			expected:   true,
		},
		{
			name:       "cd command",
			command:    "cd /home/user",
			workingDir: "/tmp",
			expected:   true,
		},
		{
			name:       "node command",
			command:    "node app.js",
			workingDir: "/tmp",
			expected:   true,
		},
		{
			name:       "ssh command",
			command:    "ssh user@example.com",
			workingDir: "/tmp",
			expected:   true,
		},
		{
			name:       "ls command",
			command:    "ls -la",
			workingDir: "/tmp",
			expected:   true, // now all commands create activities
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activity := tracker.parseCommandToSingleActivity(tt.command, tt.workingDir)
			if tt.expected && activity == nil {
				t.Errorf("Expected activity to be created for command '%s', got nil", tt.command)
			}
			if !tt.expected && activity != nil {
				t.Errorf("Expected no activity for command '%s', got %+v", tt.command, activity)
			}
		})
	}
}

func TestIsEditor(t *testing.T) {
	cfg := &config.Config{}
	tracker := NewTracker(cfg)

	tests := []struct {
		cmdName  string
		expected bool
	}{
		{"vim", true},
		{"vi", true},
		{"nvim", true},
		{"emacs", true},
		{"nano", true},
		{"code", true},
		{"subl", true},
		{"atom", true},
		{"ls", false},
		{"grep", false},
		{"cat", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmdName, func(t *testing.T) {
			result := tracker.isEditor(tt.cmdName)
			if result != tt.expected {
				t.Errorf("Expected isEditor('%s') to be %t, got %t", tt.cmdName, tt.expected, result)
			}
		})
	}
}

func TestHandleEditorCommand(t *testing.T) {
	cfg := &config.Config{}
	tracker := NewTracker(cfg)

	// Create a temporary file to test with
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name       string
		fields     []string
		workingDir string
		expected   int
	}{
		{
			name:       "vim with existing file",
			fields:     []string{"vim", testFile},
			workingDir: tempDir,
			expected:   1,
		},
		{
			name:       "vim with non-existing file",
			fields:     []string{"vim", "nonexistent.go"},
			workingDir: tempDir,
			expected:   1, // Still tracks the app
		},
		{
			name:       "vim with flags",
			fields:     []string{"vim", "-n", testFile},
			workingDir: tempDir,
			expected:   1,
		},
		{
			name:       "vim without arguments",
			fields:     []string{"vim"},
			workingDir: tempDir,
			expected:   1, // Tracks vim app
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activities := tracker.handleEditorCommand(tt.fields, tt.workingDir)
			if len(activities) != tt.expected {
				t.Errorf("Expected %d activities, got %d", tt.expected, len(activities))
			}

			if len(activities) > 0 {
				activity := activities[0]
				if activity.Category != "coding" {
					t.Errorf("Expected category 'coding', got '%s'", activity.Category)
				}
			}
		})
	}
}

func TestParseRemoteConnection(t *testing.T) {
	cfg := &config.Config{}
	tracker := NewTracker(cfg)

	tests := []struct {
		command  string
		expected string
	}{
		{"ssh user@example.com", "example.com"},
		{"ssh example.com", "example.com"},
		{"mysql -h db.example.com -u user", "db.example.com"},
		{"psql -h localhost -d mydb", "localhost"},
		{"redis-cli -h redis.example.com", "redis.example.com"},
		{"ls -la", ""}, // not a remote command
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := tracker.parseRemoteConnection(tt.command)
			if result != tt.expected {
				t.Errorf("Expected domain '%s', got '%s' for command '%s'", tt.expected, result, tt.command)
			}
		})
	}
}

func TestDetectProject(t *testing.T) {
	cfg := &config.Config{}
	tracker := NewTracker(cfg)

	// Test with explicit project in config
	cfg.Project = "explicit-project"
	result := tracker.detectProject("/any/path")
	if result != "explicit-project" {
		t.Errorf("Expected 'explicit-project', got '%s'", result)
	}

	// Test project detection from directory structure
	cfg.Project = ""
	tempDir := t.TempDir()

	// Create a project structure
	projectDir := filepath.Join(tempDir, "my-project")
	os.MkdirAll(projectDir, 0755)

	// Create a go.mod file
	goModFile := filepath.Join(projectDir, "go.mod")
	os.WriteFile(goModFile, []byte("module my-project"), 0644)

	// Test project detection
	testFile := filepath.Join(projectDir, "main.go")
	result = tracker.detectProject(testFile)
	if result != "my-project" {
		t.Errorf("Expected 'my-project', got '%s'", result)
	}
}

func TestTrackFile(t *testing.T) {
	// This test requires mocking the wakatime CLI
	// For now, we'll test the basic logic without actually sending heartbeats
	cfg := &config.Config{}
	tracker := NewTracker(cfg)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	// This would normally send a heartbeat, but we can't test that without mocking
	// We'll just ensure the method doesn't panic and accepts the parameters
	err := tracker.TrackFile(testFile, false)

	// We expect an error because wakatime-cli is not installed in test environment
	// But we want to make sure the method processes the parameters correctly
	if err == nil {
		t.Log("TrackFile succeeded (wakatime-cli must be available)")
	} else {
		// This is expected in test environment
		if !strings.Contains(err.Error(), "wakatime-cli") && !strings.Contains(err.Error(), "executable") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestShowEditorSuggestion(t *testing.T) {
	cfg := &config.Config{
		DisableEditorSuggestions:  false,
		EditorSuggestionFrequency: time.Millisecond, // Very short for testing
	}
	tracker := NewTracker(cfg)

	// Test that suggestion is not shown when disabled
	cfg.DisableEditorSuggestions = true
	tracker.showEditorSuggestion("vim")
	// No way to verify output was suppressed without capturing stderr

	// Test normal suggestion flow
	cfg.DisableEditorSuggestions = false
	tracker.showEditorSuggestion("vim")

	// Second call should be rate-limited (but with millisecond frequency, it won't be)
	time.Sleep(2 * time.Millisecond)
	tracker.showEditorSuggestion("vim")
}

func TestGetEditorSuggestion(t *testing.T) {
	cfg := &config.Config{}
	tracker := NewTracker(cfg)

	tests := []struct {
		editor  string
		hasText bool
	}{
		{"vim", true},
		{"emacs", true},
		{"code", true},
		{"nano", true},
		{"sublime", true},
		{"atom", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.editor, func(t *testing.T) {
			suggestion := tracker.getEditorSuggestion(tt.editor)
			if tt.hasText && suggestion == "" {
				t.Errorf("Expected suggestion for editor '%s', got empty string", tt.editor)
			}
			if !tt.hasText && suggestion != "" {
				t.Errorf("Expected no suggestion for editor '%s', got '%s'", tt.editor, suggestion)
			}
		})
	}
}

func TestShouldSendHeartbeat(t *testing.T) {
	cfg := &config.Config{}
	tracker := NewTracker(cfg)

	// Test write events (always send)
	writeActivity := &Activity{
		Entity:  "/test/file.go",
		IsWrite: true,
	}
	if !tracker.shouldSendHeartbeat(writeActivity) {
		t.Error("Expected to send heartbeat for write event")
	}

	// Test file change (always send)
	tracker.lastSentFile = "/previous/file.go"
	fileChangeActivity := &Activity{
		Entity:  "/test/file.go",
		IsWrite: false,
	}
	if !tracker.shouldSendHeartbeat(fileChangeActivity) {
		t.Error("Expected to send heartbeat when file changes")
	}

	// Test same file within 2 minutes (should NOT send)
	tracker.lastSentFile = "/test/file.go"
	tracker.lastSentTime = time.Now().Add(-1 * time.Minute) // 1 minute ago
	sameFileActivity := &Activity{
		Entity:  "/test/file.go",
		IsWrite: false,
	}
	if tracker.shouldSendHeartbeat(sameFileActivity) {
		t.Error("Expected NOT to send heartbeat for same file within 2 minutes")
	}

	// Test same file after 2 minutes (should send)
	tracker.lastSentTime = time.Now().Add(-3 * time.Minute) // 3 minutes ago
	if !tracker.shouldSendHeartbeat(sameFileActivity) {
		t.Error("Expected to send heartbeat for same file after 2 minutes")
	}

	// Test first heartbeat (no previous state)
	newTracker := NewTracker(cfg)
	firstActivity := &Activity{
		Entity:  "/test/file.go",
		IsWrite: false,
	}
	if !newTracker.shouldSendHeartbeat(firstActivity) {
		t.Error("Expected to send first heartbeat")
	}
}
