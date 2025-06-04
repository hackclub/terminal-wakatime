package monitor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
	"github.com/hackclub/terminal-wakatime/pkg/tracker"
)

type Monitor struct {
	config  *config.Config
	tracker *tracker.Tracker
	logFile string
}

type CommandEvent struct {
	Command    string
	Duration   time.Duration
	WorkingDir string
	Timestamp  time.Time
}

func NewMonitor(cfg *config.Config) *Monitor {
	logFile := filepath.Join(cfg.WakaTimeDir(), "commands.log")

	return &Monitor{
		config:  cfg,
		tracker: tracker.NewTracker(cfg),
		logFile: logFile,
	}
}

func (m *Monitor) ProcessCommand(command string, duration time.Duration, workingDir string) error {
	// Log the command for debugging
	m.logCommand(command, duration, workingDir)

	// Skip very short commands
	if duration < m.config.MinCommandTime {
		return nil
	}

	// Track the command
	return m.tracker.TrackCommand(command, workingDir)
}

func (m *Monitor) ProcessFileEdit(filePath string, isWrite bool) error {
	// Ensure absolute path
	if !filepath.IsAbs(filePath) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		filePath = filepath.Join(wd, filePath)
	}

	return m.tracker.TrackFile(filePath, isWrite)
}

func (m *Monitor) StartFileWatcher(ctx context.Context, directories []string) error {
	// This is a simplified file watcher
	// In a production version, you'd use fsnotify or similar
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	watchedFiles := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for _, dir := range directories {
				if err := m.scanDirectory(dir, watchedFiles); err != nil {
					if m.config.Debug {
						fmt.Fprintf(os.Stderr, "Error scanning directory %s: %v\n", dir, err)
					}
				}
			}
		}
	}
}

func (m *Monitor) scanDirectory(dir string, watchedFiles map[string]time.Time) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check if file is a code file
		if !m.isCodeFile(path) {
			return nil
		}

		modTime := info.ModTime()
		if lastSeen, exists := watchedFiles[path]; !exists || modTime.After(lastSeen) {
			watchedFiles[path] = modTime

			// Track file modification
			if exists {
				m.tracker.TrackFile(path, true)
			}
		}

		return nil
	})
}

func (m *Monitor) isCodeFile(filePath string) bool {
	codeExtensions := []string{
		".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".c", ".cpp", ".h", ".hpp",
		".rs", ".php", ".rb", ".swift", ".kt", ".scala", ".clj", ".hs", ".ml", ".elm",
		".css", ".scss", ".sass", ".less", ".html", ".xml", ".json", ".yaml", ".yml",
		".toml", ".ini", ".cfg", ".conf", ".sh", ".bash", ".zsh", ".fish", ".ps1",
		".sql", ".md", ".rst", ".tex", ".dockerfile", ".makefile", ".cmake",
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	for _, codeExt := range codeExtensions {
		if ext == codeExt {
			return true
		}
	}

	// Check for files without extensions that might be code
	base := strings.ToLower(filepath.Base(filePath))
	specialFiles := []string{
		"dockerfile", "makefile", "cmakelists.txt", "rakefile", "gemfile",
		"requirements.txt", "setup.py", "package.json", "cargo.toml", "go.mod",
	}

	for _, special := range specialFiles {
		if base == special {
			return true
		}
	}

	return false
}

func (m *Monitor) logCommand(command string, duration time.Duration, workingDir string) {
	if !m.config.Debug {
		return
	}

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(m.logFile), 0755); err != nil {
		return
	}

	file, err := os.OpenFile(m.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s\t%s\t%v\t%s\n", timestamp, workingDir, duration, command)
	file.WriteString(logEntry)
}

func (m *Monitor) GetRecentCommands(limit int) ([]CommandEvent, error) {
	file, err := os.Open(m.logFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []CommandEvent
	scanner := bufio.NewScanner(file)

	// Read all lines into memory (for simplicity)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Process last N lines
	start := len(lines) - limit
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines); i++ {
		event, err := m.parseLogLine(lines[i])
		if err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

func (m *Monitor) parseLogLine(line string) (CommandEvent, error) {
	parts := strings.Split(line, "\t")
	if len(parts) < 4 {
		return CommandEvent{}, fmt.Errorf("invalid log line format")
	}

	timestamp, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return CommandEvent{}, err
	}

	workingDir := parts[1]

	duration, err := time.ParseDuration(parts[2])
	if err != nil {
		return CommandEvent{}, err
	}

	command := parts[3]

	return CommandEvent{
		Command:    command,
		Duration:   duration,
		WorkingDir: workingDir,
		Timestamp:  timestamp,
	}, nil
}

func (m *Monitor) GetStatus() (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Get recent activity count
	recentCommands, err := m.GetRecentCommands(10)
	if err == nil {
		status["recent_commands"] = len(recentCommands)
	}

	// Check if wakatime CLI is installed
	status["wakatime_cli_installed"] = m.tracker != nil

	// Get configuration status
	status["api_key_configured"] = m.config.APIKey != ""
	status["debug_enabled"] = m.config.Debug
	status["heartbeat_frequency"] = m.config.HeartbeatFrequency.String()

	return status, nil
}

func ParseTrackCommand(args []string) (*CommandEvent, error) {
	var command, pwd string
	var duration time.Duration

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--command":
			if i+1 < len(args) {
				command = args[i+1]
				i++
			}
		case "--duration":
			if i+1 < len(args) {
				if seconds, err := strconv.Atoi(args[i+1]); err == nil {
					duration = time.Duration(seconds) * time.Second
				}
				i++
			}
		case "--pwd":
			if i+1 < len(args) {
				pwd = args[i+1]
				i++
			}
		}
	}

	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	if pwd == "" {
		if wd, err := os.Getwd(); err == nil {
			pwd = wd
		}
	}

	return &CommandEvent{
		Command:    command,
		Duration:   duration,
		WorkingDir: pwd,
		Timestamp:  time.Now(),
	}, nil
}
