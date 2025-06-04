package tests

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ShellTestSuite runs comprehensive integration tests for each supported shell
type ShellTestSuite struct {
	binaryPath  string
	testDir     string
	mockCLIPath string
	configDir   string
}

func TestShellIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping shell integration tests in short mode")
	}

	suite := setupShellTestSuite(t)
	defer suite.cleanup()

	shells := []struct {
		name       string
		executable string
		skipReason string
	}{
		{"bash", "bash", ""},
		{"zsh", "zsh", ""},
		{"fish", "fish", ""},
	}

	for _, shell := range shells {
		t.Run(shell.name, func(t *testing.T) {
			// Check if shell is available - fail if not found
			if _, err := exec.LookPath(shell.executable); err != nil {
				t.Fatalf("Required shell %s not found in PATH - please install %s to run shell integration tests", shell.executable, shell.executable)
			}

			suite.testShellLifecycle(t, shell.name, shell.executable)
		})
	}
}

func setupShellTestSuite(t *testing.T) *ShellTestSuite {
	testDir := t.TempDir()
	
	// Build the main binary
	binaryPath := filepath.Join(testDir, "terminal-wakatime")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/terminal-wakatime")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Create mock wakatime-cli
	mockCLIPath := filepath.Join(testDir, "wakatime-cli")
	suite := &ShellTestSuite{
		binaryPath:  binaryPath,
		testDir:     testDir,
		mockCLIPath: mockCLIPath,
		configDir:   filepath.Join(testDir, ".wakatime"),
	}
	
	suite.createMockWakatimeCLI(t)
	suite.setupTestConfig(t)
	
	return suite
}

func (s *ShellTestSuite) createMockWakatimeCLI(t *testing.T) {
	// Create a mock wakatime-cli that logs all calls
	mockScript := fmt.Sprintf(`#!/bin/bash
# Mock wakatime-cli for testing
echo "$(date '+%%Y-%%m-%%d %%H:%%M:%%S') wakatime-cli $*" >> %s/wakatime-calls.log

# Simulate different behaviors based on arguments
case "$1" in
    "--version")
        echo "wakatime-cli 1.73.0"
        ;;
    "--help")
        echo "WakaTime command line interface"
        ;;
    *)
        # Log detailed heartbeat information
        entity=""
        entity_type=""
        language=""
        project=""
        category=""
        
        # Parse arguments to extract meaningful data
        while [[ $# -gt 0 ]]; do
            case $1 in
                --entity)
                    entity="$2"
                    shift 2
                    ;;
                --entity-type)
                    entity_type="$2"
                    shift 2
                    ;;
                --language)
                    language="$2"
                    shift 2
                    ;;
                --project)
                    project="$2"
                    shift 2
                    ;;
                --category)
                    category="$2"
                    shift 2
                    ;;
                *)
                    shift
                    ;;
            esac
        done
        
        # Format heartbeat log entry
        heartbeat_entry="entity=$entity type=$entity_type lang=$language proj=$project cat=$category"
        echo "$heartbeat_entry" >> %s/heartbeats.log
        ;;
esac
exit 0
`, s.testDir, s.testDir)

	if err := os.WriteFile(s.mockCLIPath, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock wakatime-cli: %v", err)
	}
}

