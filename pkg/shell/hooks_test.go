package shell

import (
	"os"
	"strings"
	"testing"
)

func TestDetectShell(t *testing.T) {
	tests := []struct {
		shellPath string
		expected  Shell
	}{
		{"/bin/bash", Bash},
		{"/usr/bin/bash", Bash},
		{"/bin/zsh", Zsh},
		{"/usr/local/bin/zsh", Zsh},
		{"/usr/bin/fish", Fish},
		{"/usr/local/bin/fish", Fish},
		{"/bin/sh", Bash},             // fallback
		{"", Bash},                    // fallback when SHELL is empty
		{"/some/unknown/shell", Bash}, // fallback for unknown shells
	}

	for _, tt := range tests {
		t.Run(tt.shellPath, func(t *testing.T) {
			// Save original SHELL env var
			originalShell := os.Getenv("SHELL")
			defer os.Setenv("SHELL", originalShell)

			os.Setenv("SHELL", tt.shellPath)
			result := detectShell()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s for shell path %s", tt.expected, result, tt.shellPath)
			}
		})
	}
}

func TestNewIntegration(t *testing.T) {
	binPath := "/usr/local/bin/terminal-wakatime"
	integration := NewIntegration(binPath)

	if integration.binPath != binPath {
		t.Errorf("Expected binPath %s, got %s", binPath, integration.binPath)
	}

	if integration.shell == "" {
		t.Error("Expected shell to be detected")
	}
}

func TestGenerateBashHooks(t *testing.T) {
	integration := &Integration{
		shell:   Bash,
		binPath: "/usr/local/bin/terminal-wakatime",
	}

	hooks := integration.generateBashHooks()

	// Check that essential components are present
	expectedParts := []string{
		"__terminal_wakatime_preexec",
		"__terminal_wakatime_postexec",
		"PROMPT_COMMAND",
		integration.binPath,
	}

	for _, part := range expectedParts {
		if !strings.Contains(hooks, part) {
			t.Errorf("Expected hooks to contain '%s'", part)
		}
	}

	// Check that the binary path is properly quoted/escaped
	if !strings.Contains(hooks, `"`+integration.binPath+`"`) {
		t.Error("Expected binary path to be properly quoted in hooks")
	}
}

func TestGenerateZshHooks(t *testing.T) {
	integration := &Integration{
		shell:   Zsh,
		binPath: "/usr/local/bin/terminal-wakatime",
	}

	hooks := integration.generateZshHooks()

	expectedParts := []string{
		"__terminal_wakatime_preexec",
		"__terminal_wakatime_precmd",
		"preexec_functions",
		"precmd_functions",
		integration.binPath,
	}

	for _, part := range expectedParts {
		if !strings.Contains(hooks, part) {
			t.Errorf("Expected hooks to contain '%s'", part)
		}
	}
}

func TestGenerateFishHooks(t *testing.T) {
	integration := &Integration{
		shell:   Fish,
		binPath: "/usr/local/bin/terminal-wakatime",
	}

	hooks := integration.generateFishHooks()

	expectedParts := []string{
		"__terminal_wakatime_preexec",
		"__terminal_wakatime_postexec",
		"fish_preexec",
		"fish_postexec",
		integration.binPath,
	}

	for _, part := range expectedParts {
		if !strings.Contains(hooks, part) {
			t.Errorf("Expected hooks to contain '%s'", part)
		}
	}

	// Fish uses different syntax, check for Fish-specific elements
	if !strings.Contains(hooks, "function ") {
		t.Error("Expected Fish function syntax")
	}

	if !strings.Contains(hooks, "--on-event") {
		t.Error("Expected Fish event handling syntax")
	}
}

