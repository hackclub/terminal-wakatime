package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/ini.v1"
)

const (
	DefaultAPIURL             = "https://api.wakatime.com/api/v1"
	DefaultHeartbeatFrequency = 2 * time.Minute // For display only - wakatime-cli handles actual rate limiting
	DefaultMinCommandTime     = 2 * time.Second
	DefaultConfigFile         = ".wakatime.cfg"
	DefaultWakaTimeDir        = ".wakatime"
	PluginName                = "terminal-wakatime"
	// WakaTime official plugin interval - hardcoded as per spec
	WakaTimeInterval = 2 * time.Minute
)

// PluginVersion will be set at build time via ldflags
var PluginVersion = "dev"

type Config struct {
	APIKey                     string
	APIUrl                     string
	Debug                      bool
	HideFilenames              bool
	HeartbeatFrequency         time.Duration
	MinCommandTime             time.Duration
	DisableEditorSuggestions   bool
	EditorSuggestionFrequency  time.Duration
	EditorSuggestions          []string
	Project                    string
	Exclude                    []string
	Include                    []string
	IncludeOnlyWithProjectFile bool
	configFile                 string
	wakaTimeDir                string
}

func NewConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configFile := filepath.Join(homeDir, DefaultConfigFile)
	wakaTimeDir := filepath.Join(homeDir, DefaultWakaTimeDir)

	config := &Config{
		APIUrl:                    DefaultAPIURL,
		Debug:                     false,
		HideFilenames:             false,
		HeartbeatFrequency:        DefaultHeartbeatFrequency,
		MinCommandTime:            DefaultMinCommandTime,
		DisableEditorSuggestions:  false,
		EditorSuggestionFrequency: 24 * time.Hour,
		EditorSuggestions:         []string{"vim", "emacs", "code", "sublime", "atom"},
		configFile:                configFile,
		wakaTimeDir:               wakaTimeDir,
	}

	if err := config.Load(); err != nil {
		return config, err
	}

	return config, nil
}

func (c *Config) Load() error {
	// Load from config file if it exists
	if _, err := os.Stat(c.configFile); !os.IsNotExist(err) {
		cfg, err := ini.Load(c.configFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}

		section := cfg.Section("settings")

		if key := section.Key("api_key"); key.String() != "" {
			c.APIKey = key.String()
		}

		if url := section.Key("api_url"); url.String() != "" {
			c.APIUrl = url.String()
		}

		if debug, err := section.Key("debug").Bool(); err == nil {
			c.Debug = debug
		}

		if hide, err := section.Key("hidefilenames").Bool(); err == nil {
			c.HideFilenames = hide
		}

		if project := section.Key("project"); project.String() != "" {
			c.Project = project.String()
		}

		if exclude := section.Key("exclude").Strings("\n"); len(exclude) > 0 {
			c.Exclude = exclude
		}

		if include := section.Key("include").Strings("\n"); len(include) > 0 {
			c.Include = include
		}

		if includeOnly, err := section.Key("include_only_with_project_file").Bool(); err == nil {
			c.IncludeOnlyWithProjectFile = includeOnly
		}
	}

	// Load environment variables for terminal-wakatime specific settings
	if freq := os.Getenv("TERMINAL_WAKATIME_HEARTBEAT_FREQUENCY"); freq != "" {
		if seconds, err := strconv.Atoi(freq); err == nil {
			c.HeartbeatFrequency = time.Duration(seconds) * time.Second
		}
	}

	if minTime := os.Getenv("TERMINAL_WAKATIME_MIN_COMMAND_TIME"); minTime != "" {
		if seconds, err := strconv.Atoi(minTime); err == nil {
			c.MinCommandTime = time.Duration(seconds) * time.Second
		}
	}

	if disable := os.Getenv("TERMINAL_WAKATIME_DISABLE_EDITOR_SUGGESTIONS"); disable == "true" {
		c.DisableEditorSuggestions = true
	}

	return nil
}

func (c *Config) Save() error {
	cfg := ini.Empty()
	section := cfg.Section("settings")

	section.Key("api_key").SetValue(c.APIKey)
	section.Key("api_url").SetValue(c.APIUrl)
	section.Key("debug").SetValue(strconv.FormatBool(c.Debug))
	section.Key("hidefilenames").SetValue(strconv.FormatBool(c.HideFilenames))

	if c.Project != "" {
		section.Key("project").SetValue(c.Project)
	}

	if len(c.Exclude) > 0 {
		section.Key("exclude").SetValue(joinStrings(c.Exclude, "\n"))
	}

	if len(c.Include) > 0 {
		section.Key("include").SetValue(joinStrings(c.Include, "\n"))
	}

	section.Key("include_only_with_project_file").SetValue(strconv.FormatBool(c.IncludeOnlyWithProjectFile))

	if err := os.MkdirAll(filepath.Dir(c.configFile), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return cfg.SaveTo(c.configFile)
}

func (c *Config) WakaTimeDir() string {
	return c.wakaTimeDir
}

func (c *Config) ConfigFile() string {
	return c.configFile
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if c.APIUrl == "" {
		return fmt.Errorf("API URL is required")
	}

	return nil
}

func joinStrings(slice []string, sep string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for _, s := range slice[1:] {
		result += sep + s
	}
	return result
}