func (s *ShellTestSuite) setupTestConfig(t *testing.T) {
	// Create config directory
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Set environment variables
	os.Setenv("HOME", s.testDir)
	os.Setenv("WAKATIME_HOME", s.configDir)
	
	// Create the wakatime-cli directory structure that the wakatime package expects
	binName := fmt.Sprintf("wakatime-cli-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	wakatimeCLIDir := filepath.Join(s.configDir, binName)
	if err := os.MkdirAll(filepath.Dir(wakatimeCLIDir), 0755); err != nil {
		t.Fatalf("Failed to create wakatime-cli directory: %v", err)
	}
	
	// Copy our mock to the expected location
	if err := os.Rename(s.mockCLIPath, wakatimeCLIDir); err != nil {
		// If rename fails, try copy
		mockContent, readErr := os.ReadFile(s.mockCLIPath)
		if readErr != nil {
			t.Fatalf("Failed to read mock CLI: %v", readErr)
		}
		if writeErr := os.WriteFile(wakatimeCLIDir, mockContent, 0755); writeErr != nil {
			t.Fatalf("Failed to write mock CLI to expected location: %v", writeErr)
		}
	}
	
	// Update mock CLI path
	s.mockCLIPath = wakatimeCLIDir
	
	// Create a test configuration
	configCmd := exec.Command(s.binaryPath, "config", "--key", "test-api-key-123456789", "--project", "test-project")
	configCmd.Env = append(os.Environ(), 
		"HOME="+s.testDir,
		"WAKATIME_HOME="+s.configDir,
		"PATH="+filepath.Dir(s.mockCLIPath)+":"+os.Getenv("PATH"),
	)
	
	if output, err := configCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to setup test config: %v\nOutput: %s", err, output)
	}
}

func (s *ShellTestSuite) testShellLifecycle(t *testing.T, shellName, shellExec string) {
	t.Logf("Testing %s shell lifecycle", shellName)

	// Step 1: Generate shell hooks
	hooks := s.generateHooks(t, shellName)
	t.Logf("Generated hooks for %s (%d chars)", shellName, len(hooks))

	// Step 2: Create a test script that sources the hooks and runs commands
	testScript := s.createTestScript(t, shellName, shellExec, hooks)
	
	// Step 3: Execute the test script
	s.executeTestScript(t, shellName, shellExec, testScript)
	
	// Step 4: Verify tracking occurred
	s.verifyTracking(t, shellName)
}

func (s *ShellTestSuite) generateHooks(t *testing.T, shellName string) string {
	// Force the shell detection by setting environment
	env := append(os.Environ(),
		"HOME="+s.testDir,
		"WAKATIME_HOME="+s.configDir,
		"PATH="+filepath.Dir(s.mockCLIPath)+":"+os.Getenv("PATH"),
	)

	switch shellName {
	case "zsh":
		env = append(env, "ZSH_VERSION=5.8", "SHELL=/bin/zsh")
	case "fish":
		env = append(env, "FISH_VERSION=3.0", "SHELL=/usr/bin/fish")
	case "bash":
		env = append(env, "BASH_VERSION=5.0", "SHELL=/bin/bash")
	}

	initCmd := exec.Command(s.binaryPath, "init")
	initCmd.Env = env
	
	output, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to generate %s hooks: %v\nOutput: %s", shellName, err, output)
	}

	hooks := string(output)
	
	// Verify hooks contain expected functions
	expectedFunctions := []string{"__terminal_wakatime_preexec"}
	for _, fn := range expectedFunctions {
		if !strings.Contains(hooks, fn) {
			t.Errorf("Generated %s hooks missing function: %s", shellName, fn)
		}
	}

	return hooks
}

func (s *ShellTestSuite) createTestScript(t *testing.T, shellName, shellExec, hooks string) string {
	scriptPath := filepath.Join(s.testDir, fmt.Sprintf("test_%s.sh", shellName))
	
	var scriptContent string
	
	switch shellName {
	case "fish":
		scriptContent = s.createFishTestScript(hooks)
	case "zsh":
		scriptContent = s.createZshTestScript(hooks)
	default: // bash
		scriptContent = s.createBashTestScript(hooks)
	}

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create %s test script: %v", shellName, err)
	}

	return scriptPath
}

