package shell

import (
	"fmt"
	"os"
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

func detectShell() Shell {
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
