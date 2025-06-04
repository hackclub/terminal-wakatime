package tracker

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
	"github.com/hackclub/terminal-wakatime/pkg/wakatime"
)

type ActivityType string

const (
	ActivityFile   ActivityType = "file"
	ActivityApp    ActivityType = "app"
	ActivityDomain ActivityType = "domain"
)

type Activity struct {
	Entity     string
	EntityType ActivityType
	Category   string
	Language   string
	Project    string
	Branch     string
	IsWrite    bool
	Timestamp  time.Time
}

type Tracker struct {
	config       *config.Config
	wakatime     *wakatime.CLI
	lastSentTime time.Time
	lastSentFile string
	suggestions  map[string]time.Time
}

var (
	// Editor patterns for detection
	editorPatterns = map[string]*regexp.Regexp{
		"vim":     regexp.MustCompile(`\b(vi|vim|nvim)\b`),
		"emacs":   regexp.MustCompile(`\bemacs\b`),
		"nano":    regexp.MustCompile(`\bnano\b`),
		"code":    regexp.MustCompile(`\bcode\b`),
		"sublime": regexp.MustCompile(`\b(subl|sublime_text)\b`),
		"atom":    regexp.MustCompile(`\batom\b`),
		"idea":    regexp.MustCompile(`\b(idea|intellij|pycharm|webstorm)\b`),
	}

	// App patterns for coding tools
	codingApps = map[string]string{
		"amp":     "coding",
		"claude":  "coding",
		"cursor":  "coding",
		"node":    "coding",
		"python":  "coding",
		"go":      "coding",
		"java":    "coding",
		"rust":    "coding",
		"php":     "coding",
		"ruby":    "coding",
		"psql":    "coding",
		"mysql":   "coding",
		"mongo":   "coding",
		"redis":   "coding",
		"docker":  "building",
		"kubectl": "coding",
		"helm":    "coding",
		"make":    "building",
		"cmake":   "building",
		"ninja":   "building",
		"npm":     "building",
		"yarn":    "building",
		"pip":     "building",
		"cargo":   "building",
		"mvn":     "building",
		"gradle":  "building",
		"git":     "code reviewing",
	}

	// Remote connection patterns
	remotePatterns = []*regexp.Regexp{
		regexp.MustCompile(`ssh\s+(?:.*@)?([^@\s]+)`),
		regexp.MustCompile(`mysql\s+.*-h\s+([^\s]+)`),
		regexp.MustCompile(`psql\s+.*-h\s+([^\s]+)`),
		regexp.MustCompile(`redis-cli\s+.*-h\s+([^\s]+)`),
	}
)

func NewTracker(cfg *config.Config) *Tracker {
	return &Tracker{
		config:      cfg,
		wakatime:    wakatime.NewCLI(cfg),
		suggestions: make(map[string]time.Time),
	}
}

func (t *Tracker) TrackCommand(command string, workingDir string) error {
	activities := t.parseCommand(command, workingDir)

	for _, activity := range activities {
		if err := t.sendActivity(activity); err != nil {
			return err
		}
	}

	return nil
}

func (t *Tracker) TrackFile(filePath string, isWrite bool) error {
	activity := &Activity{
		Entity:     filePath,
		EntityType: ActivityFile,
		Category:   "coding",
		IsWrite:    isWrite,
		Timestamp:  time.Now(),
	}

	// Detect project from file path
	activity.Project = t.detectProject(filePath)

	return t.sendActivity(activity)
}

func (t *Tracker) parseCommand(command string, workingDir string) []*Activity {
	var activities []*Activity

	fields := strings.Fields(command)
	if len(fields) == 0 {
		return activities
	}

	cmdName := filepath.Base(fields[0])

	// Check for editor commands
	if t.isEditor(cmdName) {
		activities = append(activities, t.handleEditorCommand(fields, workingDir)...)
		t.showEditorSuggestion(cmdName)
		return activities
	}

	// Check for coding apps
	if category, isCodingApp := codingApps[cmdName]; isCodingApp {
		activity := &Activity{
			Entity:     cmdName,
			EntityType: ActivityApp,
			Category:   category,
			Project:    t.detectProject(workingDir),
			Timestamp:  time.Now(),
		}
		activities = append(activities, activity)
		return activities
	}

	// Check for remote connections
	if domain := t.parseRemoteConnection(command); domain != "" {
		activity := &Activity{
			Entity:     domain,
			EntityType: ActivityDomain,
			Category:   "coding",
			Project:    t.detectProject(workingDir),
			Timestamp:  time.Now(),
		}
		activities = append(activities, activity)
		return activities
	}

	// Check for directory changes
	if cmdName == "cd" && len(fields) > 1 {
		targetDir := fields[1]
		if !filepath.IsAbs(targetDir) {
			targetDir = filepath.Join(workingDir, targetDir)
		}

		activity := &Activity{
			Entity:     targetDir,
			EntityType: ActivityFile,
			Category:   "browsing",
			Project:    t.detectProject(targetDir),
			Timestamp:  time.Now(),
		}
		activities = append(activities, activity)
	}

	return activities
}

func (t *Tracker) isEditor(cmdName string) bool {
	for _, pattern := range editorPatterns {
		if pattern.MatchString(cmdName) {
			return true
		}
	}
	return false
}