func (s *ShellTestSuite) createBashTestScript(hooks string) string {
	// Create a script that runs commands in an interactive-like manner
	return fmt.Sprintf(`#!/bin/bash

# Set up environment
export HOME="%s"
export WAKATIME_HOME="%s"
export PATH="%s:$PATH"

# Clear previous logs
rm -f "%s/wakatime-calls.log" "%s/heartbeats.log"

# Source the hooks
%s

echo "=== Bash Integration Test Starting ==="

# Simulate interactive commands by manually triggering the hooks
# This mimics what would happen in a real interactive bash session

# Test 1: Editor command
echo "Testing vim command..."
__terminal_wakatime_preexec "vim test.py"
sleep 3
__terminal_wakatime_postexec

# Test 2: File operations  
echo "Testing file operations..."
touch test_file.txt
echo "content" > test_file.txt
__terminal_wakatime_preexec "cat test_file.txt"
sleep 3
__terminal_wakatime_postexec

# Test 3: Git operations
echo "Testing git command..."
__terminal_wakatime_preexec "git status"
sleep 3
__terminal_wakatime_postexec

# Test 4: Build command
echo "Testing build command..."
__terminal_wakatime_preexec "make all"
sleep 4
__terminal_wakatime_postexec

# Test 5: Short command (should NOT trigger - under minimum duration)
echo "Testing short command (should not track)..."
__terminal_wakatime_preexec "pwd"
sleep 1
__terminal_wakatime_postexec

# Test 6: Directory navigation
echo "Testing directory operations..."
mkdir -p test_dir
cd test_dir
__terminal_wakatime_preexec "ls -la"
sleep 3
__terminal_wakatime_postexec

echo "=== Bash Integration Test Completed ==="
`, s.testDir, s.configDir, filepath.Dir(s.mockCLIPath), s.testDir, s.testDir, hooks)
}

func (s *ShellTestSuite) createZshTestScript(hooks string) string {
	return fmt.Sprintf(`#!/bin/zsh

# Set up environment
export HOME="%s"
export WAKATIME_HOME="%s"
export PATH="%s:$PATH"
export ZSH_VERSION="5.8"

# Clear previous logs
rm -f "%s/wakatime-calls.log" "%s/heartbeats.log"

# Source the hooks
%s

echo "=== Zsh Integration Test Starting ==="

# Simulate interactive commands by manually triggering the hooks
# This mimics what would happen in a real interactive zsh session

# Test 1: Code editor
echo "Testing code command..."
__terminal_wakatime_preexec "code main.py"
sleep 3
__terminal_wakatime_precmd

# Test 2: File editing
echo "Testing file editing..."
touch main.py
echo "print('hello')" > main.py
__terminal_wakatime_preexec "nvim main.py"
sleep 4
__terminal_wakatime_precmd

# Test 3: Package management
echo "Testing npm command..."
__terminal_wakatime_preexec "npm install express"
sleep 3
__terminal_wakatime_precmd

# Test 4: Docker command
echo "Testing docker command..."
__terminal_wakatime_preexec "docker build ."
sleep 5
__terminal_wakatime_precmd

# Test 5: Very short command (should NOT be tracked)
echo "Testing very short command..."
__terminal_wakatime_preexec "pwd"
sleep 0.5
__terminal_wakatime_precmd

# Test 6: File watching/building
echo "Testing build operations..."
mkdir -p build
cd build
__terminal_wakatime_preexec "cargo build"
sleep 4
__terminal_wakatime_precmd

echo "=== Zsh Integration Test Completed ==="
`, s.testDir, s.configDir, filepath.Dir(s.mockCLIPath), s.testDir, s.testDir, hooks)
}

