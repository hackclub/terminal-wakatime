package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

const (
	DefaultAPIURL             = "https://api.wakatime.com/api/v1"
	DefaultHeartbeatFrequency = 2 * time.Minute // For display only - wakatime-cli handles actual rate limiting
	DefaultMinCommandTime     = 0 * time.Second
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
	saved                      savedSettings
}

// savedSettings is a snapshot of the [settings] values as loaded from the
// config file, so Save only writes the settings changed in this process.
type savedSettings struct {
	apiKey                     string
	apiURL                     string
	debug                      bool
	hideFilenames              bool
	project                    string
	exclude                    []string
	include                    []string
	includeOnlyWithProjectFile bool
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

	c.saved = c.snapshot()

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

// Save writes the settings changed since Load back to the config file.
// ~/.wakatime.cfg is shared with wakatime-cli and every other WakaTime
// plugin, so every other key, section and comment is left untouched.
func (c *Config) Save() error {
	saved := c.saved
	var changed []keyValue

	if c.APIKey != saved.apiKey {
		changed = append(changed, keyValue{"api_key", c.APIKey})
	}
	if c.APIUrl != saved.apiURL {
		changed = append(changed, keyValue{"api_url", c.APIUrl})
	}
	if c.Debug != saved.debug {
		changed = append(changed, keyValue{"debug", strconv.FormatBool(c.Debug)})
	}
	if c.HideFilenames != saved.hideFilenames {
		changed = append(changed, keyValue{"hidefilenames", strconv.FormatBool(c.HideFilenames)})
	}
	if c.Project != saved.project {
		changed = append(changed, keyValue{"project", c.Project})
	}
	if !slices.Equal(c.Exclude, saved.exclude) {
		changed = append(changed, keyValue{"exclude", strings.Join(c.Exclude, "\n")})
	}
	if !slices.Equal(c.Include, saved.include) {
		changed = append(changed, keyValue{"include", strings.Join(c.Include, "\n")})
	}
	if c.IncludeOnlyWithProjectFile != saved.includeOnlyWithProjectFile {
		changed = append(changed, keyValue{"include_only_with_project_file", strconv.FormatBool(c.IncludeOnlyWithProjectFile)})
	}

	if err := updateINISection(c.configFile, "settings", changed); err != nil {
		return err
	}

	c.saved = c.snapshot()
	return nil
}

func (c *Config) snapshot() savedSettings {
	return savedSettings{
		apiKey:                     c.APIKey,
		apiURL:                     c.APIUrl,
		debug:                      c.Debug,
		hideFilenames:              c.HideFilenames,
		project:                    c.Project,
		exclude:                    slices.Clone(c.Exclude),
		include:                    slices.Clone(c.Include),
		includeOnlyWithProjectFile: c.IncludeOnlyWithProjectFile,
	}
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

// PluginVersion returns the current plugin version
func (c *Config) PluginVersion() string {
	return PluginVersion
}
