package monitor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
	"github.com/hackclub/terminal-wakatime/pkg/tracker"
	"github.com/hackclub/terminal-wakatime/pkg/updater"
)

type Monitor struct {
	config  *config.Config
	tracker *tracker.Tracker
	updater *updater.Updater
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

	// Get current binary path for updater
	binaryPath, _ := os.Executable()
	upd := updater.NewUpdater(cfg.PluginVersion(), cfg.WakaTimeDir(), binaryPath)

	return &Monitor{
		config:  cfg,
		tracker: tracker.NewTracker(cfg),
		updater: upd,
		logFile: logFile,
	}
}

func (m *Monitor) ProcessCommand(command string, duration time.Duration, workingDir string) error {
	// Check for pending update notifications (show once then clear)
	m.checkAndShowUpdateNotification()

	// Check for updates in background (non-blocking)
	// Skip updates if disabled via environment variable (useful for tests)
	if os.Getenv("TERMINAL_WAKATIME_DISABLE_UPDATES") == "" {
		go m.updater.CheckAndUpdate()
	}

	// Log the command for debugging
	m.logCommand(command, duration, workingDir)

	// Skip very short commands
	if duration < m.config.MinCommandTime {
		return nil
	}

	// Track the command
	return m.tracker.TrackCommand(command, workingDir)
}

// checkAndShowUpdateNotification checks for pending update notifications and shows them
func (m *Monitor) checkAndShowUpdateNotification() {
	updateInfo, err := m.updater.GetPendingUpdateInfo()
	if err != nil || updateInfo == nil {
		return // No pending notification or error reading it
	}

	// Show the notification
	fmt.Fprintf(os.Stderr, "\n🚀 FYI! terminal-wakatime here. I self-updated from %s to %s.\n\n",
		updateInfo.FromVersion, updateInfo.ToVersion)

	// Clear the notification (it's shown once)
	m.updater.ClearPendingUpdateInfo()
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