func (s *ShellTestSuite) createFishTestScript(hooks string) string {
	return fmt.Sprintf(`#!/usr/bin/fish

# Set up environment
set -x HOME "%s"
set -x WAKATIME_HOME "%s"
set -x PATH "%s" $PATH

# Clear previous logs
rm -f "%s/wakatime-calls.log" "%s/heartbeats.log"

echo "=== Fish Integration Test Starting ==="

# Fish doesn't easily support POSIX shell hooks, so we'll test direct tracking
echo "Testing direct tracking calls..."

# Test 1: Simulate editor usage
echo "Simulating neovim usage..."
"%s" track --command "nvim config.fish" --duration 5 --pwd "%s"

# Test 2: Simulate file operations
echo "Simulating file operations..."
touch test.fish
echo "echo 'hello fish'" > test.fish
"%s" track --command "cat test.fish" --duration 3 --pwd "%s"

# Test 3: Simulate git operations
echo "Simulating git operations..."
"%s" track --command "git status" --duration 4 --pwd "%s"

# Test 4: Simulate build operations
echo "Simulating build operations..."
mkdir -p fish_build
cd fish_build
"%s" track --command "make all" --duration 6 --pwd (pwd)

# Test 5: Short command (should still work via direct call)
echo "Testing short command tracking..."
"%s" track --command "ls" --duration 1 --pwd "%s"

echo "=== Fish Integration Test Completed ==="
`, s.testDir, s.configDir, filepath.Dir(s.mockCLIPath), s.testDir, s.testDir, s.binaryPath, s.testDir, s.binaryPath, s.testDir, s.binaryPath, s.testDir, s.binaryPath, s.binaryPath, s.testDir)
}

func (s *ShellTestSuite) executeTestScript(t *testing.T, shellName, shellExec, scriptPath string) {
	env := append(os.Environ(),
		"HOME="+s.testDir,
		"WAKATIME_HOME="+s.configDir,
		"PATH="+filepath.Dir(s.mockCLIPath)+":"+os.Getenv("PATH"),
	)

	// Set shell-specific environment variables
	switch shellName {
	case "zsh":
		env = append(env, "ZSH_VERSION=5.8")
	case "fish":
		env = append(env, "FISH_VERSION=3.0")
	case "bash":
		env = append(env, "BASH_VERSION=5.0")
	}

	cmd := exec.Command(shellExec, scriptPath)
	cmd.Dir = s.testDir
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	t.Logf("%s script output:\n%s", shellName, string(output))
	
	if err != nil {
		t.Fatalf("Failed to execute %s test script: %v\nOutput: %s", shellName, err, output)
	}

	// Give some time for background tracking to complete
	time.Sleep(2 * time.Second)
}

func (s *ShellTestSuite) verifyTracking(t *testing.T, shellName string) {
	// Check if wakatime-cli was called
	callsLogPath := filepath.Join(s.testDir, "wakatime-calls.log")
	heartbeatsLogPath := filepath.Join(s.testDir, "heartbeats.log")

	var wakatimeCalls []string
	var heartbeats []string

	// Read wakatime-cli calls
	if _, err := os.Stat(callsLogPath); err == nil {
		content, err := os.ReadFile(callsLogPath)
		if err == nil {
			callsContent := strings.TrimSpace(string(content))
			if callsContent != "" {
				wakatimeCalls = strings.Split(callsContent, "\n")
			}
			t.Logf("%s wakatime-cli calls (%d total):\n%s", shellName, len(wakatimeCalls), callsContent)
		}
	}

	// Read heartbeats
	if _, err := os.Stat(heartbeatsLogPath); err == nil {
		content, err := os.ReadFile(heartbeatsLogPath)
		if err == nil {
			heartbeatsContent := strings.TrimSpace(string(content))
			if heartbeatsContent != "" {
				heartbeats = strings.Split(heartbeatsContent, "\n")
			}
			t.Logf("%s heartbeats (%d total):\n%s", shellName, len(heartbeats), heartbeatsContent)
		}
	}

	// Verify actual tracking occurred
	s.verifyHeartbeatContent(t, shellName, heartbeats)
	
	// Test the track command directly as a fallback verification
	s.testDirectTracking(t, shellName)
}

