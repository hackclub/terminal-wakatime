package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Shell string

const (
	Bash Shell = "bash"
	Zsh  Shell = "zsh"
	Fish Shell = "fish"
)

type Integration struct {
	shell         Shell
	binPath       string
	enableTiming  bool
	enableDetails bool
}

func NewIntegration(binPath string) *Integration {
	shell := detectShell()

	return &Integration{
		shell:         shell,
		binPath:       binPath,
		enableTiming:  os.Getenv("TERMINAL_WAKATIME_COMMAND_TIMING") == "true",
		enableDetails: os.Getenv("TERMINAL_WAKATIME_PROCESS_DETAILS") == "true",
	}
}

func NewIntegrationForShell(binPath, shellName string) *Integration {
	var shell Shell
	switch strings.ToLower(shellName) {
	case "fish":
		shell = Fish
	case "zsh":
		shell = Zsh
	case "bash":
		shell = Bash
	default:
		shell = Bash // Default fallback
	}
	
	return &Integration{
		shell:         shell,
		binPath:       binPath,
		enableTiming:  os.Getenv("TERMINAL_WAKATIME_COMMAND_TIMING") == "true",
		enableDetails: os.Getenv("TERMINAL_WAKATIME_PROCESS_DETAILS") == "true",
	}
}

// isRunningInFish checks if we're currently running inside a Fish shell
// Fish is tricky to detect because FISH_VERSION is not exported as an env var
func isRunningInFish() bool {
	// The most reliable way is to check the $SHELL but also see if 
	// we're being piped from fish (which would indicate fish | source)
	shell := os.Getenv("SHELL")
	if shell != "" && filepath.Base(shell) == "fish" {
		return true
	}
	
	// Alternative: Check if stdin suggests we're being piped from fish
	// When fish runs "terminal-wakatime init | source", we can sometimes detect this
	return false
}

func detectShell() Shell {
	// Check for shell-specific environment variables first
	// These are more reliable than $SHELL when shells are nested
	
	// For zsh and bash, check version environment variables
	zshVersion := os.Getenv("ZSH_VERSION")  
	if zshVersion != "" {
		return Zsh
	}
	
	bashVersion := os.Getenv("BASH_VERSION")
	if bashVersion != "" {
		return Bash
	}
	
	// For fish, check if we can run fish built-in commands
	// Fish doesn't export FISH_VERSION as an environment variable
	if isRunningInFish() {
		return Fish
	}
	
	// Fallback to $SHELL environment variable
	shell := os.Getenv("SHELL")
	if shell == "" {
		return Bash // Default fallback
	}

	base := filepath.Base(shell)
	switch base {
	case "zsh":
		return Zsh
	case "fish":
		return Fish
	case "bash":
		return Bash
	default:
		return Bash // Default to bash-compatible
	}
}

func (i *Integration) GenerateHooks() string {
	switch i.shell {
	case Bash:
		return i.generateBashHooks()
	case Zsh:
		return i.generateZshHooks()
	case Fish:
		return i.generateFishHooks()
	default:
		return i.generateBashHooks()
	}
}

func (i *Integration) generateBashHooks() string {
	preExec := fmt.Sprintf(`
__terminal_wakatime_preexec() {
    if [ -n "$1" ]; then
        export __TERMINAL_WAKATIME_COMMAND="$1"
        export __TERMINAL_WAKATIME_START_TIME="$(date +%%s)"
        export __TERMINAL_WAKATIME_PWD="$PWD"
    fi
}`)

	postExec := fmt.Sprintf(`
__terminal_wakatime_postexec() {
    if [ -n "$__TERMINAL_WAKATIME_COMMAND" ]; then
        local end_time="$(date +%%s)"
        local duration=$((end_time - __TERMINAL_WAKATIME_START_TIME))
        
        # Only track commands that run for a minimum duration
        if [ "$duration" -ge 2 ]; then
            "%s" track --command "$__TERMINAL_WAKATIME_COMMAND" --duration "$duration" --pwd "$__TERMINAL_WAKATIME_PWD" >/dev/null 2>&1 &
        fi
        
        unset __TERMINAL_WAKATIME_COMMAND
        unset __TERMINAL_WAKATIME_START_TIME
        unset __TERMINAL_WAKATIME_PWD
    fi
}`, i.binPath)

	promptCommand := `
if [[ "$PROMPT_COMMAND" != *"__terminal_wakatime_postexec"* ]]; then
    PROMPT_COMMAND="__terminal_wakatime_postexec; $PROMPT_COMMAND"
fi`

	// Add preexec hook for bash (requires bash-preexec or manual setup)
	preexecSetup := `
if [[ -n "$BASH_VERSION" ]]; then
    if command -v __bp_install >/dev/null 2>&1; then
        # bash-preexec is available
        preexec_functions+=(__terminal_wakatime_preexec)
    else
        # Fallback: use DEBUG trap (less reliable but works)
        if [[ "$PS4" != *"__terminal_wakatime_preexec"* ]]; then
            __original_ps4="$PS4"
            PS4='$(__terminal_wakatime_preexec "$BASH_COMMAND"; echo "$__original_ps4")'
            set -T
        fi
    fi
fi`

	return fmt.Sprintf("%s\n%s\n%s\n%s", preExec, postExec, promptCommand, preexecSetup)
}

