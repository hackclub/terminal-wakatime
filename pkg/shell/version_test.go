package shell

import (
	"strings"
	"testing"
)

func TestGetShellVersion(t *testing.T) {
	tests := []struct {
		name     string
		shell    Shell
		expected bool // whether we expect a valid version (not "unknown")
	}{
		{"bash version", Bash, true},
		{"zsh version", Zsh, true},
		{"fish version", Fish, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := GetShellVersion(tt.shell)
			
			if tt.expected {
				// Version should not be "unknown" and should contain at least one dot
				if version == "unknown" {
					t.Logf("Warning: Could not detect %s version (this is ok if %s is not installed)", tt.shell, tt.shell)
				} else if !strings.Contains(version, ".") {
					t.Errorf("Expected version to contain a dot, got: %s", version)
				} else {
					t.Logf("✓ Detected %s version: %s", tt.shell, version)
				}
			}
		})
	}
}

func TestFormatPluginString(t *testing.T) {
	pluginString := FormatPluginString("terminal-wakatime", "1.0.0")
	
	// Should be in format "shell/version terminal-wakatime/1.0.0"
	parts := strings.Split(pluginString, " ")
	if len(parts) != 2 {
		t.Errorf("Expected 2 parts separated by space, got %d: %s", len(parts), pluginString)
		return
	}
	
	shellPart := parts[0]
	pluginPart := parts[1]
	
	// Check shell part contains a slash
	if !strings.Contains(shellPart, "/") {
		t.Errorf("Expected shell part to contain '/', got: %s", shellPart)
	}
	
	// Check plugin part is correct
	if pluginPart != "terminal-wakatime/1.0.0" {
		t.Errorf("Expected plugin part to be 'terminal-wakatime/1.0.0', got: %s", pluginPart)
	}
	
	t.Logf("✓ Plugin string format is correct: %s", pluginString)
}

func TestGetCurrentShellInfo(t *testing.T) {
	shell, version := GetCurrentShellInfo()
	
	// Should detect a valid shell
	validShells := []Shell{Bash, Zsh, Fish}
	isValidShell := false
	for _, validShell := range validShells {
		if shell == validShell {
			isValidShell = true
			break
		}
	}
	
	if !isValidShell {
		t.Errorf("Expected a valid shell (bash, zsh, fish), got: %s", shell)
	}
	
	// Version should not be empty
	if version == "" {
		t.Errorf("Expected version to not be empty")
	}
	
	t.Logf("✓ Current shell: %s version %s", shell, version)
}