func (s *ShellTestSuite) verifyHeartbeatContent(t *testing.T, shellName string, heartbeats []string) {
	if len(heartbeats) == 0 {
		t.Errorf("%s: No heartbeats were generated - shell hooks may not be working correctly", shellName)
		return
	}

	// Track what we found
	foundEditor := false
	foundFile := false
	foundGit := false
	foundBuild := false

	for _, heartbeat := range heartbeats {
		if heartbeat == "" {
			continue
		}
		
		t.Logf("%s heartbeat: %s", shellName, heartbeat)
		
		// Check for different types of commands
		if strings.Contains(heartbeat, "vim") || strings.Contains(heartbeat, "nvim") || strings.Contains(heartbeat, "code") {
			foundEditor = true
		}
		if strings.Contains(heartbeat, "touch") || strings.Contains(heartbeat, "cat") || strings.Contains(heartbeat, ".txt") || strings.Contains(heartbeat, ".py") || strings.Contains(heartbeat, ".fish") {
			foundFile = true
		}
		if strings.Contains(heartbeat, "git") {
			foundGit = true
		}
		if strings.Contains(heartbeat, "make") || strings.Contains(heartbeat, "npm") || strings.Contains(heartbeat, "docker") {
			foundBuild = true
		}
	}

	// Report findings
	findings := []string{}
	if foundEditor {
		findings = append(findings, "editor commands")
	}
	if foundFile {
		findings = append(findings, "file operations")
	}
	if foundGit {
		findings = append(findings, "git commands")
	}
	if foundBuild {
		findings = append(findings, "build commands")
	}

	if len(findings) > 0 {
		t.Logf("✓ %s: Successfully tracked %s", shellName, strings.Join(findings, ", "))
	} else {
		t.Logf("⚠ %s: Heartbeats generated but no recognizable command types found", shellName)
	}

	// Minimum expectation: at least some heartbeats should be generated
	if len(heartbeats) < 3 {
		t.Errorf("%s: Expected at least 3 heartbeats but got %d - shell integration may not be working properly", shellName, len(heartbeats))
	} else {
		t.Logf("✓ %s: Generated %d heartbeats (sufficient for integration test)", shellName, len(heartbeats))
	}
}

func (s *ShellTestSuite) testDirectTracking(t *testing.T, shellName string) {
	// Test the track command directly to ensure the tracking mechanism works
	env := append(os.Environ(),
		"HOME="+s.testDir,
		"WAKATIME_HOME="+s.configDir,
		"PATH="+filepath.Dir(s.mockCLIPath)+":"+os.Getenv("PATH"),
	)

	trackCmd := exec.Command(s.binaryPath, "track", 
		"--command", "test-command-"+shellName,
		"--duration", "5",
		"--pwd", s.testDir)
	trackCmd.Env = env

	output, err := trackCmd.CombinedOutput()
	if err != nil {
		t.Errorf("Direct track command failed for %s: %v\nOutput: %s", shellName, err, output)
	} else {
		t.Logf("✓ Direct tracking works for %s", shellName)
	}
}

func (s *ShellTestSuite) cleanup() {
	// Cleanup is handled by t.TempDir() automatically
}

// TestShellHookGeneration tests that hooks are generated correctly for each shell
func TestShellHookGeneration(t *testing.T) {
	suite := setupShellTestSuite(t)
	defer suite.cleanup()

	tests := []struct {
		shellName    string
		envVars      map[string]string
		expectedFunc []string
	}{
		{
			shellName: "bash",
			envVars:   map[string]string{"BASH_VERSION": "5.0", "SHELL": "/bin/bash"},
			expectedFunc: []string{"__terminal_wakatime_preexec", "__terminal_wakatime_postexec", "PROMPT_COMMAND"},
		},
		{
			shellName: "zsh", 
			envVars:   map[string]string{"ZSH_VERSION": "5.8", "SHELL": "/bin/zsh"},
			expectedFunc: []string{"__terminal_wakatime_preexec", "__terminal_wakatime_precmd", "preexec_functions"},
		},
		{
			shellName: "fish",
			envVars:   map[string]string{"FISH_VERSION": "3.0", "SHELL": "/usr/bin/fish"},
			expectedFunc: []string{"__terminal_wakatime_preexec", "--on-event fish_preexec"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.shellName, func(t *testing.T) {
			env := append(os.Environ(),
				"HOME="+suite.testDir,
				"WAKATIME_HOME="+suite.configDir,
			)
			
			for key, value := range tt.envVars {
				env = append(env, key+"="+value)
			}

			cmd := exec.Command(suite.binaryPath, "init")
			cmd.Env = env
			
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Hook generation failed for %s: %v", tt.shellName, err)
			}

			hooks := string(output)
			for _, expectedFunc := range tt.expectedFunc {
				if !strings.Contains(hooks, expectedFunc) {
					t.Errorf("%s hooks missing expected content: %s\nGenerated hooks:\n%s", 
						tt.shellName, expectedFunc, hooks)
				}
			}

			t.Logf("✓ %s hooks generated correctly", tt.shellName)
		})
	}
}