func (t *Tracker) handleEditorCommand(fields []string, workingDir string) []*Activity {
	var activities []*Activity

	cmdName := filepath.Base(fields[0])

	// Look for file arguments
	for i := 1; i < len(fields); i++ {
		arg := fields[i]

		// Skip flags
		if strings.HasPrefix(arg, "-") {
			continue
		}

		// Resolve file path
		filePath := arg
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(workingDir, filePath)
		}

		// Check if file exists or could be created
		if _, err := os.Stat(filePath); err == nil || !os.IsNotExist(err) {
			activity := &Activity{
				Entity:     filePath,
				EntityType: ActivityFile,
				Category:   "coding",
				Project:    t.detectProject(filePath),
				Timestamp:  time.Now(),
			}
			activities = append(activities, activity)
		}
	}

	// If no files found, track the editor itself
	if len(activities) == 0 {
		activity := &Activity{
			Entity:     cmdName,
			EntityType: ActivityApp,
			Category:   "coding",
			Project:    t.detectProject(workingDir),
			Timestamp:  time.Now(),
		}
		activities = append(activities, activity)
	}

	return activities
}

func (t *Tracker) parseRemoteConnection(command string) string {
	for _, pattern := range remotePatterns {
		matches := pattern.FindStringSubmatch(command)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func (t *Tracker) detectProject(filePath string) string {
	if t.config.Project != "" {
		return t.config.Project
	}

	dir := filePath
	if !isDir(filePath) {
		dir = filepath.Dir(filePath)
	}

	// Look for project indicators
	projectFiles := []string{
		".git",
		"package.json",
		"go.mod",
		"Cargo.toml",
		"pom.xml",
		"build.gradle",
		"requirements.txt",
		"Pipfile",
		"composer.json",
		"Gemfile",
		"mix.exs",
	}

	for {
		for _, file := range projectFiles {
			if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
				return filepath.Base(dir)
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback to directory name
	return filepath.Base(filePath)
}

func (t *Tracker) sendActivity(activity *Activity) error {
	// Implement official WakaTime plugin pattern:
	// Call wakatime-cli if: enoughTimeHasPassed OR fileChanged OR isWriteEvent
	if !t.shouldSendHeartbeat(activity) {
		return nil
	}

	// Ensure wakatime-cli is installed before sending heartbeat
	if err := t.wakatime.EnsureInstalled(); err != nil {
		return fmt.Errorf("failed to ensure wakatime-cli is installed: %w", err)
	}

	// Send heartbeat - let wakatime-cli handle rate limiting and deduplication
	err := t.wakatime.SendHeartbeat(
		activity.Entity,
		string(activity.EntityType),
		activity.Category,
		activity.Language,
		activity.Project,
		activity.Branch,
		activity.IsWrite,
	)

	if err == nil {
		// Update tracking for next decision
		t.lastSentTime = activity.Timestamp
		t.lastSentFile = activity.Entity
	}

	return err
}

// shouldSendHeartbeat implements the official WakaTime plugin pattern
func (t *Tracker) shouldSendHeartbeat(activity *Activity) bool {
	// Always send on write events (file save)
	if activity.IsWrite {
		return true
	}

	// Always send if file has changed
	if activity.Entity != t.lastSentFile {
		return true
	}

	// Send if enough time has passed (2 minutes as per WakaTime spec)
	return time.Since(t.lastSentTime) >= config.WakaTimeInterval
}

func (t *Tracker) showEditorSuggestion(editor string) {
	if t.config.DisableEditorSuggestions {
		return
	}

	// Check if we've shown this suggestion recently
	key := fmt.Sprintf("editor:%s", editor)
	if lastShown, exists := t.suggestions[key]; exists {
		if time.Since(lastShown) < t.config.EditorSuggestionFrequency {
			return
		}
	}

	// Show suggestion
	suggestion := t.getEditorSuggestion(editor)
	if suggestion != "" {
		fmt.Fprintf(os.Stderr, "\n💡 %s\n\n", suggestion)
		t.suggestions[key] = time.Now()
	}
}

func (t *Tracker) getEditorSuggestion(editor string) string {
	suggestions := map[string]string{
		"vim":     "Tip: You're using Vim! For detailed tracking including keystrokes,\n   cursor movement, and mode changes, install vim-wakatime:\n   \n   https://github.com/wakatime/vim-wakatime\n   \n   Terminal WakaTime will continue tracking your session.",
		"emacs":   "Tip: You're using Emacs! Install wakatime-mode for comprehensive\n   Emacs integration: https://github.com/wakatime/wakatime-mode",
		"code":    "Tip: You're using VS Code! Install the official WakaTime extension\n   for comprehensive tracking: https://github.com/wakatime/vscode-wakatime",
		"nano":    "Tip: You're using Nano! Terminal WakaTime tracks your session.\n   For more detailed tracking, consider using an editor with a WakaTime plugin.",
		"sublime": "Tip: You're using Sublime Text! Install sublime-wakatime for\n   enhanced tracking: https://github.com/wakatime/sublime-wakatime",
		"atom":    "Tip: You're using Atom! Install atom-wakatime for enhanced\n   tracking: https://github.com/wakatime/atom-wakatime",
	}

	return suggestions[editor]
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