func (i *Integration) generateZshHooks() string {
	preExec := fmt.Sprintf(`
__terminal_wakatime_preexec() {
    if [ -n "$1" ]; then
        export __TERMINAL_WAKATIME_COMMAND="$1"
        export __TERMINAL_WAKATIME_START_TIME="$(date +%%s)"
        export __TERMINAL_WAKATIME_PWD="$PWD"
    fi
}`)

	postExec := fmt.Sprintf(`
__terminal_wakatime_precmd() {
    if [ -n "$__TERMINAL_WAKATIME_COMMAND" ]; then
        local end_time="$(date +%%s)"
        local duration=$((end_time - __TERMINAL_WAKATIME_START_TIME))
        
        # Only track commands that run for a minimum duration
        if [ "$duration" -ge 2 ]; then
            "%s" track --command "$__TERMINAL_WAKATIME_COMMAND" --duration "$duration" --pwd "$__TERMINAL_WAKATIME_PWD" >/dev/null 2>&1 &
        fi
        
        unset __TERMINAL_WAKATIME_COMMAND
        unset __TERMINAL_WAKATIME_START_TIME
        unset __TERMINAL_WAKATIME_PWD
    fi
}`, i.binPath)

	hookSetup := `
# Add hooks to zsh
if [[ -n "$ZSH_VERSION" ]]; then
    if [[ "$preexec_functions" != *"__terminal_wakatime_preexec"* ]]; then
        preexec_functions+=(__terminal_wakatime_preexec)
    fi
    
    if [[ "$precmd_functions" != *"__terminal_wakatime_precmd"* ]]; then
        precmd_functions+=(__terminal_wakatime_precmd)
    fi
fi`

	return fmt.Sprintf("%s\n%s\n%s", preExec, postExec, hookSetup)
}

func (i *Integration) generateFishHooks() string {
	return fmt.Sprintf(`
function __terminal_wakatime_preexec --on-event fish_preexec
    set -g __TERMINAL_WAKATIME_COMMAND $argv[1]
    set -g __TERMINAL_WAKATIME_START_TIME (date +%%s)
    set -g __TERMINAL_WAKATIME_PWD $PWD
end

function __terminal_wakatime_postexec --on-event fish_postexec
    if set -q __TERMINAL_WAKATIME_COMMAND
        set end_time (date +%%s)
        set duration (math $end_time - $__TERMINAL_WAKATIME_START_TIME)
        
        # Only track commands that run for a minimum duration
        if test $duration -ge 2
            "%s" track --command "$__TERMINAL_WAKATIME_COMMAND" --duration "$duration" --pwd "$__TERMINAL_WAKATIME_PWD" >/dev/null 2>&1 &
        end
        
        set -e __TERMINAL_WAKATIME_COMMAND
        set -e __TERMINAL_WAKATIME_START_TIME
        set -e __TERMINAL_WAKATIME_PWD
    end
end`, i.binPath)
}

func (i *Integration) GetShellName() string {
	return string(i.shell)
}

func (i *Integration) GetConfigFileRecommendations() []string {
	switch i.shell {
	case Bash:
		return []string{
			"~/.bashrc",
			"~/.bash_profile",
			"~/.profile",
		}
	case Zsh:
		return []string{
			"~/.zshrc",
			"~/.zprofile",
		}
	case Fish:
		return []string{
			"~/.config/fish/config.fish",
		}
	default:
		return []string{"~/.bashrc"}
	}
}