func TestGenerateHooks(t *testing.T) {
	binPath := "/usr/local/bin/terminal-wakatime"

	tests := []struct {
		shell    Shell
		contains []string
	}{
		{
			shell: Bash,
			contains: []string{
				"__terminal_wakatime_preexec",
				"__terminal_wakatime_postexec",
				"PROMPT_COMMAND",
			},
		},
		{
			shell: Zsh,
			contains: []string{
				"__terminal_wakatime_preexec",
				"__terminal_wakatime_precmd",
				"preexec_functions",
			},
		},
		{
			shell: Fish,
			contains: []string{
				"__terminal_wakatime_preexec",
				"__terminal_wakatime_postexec",
				"fish_preexec",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			integration := &Integration{
				shell:   tt.shell,
				binPath: binPath,
			}

			hooks := integration.GenerateHooks()

			for _, expected := range tt.contains {
				if !strings.Contains(hooks, expected) {
					t.Errorf("Expected hooks for %s to contain '%s'", tt.shell, expected)
				}
			}

			// Verify binary path is included
			if !strings.Contains(hooks, binPath) {
				t.Errorf("Expected hooks to contain binary path '%s'", binPath)
			}
		})
	}
}

func TestGetConfigFileRecommendations(t *testing.T) {
	tests := []struct {
		shell    Shell
		expected []string
	}{
		{
			shell: Bash,
			expected: []string{
				"~/.bashrc",
				"~/.bash_profile",
				"~/.profile",
			},
		},
		{
			shell: Zsh,
			expected: []string{
				"~/.zshrc",
				"~/.zprofile",
			},
		},
		{
			shell: Fish,
			expected: []string{
				"~/.config/fish/config.fish",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			integration := &Integration{shell: tt.shell}
			recommendations := integration.GetConfigFileRecommendations()

			if len(recommendations) != len(tt.expected) {
				t.Errorf("Expected %d recommendations, got %d", len(tt.expected), len(recommendations))
			}

			for i, expected := range tt.expected {
				if i >= len(recommendations) || recommendations[i] != expected {
					t.Errorf("Expected recommendation %d to be '%s', got '%s'", i, expected, recommendations[i])
				}
			}
		})
	}
}

func TestGenerateInstallCommand(t *testing.T) {
	binPath := "/usr/local/bin/terminal-wakatime"

	tests := []struct {
		shell    Shell
		contains []string
	}{
		{
			shell: Bash,
			contains: []string{
				`eval "$(/usr/local/bin/terminal-wakatime init)"`,
				"~/.bashrc",
			},
		},
		{
			shell: Zsh,
			contains: []string{
				`eval "$(/usr/local/bin/terminal-wakatime init)"`,
				"~/.zshrc",
			},
		},
		{
			shell: Fish,
			contains: []string{
				`eval ("/usr/local/bin/terminal-wakatime" init)`,
				"~/.config/fish/config.fish",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			integration := &Integration{
				shell:   tt.shell,
				binPath: binPath,
			}

			command := integration.GenerateInstallCommand()

			for _, expected := range tt.contains {
				if !strings.Contains(command, expected) {
					t.Errorf("Expected install command for %s to contain '%s', got: %s", tt.shell, expected, command)
				}
			}
		})
	}
}

func TestGetShellName(t *testing.T) {
	tests := []struct {
		shell    Shell
		expected string
	}{
		{Bash, "bash"},
		{Zsh, "zsh"},
		{Fish, "fish"},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			integration := &Integration{shell: tt.shell}
			name := integration.GetShellName()
			if name != tt.expected {
				t.Errorf("Expected shell name '%s', got '%s'", tt.expected, name)
			}
		})
	}
}

func TestValidateEnvironment(t *testing.T) {
	// Test with non-existent binary
	integration := &Integration{
		binPath: "/non/existent/binary",
		shell:   Bash,
	}

	issues := integration.ValidateEnvironment()

	// Should report that binary is not found
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "Binary not found") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected validation to report missing binary")
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", ""}, // Will be home + "/test", can't predict exact value
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := expandPath(tt.input)

			if strings.HasPrefix(tt.input, "~/") {
				// For home paths, just check it doesn't start with ~
				if strings.HasPrefix(result, "~") {
					t.Errorf("Expected path expansion for '%s', but result still starts with ~: %s", tt.input, result)
				}
			} else {
				// For non-home paths, should be unchanged
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}