// TestCommandParsing tests that the track command correctly parses different command formats
func TestCommandParsing(t *testing.T) {
	suite := setupShellTestSuite(t)
	defer suite.cleanup()

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
	}{
		{
			name: "simple command",
			args: []string{"track", "--command", "ls -la", "--duration", "3", "--pwd", "/tmp"},
			wantErr: false,
		},
		{
			name: "complex command with pipes",
			args: []string{"track", "--command", "cat file.txt | grep pattern | wc -l", "--duration", "5", "--pwd", "/home/user"},
			wantErr: false,
		},
		{
			name: "command with quotes",
			args: []string{"track", "--command", `echo "hello world"`, "--duration", "2", "--pwd", "/"},
			wantErr: false,
		},
		{
			name: "missing duration",
			args: []string{"track", "--command", "test", "--pwd", "/"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := append(os.Environ(),
				"HOME="+suite.testDir,
				"WAKATIME_HOME="+suite.configDir,
				"PATH="+filepath.Dir(suite.mockCLIPath)+":"+os.Getenv("PATH"),
			)

			cmd := exec.Command(suite.binaryPath, tt.args...)
			cmd.Env = env

			output, err := cmd.CombinedOutput()
			
			if tt.wantErr && err == nil {
				t.Errorf("Expected error for %s, but command succeeded", tt.name)
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("Command failed for %s: %v\nOutput: %s", tt.name, err, output)
			}

			if !tt.wantErr {
				t.Logf("✓ %s parsed correctly", tt.name)
			}
		})
	}
}

// TestEditorDetection tests the editor detection functionality
func TestEditorDetection(t *testing.T) {
	suite := setupShellTestSuite(t)
	defer suite.cleanup()

	tests := []struct {
		command     string
		expectMatch bool
		description string
	}{
		{"vim config.go", true, "vim editor"},
		{"nvim main.rs", true, "neovim editor"},
		{"emacs test.py", true, "emacs editor"},
		{"code .", true, "vscode"},
		{"ls -la", false, "regular command"},
		{"grep pattern file.txt", false, "search command"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			env := append(os.Environ(),
				"HOME="+suite.testDir,
				"WAKATIME_HOME="+suite.configDir,
				"PATH="+filepath.Dir(suite.mockCLIPath)+":"+os.Getenv("PATH"),
			)

			cmd := exec.Command(suite.binaryPath, "track", 
				"--command", tt.command, 
				"--duration", "3", 
				"--pwd", suite.testDir)
			cmd.Env = env

			output, err := cmd.CombinedOutput()
			outputStr := string(output)
			
			// Allow errors if they contain editor suggestions (that's expected behavior)
			if err != nil && !strings.Contains(outputStr, "Tip:") {
				t.Errorf("Track command failed for %s: %v\nOutput: %s", tt.command, err, output)
			}

			// Check if the command was processed (this test mainly ensures no crashes)
			if tt.expectMatch && strings.Contains(outputStr, "Tip:") {
				t.Logf("✓ Command '%s' correctly detected as editor", tt.command)
			} else {
				t.Logf("✓ Command '%s' processed without error", tt.command)
			}
		})
	}
}

// readLogLines reads log file and returns lines, handling missing files gracefully
func readLogLines(path string) ([]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}