func (i *Integration) GenerateInstallCommand() string {
	switch i.shell {
	case Fish:
		return fmt.Sprintf(`echo 'eval ("%s" init)' >> ~/.config/fish/config.fish`, i.binPath)
	default:
		configFile := "~/.bashrc"
		if i.shell == Zsh {
			configFile = "~/.zshrc"
		}
		return fmt.Sprintf(`echo 'eval "$(%s init)"' >> %s`, i.binPath, configFile)
	}
}

func (i *Integration) ValidateEnvironment() []string {
	var issues []string

	// Check if binary exists and is executable
	if _, err := os.Stat(i.binPath); os.IsNotExist(err) {
		issues = append(issues, fmt.Sprintf("Binary not found at %s", i.binPath))
	}

	// Check shell-specific requirements
	switch i.shell {
	case Bash:
		// Check if bash-preexec is available for better command tracking
		if !commandExists("__bp_install") {
			issues = append(issues, "Consider installing bash-preexec for better command tracking: https://github.com/rcaloras/bash-preexec")
		}
	case Zsh:
		// Zsh has built-in preexec/precmd support
	case Fish:
		// Fish has built-in event system
	}

	// Check for conflicting integrations
	existingIntegrations := []string{
		"WAKATIME_HOME",
		"WAKATIME_PROJECT",
		"_WAKATIME_",
	}

	for _, env := range existingIntegrations {
		if os.Getenv(env) != "" {
			issues = append(issues, fmt.Sprintf("Potential conflict: %s environment variable is set", env))
		}
	}

	return issues
}

func commandExists(cmd string) bool {
	// This is a simplified check - in a real implementation you'd use exec.LookPath
	// or run a command to check if it exists
	return false
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// GetShellVersion returns the version of the detected shell
func GetShellVersion(shell Shell) string {
	switch shell {
	case Bash:
		return getBashVersion()
	case Zsh:
		return getZshVersion()
	case Fish:
		return getFishVersion()
	default:
		return "unknown"
	}
}

// getBashVersion gets the bash version from environment or command
func getBashVersion() string {
	// First try environment variable
	if version := os.Getenv("BASH_VERSION"); version != "" {
		// BASH_VERSION is like "5.1.16(1)-release", extract just "5.1.16"
		if idx := strings.Index(version, "("); idx != -1 {
			return version[:idx]
		}
		return version
	}

	// Fallback to running bash --version
	if cmd := exec.Command("bash", "--version"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 0 {
				// First line is like "GNU bash, version 5.1.16(1)-release (x86_64-apple-darwin21.0)"
				words := strings.Fields(lines[0])
				for _, word := range words {
					if strings.Contains(word, ".") && (strings.HasPrefix(word, "version") || 
						(len(word) > 0 && word[0] >= '0' && word[0] <= '9')) {
						version := strings.TrimPrefix(word, "version")
						if idx := strings.Index(version, "("); idx != -1 {
							return version[:idx]
						}
						if idx := strings.Index(version, "-"); idx != -1 {
							return version[:idx]
						}
						return version
					}
				}
			}
		}
	}

	return "unknown"
}

// getZshVersion gets the zsh version from environment or command
func getZshVersion() string {
	// First try environment variable
	if version := os.Getenv("ZSH_VERSION"); version != "" {
		return version
	}

	// Fallback to running zsh --version
	if cmd := exec.Command("zsh", "--version"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			// Output is like "zsh 5.8 (x86_64-apple-darwin21.0)"
			words := strings.Fields(string(output))
			if len(words) >= 2 {
				return words[1]
			}
		}
	}

	return "unknown"
}

// getFishVersion gets the fish version
func getFishVersion() string {
	// Fish doesn't export FISH_VERSION, so we need to run fish --version
	if cmd := exec.Command("fish", "--version"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			// Output is like "fish, version 3.4.1"
			words := strings.Fields(string(output))
			if len(words) >= 3 {
				return words[2]
			}
		}
	}

	return "unknown"
}

// FormatPluginString formats the plugin string according to WakaTime spec:
// "{editor_name}/{editor_version} {plugin_name}/{plugin_version}"
func FormatPluginString(pluginName, pluginVersion string) string {
	shell := detectShell()
	shellVersion := GetShellVersion(shell)
	
	return fmt.Sprintf("%s/%s %s/%s", string(shell), shellVersion, pluginName, pluginVersion)
}

// GetCurrentShellInfo returns the current shell and its version
func GetCurrentShellInfo() (Shell, string) {
	shell := detectShell()
	version := GetShellVersion(shell)
	return shell, version
}
