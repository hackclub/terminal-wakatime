package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
	"github.com/hackclub/terminal-wakatime/pkg/monitor"
	"github.com/hackclub/terminal-wakatime/pkg/shell"
	"github.com/hackclub/terminal-wakatime/pkg/wakatime"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	verbose bool
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func execute() error {
	var err error
	cfg, err = config.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	rootCmd := &cobra.Command{
		Use:   "terminal-wakatime",
		Short: "WakaTime plugin for tracking coding time in terminal environments",
		Long: `Terminal WakaTime tracks your coding activity in terminal environments
and sends it to WakaTime for time tracking and analytics.

It monitors terminal activity across multiple shells (Bash, Zsh, Fish, etc.)
and detects when you're working on files, using coding tools, or connecting
to remote systems.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				cfg.Debug = true
			}
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(heartbeatCmd())
	rootCmd.AddCommand(trackCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(testCmd())
	rootCmd.AddCommand(depsCmd())
	rootCmd.AddCommand(debugCmd())
	rootCmd.AddCommand(versionCmd())

	return rootCmd.Execute()
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [shell]",
		Short: "Generate shell integration code",
		Long: `Generate shell-specific integration code that should be added to your shell configuration.

For Bash/Zsh: eval "$(terminal-wakatime init)"
For Fish: terminal-wakatime init fish | source

Optionally specify the shell type: terminal-wakatime init fish`,
		RunE: func(cmd *cobra.Command, args []string) error {
			binPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}

			var integration *shell.Integration
			if len(args) > 0 {
				// Shell type specified as argument
				integration = shell.NewIntegrationForShell(binPath, args[0])
			} else {
				// Auto-detect shell
				integration = shell.NewIntegration(binPath)
			}
			
			hooks := integration.GenerateHooks()
			fmt.Print(hooks)
			return nil
		},
	}

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure terminal-wakatime settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigCommand(cmd, args)
		},
	}

	cmd.Flags().String("key", "", "Set WakaTime API key")
	cmd.Flags().String("project", "", "Set default project name")
	cmd.Flags().Int("heartbeat-frequency", 0, "Set heartbeat frequency in seconds (for display only - wakatime-cli handles actual rate limiting)")
	cmd.Flags().Bool("debug", false, "Enable debug mode")
	cmd.Flags().Bool("show", false, "Show current configuration")
	cmd.Flags().Bool("disable-editor-suggestions", false, "Disable editor plugin suggestions")

	return cmd
}

func runConfigCommand(cmd *cobra.Command, args []string) error {
	show, _ := cmd.Flags().GetBool("show")
	if show {
		return showConfig()
	}

	modified := false

	if key, _ := cmd.Flags().GetString("key"); key != "" {
		cfg.APIKey = key
		modified = true
	}

	if project, _ := cmd.Flags().GetString("project"); project != "" {
		cfg.Project = project
		modified = true
	}

	if freq, _ := cmd.Flags().GetInt("heartbeat-frequency"); freq > 0 {
		cfg.HeartbeatFrequency = time.Duration(freq) * time.Second
		modified = true
	}

	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		cfg.Debug = true
		modified = true
	}

	if disable, _ := cmd.Flags().GetBool("disable-editor-suggestions"); disable {
		cfg.DisableEditorSuggestions = true
		modified = true
	}

	if modified {
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("Configuration saved successfully")
	} else {
		fmt.Println("No configuration changes specified")
	}

	return nil
}

func showConfig() error {
	fmt.Printf("Configuration file: %s\n", cfg.ConfigFile())
	fmt.Printf("API Key: %s\n", maskAPIKey(cfg.APIKey))
	fmt.Printf("API URL: %s\n", cfg.APIUrl)
	fmt.Printf("Debug: %t\n", cfg.Debug)
	fmt.Printf("Hide Filenames: %t\n", cfg.HideFilenames)
	fmt.Printf("Heartbeat Frequency: %s\n", cfg.HeartbeatFrequency)
	fmt.Printf("Min Command Time: %s\n", cfg.MinCommandTime)
	fmt.Printf("Project: %s\n", cfg.Project)
	fmt.Printf("Disable Editor Suggestions: %t\n", cfg.DisableEditorSuggestions)

	if len(cfg.Exclude) > 0 {
		fmt.Printf("Exclude: %s\n", strings.Join(cfg.Exclude, ", "))
	}

	if len(cfg.Include) > 0 {
		fmt.Printf("Include: %s\n", strings.Join(cfg.Include, ", "))
	}

	return nil
}

func maskAPIKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

func heartbeatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Send a heartbeat to WakaTime",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHeartbeatCommand(cmd, args)
		},
	}

	cmd.Flags().String("entity", "", "Entity (file path, app name, or domain)")
	cmd.Flags().String("entity-type", "file", "Entity type (file, app, domain)")
	cmd.Flags().String("category", "", "Category (coding, building, browsing, etc.)")
	cmd.Flags().String("language", "", "Programming language")
	cmd.Flags().String("project", "", "Project name")
	cmd.Flags().String("branch", "", "Git branch")
	cmd.Flags().Bool("write", false, "Mark as write operation")

	cmd.MarkFlagRequired("entity")

	return cmd
}

func runHeartbeatCommand(cmd *cobra.Command, args []string) error {
	// Ensure wakatime-cli is installed
	wakatimeCLI := wakatime.NewCLI(cfg)
	if err := wakatimeCLI.EnsureInstalled(); err != nil {
		return fmt.Errorf("failed to ensure wakatime-cli is installed: %w", err)
	}

	entity, _ := cmd.Flags().GetString("entity")
	entityType, _ := cmd.Flags().GetString("entity-type")
	category, _ := cmd.Flags().GetString("category")
	language, _ := cmd.Flags().GetString("language")
	project, _ := cmd.Flags().GetString("project")
	branch, _ := cmd.Flags().GetString("branch")
	isWrite, _ := cmd.Flags().GetBool("write")

	return wakatimeCLI.SendHeartbeat(entity, entityType, category, language, project, branch, isWrite)
}

func trackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "track",
		Short: "Track a command execution",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTrackCommand(cmd, args)
		},
	}

	cmd.Flags().String("command", "", "Command that was executed")
	cmd.Flags().Int("duration", 0, "Duration in seconds")
	cmd.Flags().String("pwd", "", "Working directory")

	return cmd
}

func runTrackCommand(cmd *cobra.Command, args []string) error {
	// Try to get values from flags first
	command, _ := cmd.Flags().GetString("command")
	duration, _ := cmd.Flags().GetInt("duration")
	pwd, _ := cmd.Flags().GetString("pwd")

	// If no flags provided, try to parse from args (backward compatibility)
	if command == "" && len(args) > 0 {
		event, err := monitor.ParseTrackCommand(args)
		if err != nil {
			return err
		}
		command = event.Command
		duration = int(event.Duration.Seconds())
		pwd = event.WorkingDir
	}

	// Validate required fields
	if command == "" {
		return fmt.Errorf("command is required (use --command flag)")
	}
	if duration <= 0 {
		return fmt.Errorf("duration must be greater than 0 (use --duration flag)")
	}
	if pwd == "" {
		return fmt.Errorf("working directory is required (use --pwd flag)")
	}

	mon := monitor.NewMonitor(cfg)
	return mon.ProcessCommand(command, time.Duration(duration)*time.Second, pwd)
}

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current status and recent activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			mon := monitor.NewMonitor(cfg)
			status, err := mon.GetStatus()
			if err != nil {
				return err
			}

			fmt.Println("Terminal WakaTime Status:")
			fmt.Println("========================")

			for key, value := range status {
				fmt.Printf("%s: %v\n", formatKey(key), value)
			}

			// Show recent commands
			recentCommands, err := mon.GetRecentCommands(5)
			if err == nil && len(recentCommands) > 0 {
				fmt.Println("\nRecent Commands:")
				fmt.Println("================")
				for _, cmd := range recentCommands {
					fmt.Printf("%s: %s (duration: %v)\n",
						cmd.Timestamp.Format("15:04:05"),
						truncateString(cmd.Command, 50),
						cmd.Duration)
				}
			}

			return nil
		},
	}

	return cmd
}

func testCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test connection to WakaTime API",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure wakatime-cli is installed
			wakatimeCLI := wakatime.NewCLI(cfg)
			if err := wakatimeCLI.EnsureInstalled(); err != nil {
				return fmt.Errorf("failed to ensure wakatime-cli is installed: %w", err)
			}

			// Test configuration
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("configuration validation failed: %w", err)
			}

			// Test API connection
			if err := wakatimeCLI.TestConnection(); err != nil {
				return fmt.Errorf("API connection test failed: %w", err)
			}

			fmt.Println("✓ Configuration is valid")
			fmt.Println("✓ WakaTime CLI is installed")
			fmt.Println("✓ API connection successful")
			return nil
		},
	}

	return cmd
}

func depsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Manage dependencies (wakatime-cli)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDepsCommand(cmd, args)
		},
	}

	cmd.Flags().Bool("status", false, "Check dependency status")
	cmd.Flags().Bool("reinstall", false, "Force reinstall dependencies")

	return cmd
}

func runDepsCommand(cmd *cobra.Command, args []string) error {
	wakatimeCLI := wakatime.NewCLI(cfg)

	status, _ := cmd.Flags().GetBool("status")
	reinstall, _ := cmd.Flags().GetBool("reinstall")

	if status {
		if wakatimeCLI.IsInstalled() {
			fmt.Printf("✓ WakaTime CLI is installed at: %s\n", wakatimeCLI.BinaryPath())
		} else {
			fmt.Println("✗ WakaTime CLI is not installed")
		}
		return nil
	}

	if reinstall {
		// Remove existing binary
		os.Remove(wakatimeCLI.BinaryPath())
	}

	fmt.Println("Installing/updating WakaTime CLI...")
	if err := wakatimeCLI.EnsureInstalled(); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	fmt.Println("✓ Dependencies installed successfully")
	return nil
}

func debugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Show debug information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDebugCommand(cmd, args)
		},
	}

	cmd.Flags().Bool("system", false, "Show system information")
	cmd.Flags().Bool("shell", false, "Show shell environment")
	cmd.Flags().Bool("heartbeats", false, "Show recent heartbeats")

	return cmd
}

func runDebugCommand(cmd *cobra.Command, args []string) error {
	system, _ := cmd.Flags().GetBool("system")
	shellEnv, _ := cmd.Flags().GetBool("shell")
	heartbeats, _ := cmd.Flags().GetBool("heartbeats")

	if !system && !shellEnv && !heartbeats {
		// Show all by default
		system = true
		shellEnv = true
		heartbeats = true
	}

	if system {
		fmt.Println("System Information:")
		fmt.Println("==================")
		fmt.Printf("Executable: %s\n", getExecutablePath())
		fmt.Printf("Config file: %s\n", cfg.ConfigFile())
		fmt.Printf("WakaTime directory: %s\n", cfg.WakaTimeDir())
		fmt.Printf("Debug enabled: %t\n", cfg.Debug)
		fmt.Println()
	}

	if shellEnv {
		fmt.Println("Shell Environment:")
		fmt.Println("==================")
		binPath, _ := os.Executable()
		integration := shell.NewIntegration(binPath)
		fmt.Printf("Detected shell: %s\n", integration.GetShellName())

		issues := integration.ValidateEnvironment()
		if len(issues) > 0 {
			fmt.Println("Issues found:")
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue)
			}
		} else {
			fmt.Println("✓ No issues found")
		}
		fmt.Println()
	}

	if heartbeats {
		fmt.Println("Recent Activity:")
		fmt.Println("================")
		mon := monitor.NewMonitor(cfg)
		commands, err := mon.GetRecentCommands(10)
		if err != nil {
			fmt.Printf("Error reading recent commands: %v\n", err)
		} else if len(commands) == 0 {
			fmt.Println("No recent activity found")
		} else {
			for _, cmd := range commands {
				fmt.Printf("%s: %s (duration: %v, dir: %s)\n",
					cmd.Timestamp.Format("2006-01-02 15:04:05"),
					truncateString(cmd.Command, 60),
					cmd.Duration,
					filepath.Base(cmd.WorkingDir))
			}
		}
	}

	return nil
}

func formatKey(key string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(key, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.Title(part)
		}
	}
	return strings.Join(parts, " ")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("terminal-wakatime version %s\n", config.PluginVersion)
		},
	}
	return cmd
}

func getExecutablePath() string {
	path, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	return path
}
