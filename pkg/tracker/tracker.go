package tracker

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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
	Entity        string
	EntityType    ActivityType
	Category      string
	Language      string
	Project       string
	Branch        string
	IsWrite       bool
	Timestamp     time.Time
	Lines         *int
	LineNo        *int
	CursorPos     *int
	LineAdditions *int
	LineDeletions *int
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
	activity := t.parseCommandToSingleActivity(command, workingDir)
	if activity != nil {
		return t.sendActivity(activity)
	}
	return nil
}

func (t *Tracker) TrackFile(filePath string, isWrite bool) error {
	activity := &Activity{
		Entity:     filePath,
		EntityType: ActivityFile,
		Category:   "coding",
		Language:   detectLanguage(filePath),
		Project:    t.detectProject(filePath),
		Branch:     getGitBranch(filepath.Dir(filePath)),
		IsWrite:    isWrite,
		Timestamp:  time.Now(),
		Lines:      getFileLines(filePath),
		LineNo:     getDefaultLineNumber(),
		CursorPos:  getDefaultCursorPos(),
	}

	return t.sendActivity(activity)
}

func (t *Tracker) parseCommandToSingleActivity(command string, workingDir string) *Activity {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return nil
	}

	cmdName := filepath.Base(fields[0])

	// Check for editor commands
	if t.isEditor(cmdName) {
		return t.handleEditorCommandSingle(fields, workingDir)
	}

	// Check for git commands - handle with rich metadata
	if cmdName == "git" {
		return t.handleGitCommandSingle(fields, workingDir)
	}

	// Check for build/test commands
	if t.isBuildTestCommand(cmdName) {
		return t.handleBuildTestCommandSingle(fields, workingDir)
	}

	// Check for coding apps
	if category, isCodingApp := codingApps[cmdName]; isCodingApp {
		return &Activity{
			Entity:     cmdName,
			EntityType: ActivityApp,
			Category:   category,
			Project:    t.detectProject(workingDir),
			Branch:     getGitBranch(workingDir),
			Timestamp:  time.Now(),
		}
	}

	// Check for remote connections
	if domain := t.parseRemoteConnection(command); domain != "" {
		return &Activity{
			Entity:     domain,
			EntityType: ActivityDomain,
			Category:   "coding",
			Project:    t.detectProject(workingDir),
			Timestamp:  time.Now(),
		}
	}

	// Check for directory changes
	if cmdName == "cd" && len(fields) > 1 {
		targetDir := fields[1]
		if !filepath.IsAbs(targetDir) {
			targetDir = filepath.Join(workingDir, targetDir)
		}

		return &Activity{
			Entity:     targetDir,
			EntityType: ActivityFile,
			Category:   "browsing",
			Project:    t.detectProject(targetDir),
			Branch:     getGitBranch(targetDir),
			Timestamp:  time.Now(),
		}
	}

	// For any other terminal command, create a general coding activity
	return &Activity{
		Entity:     cmdName,
		EntityType: ActivityApp,
		Category:   "coding",
		Project:    t.detectProject(workingDir),
		Branch:     getGitBranch(workingDir),
		Timestamp:  time.Now(),
	}
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
				Language:   detectLanguage(filePath),
				Project:    t.detectProject(filePath),
				Branch:     getGitBranch(filepath.Dir(filePath)),
				IsWrite:    true, // File editing is typically writing
				Timestamp:  time.Now(),
				Lines:      getFileLines(filePath),
				LineNo:     getDefaultLineNumber(),
				CursorPos:  getDefaultCursorPos(),
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
			Branch:     getGitBranch(workingDir),
			Timestamp:  time.Now(),
		}
		activities = append(activities, activity)
	}

	return activities
}

// handleEditorCommandSingle processes editor commands into a single activity
func (t *Tracker) handleEditorCommandSingle(fields []string, workingDir string) *Activity {
	cmdName := filepath.Base(fields[0])
	t.showEditorSuggestion(cmdName)

	totalLines := 0
	var primaryFile string
	var primaryLanguage string
	fileCount := 0

	// Look for file arguments and aggregate metadata
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
			fileCount++
			if primaryFile == "" {
				primaryFile = filePath
				primaryLanguage = detectLanguage(filePath)
			}
			if lines := getFileLines(filePath); lines != nil {
				totalLines += *lines
			}
		}
	}

	// If we found files, create activity for the primary file with aggregated metadata
	if fileCount > 0 {
		return &Activity{
			Entity:     primaryFile,
			EntityType: ActivityFile,
			Category:   "coding",
			Language:   primaryLanguage,
			Project:    t.detectProject(primaryFile),
			Branch:     getGitBranch(filepath.Dir(primaryFile)),
			IsWrite:    true,
			Timestamp:  time.Now(),
			Lines:      &totalLines,
			LineNo:     getDefaultLineNumber(),
			CursorPos:  getDefaultCursorPos(),
		}
	}

	// If no files found, track the editor itself
	return &Activity{
		Entity:     cmdName,
		EntityType: ActivityApp,
		Category:   "coding",
		Project:    t.detectProject(workingDir),
		Branch:     getGitBranch(workingDir),
		Timestamp:  time.Now(),
	}
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
		activity.Lines,
		activity.LineNo,
		activity.CursorPos,
		activity.LineAdditions,
		activity.LineDeletions,
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

// detectLanguage detects programming language from file extension
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "Go"
	case ".js", ".jsx":
		return "JavaScript"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".c":
		return "C"
	case ".cpp", ".cc", ".cxx":
		return "C++"
	case ".h", ".hpp":
		return "C Header"
	case ".php":
		return "PHP"
	case ".rb":
		return "Ruby"
	case ".sh", ".bash", ".zsh":
		return "Shell"
	case ".md", ".markdown":
		return "Markdown"
	case ".html", ".htm":
		return "HTML"
	case ".css":
		return "CSS"
	case ".scss", ".sass":
		return "SCSS"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".xml":
		return "XML"
	case ".sql":
		return "SQL"
	case ".dockerfile":
		return "Docker"
	case ".toml":
		return "TOML"
	case ".ini":
		return "INI"
	default:
		// Check for special cases
		basename := strings.ToLower(filepath.Base(filePath))
		switch basename {
		case "dockerfile":
			return "Docker"
		case "makefile":
			return "Makefile"
		case "go.mod", "go.sum":
			return "Go Module"
		case "package.json", "package-lock.json":
			return "JSON"
		case "cargo.toml", "cargo.lock":
			return "TOML"
		}
		return ""
	}
}

// getFileLines returns the number of lines in a file
func getFileLines(filePath string) *int {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}

	if err := scanner.Err(); err != nil {
		return nil
	}

	return &lines
}

// getGitBranch returns the current git branch for the given directory
func getGitBranch(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getGitChangedFiles returns files changed in the last commit with their line changes
func getGitChangedFiles(dir string) ([]GitFileChange, error) {
	cmd := exec.Command("git", "diff", "--stat", "HEAD~1", "HEAD", "--numstat")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var changes []GitFileChange
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			added, _ := strconv.Atoi(parts[0])
			deleted, _ := strconv.Atoi(parts[1])
			filePath := parts[2]

			changes = append(changes, GitFileChange{
				FilePath:      filePath,
				LineAdditions: added,
				LineDeletions: deleted,
			})
		}
	}

	return changes, nil
}

type GitFileChange struct {
	FilePath      string
	LineAdditions int
	LineDeletions int
}

// handleGitCommand processes git commands with rich metadata
func (t *Tracker) handleGitCommand(fields []string, workingDir string) []*Activity {
	var activities []*Activity

	if len(fields) < 2 {
		return activities
	}

	gitSubcommand := fields[1]

	switch gitSubcommand {
	case "commit", "push", "merge", "rebase":
		// For commit operations, track the files being committed with line changes
		changes, err := getGitChangedFiles(workingDir)
		if err != nil || len(changes) == 0 {
			// Fallback to simple git activity
			activity := &Activity{
				Entity:     "git " + gitSubcommand,
				EntityType: ActivityApp,
				Category:   "code reviewing",
				Project:    t.detectProject(workingDir),
				Branch:     getGitBranch(workingDir),
				IsWrite:    true,
				Timestamp:  time.Now(),
			}
			activities = append(activities, activity)
		} else {
			// Create activities for each changed file
			for _, change := range changes {
				filePath := filepath.Join(workingDir, change.FilePath)
				activity := &Activity{
					Entity:        filePath,
					EntityType:    ActivityFile,
					Category:      "code reviewing",
					Language:      detectLanguage(filePath),
					Project:       t.detectProject(workingDir),
					Branch:        getGitBranch(workingDir),
					IsWrite:       true,
					Timestamp:     time.Now(),
					Lines:         getFileLines(filePath),
					LineAdditions: &change.LineAdditions,
					LineDeletions: &change.LineDeletions,
				}
				activities = append(activities, activity)
			}
		}

	case "status", "log", "diff", "show":
		// Read operations
		activity := &Activity{
			Entity:     "git " + gitSubcommand,
			EntityType: ActivityApp,
			Category:   "code reviewing",
			Project:    t.detectProject(workingDir),
			Branch:     getGitBranch(workingDir),
			IsWrite:    false,
			Timestamp:  time.Now(),
		}
		activities = append(activities, activity)

	default:
		// Other git operations
		activity := &Activity{
			Entity:     "git " + gitSubcommand,
			EntityType: ActivityApp,
			Category:   "code reviewing",
			Project:    t.detectProject(workingDir),
			Branch:     getGitBranch(workingDir),
			Timestamp:  time.Now(),
		}
		activities = append(activities, activity)
	}

	return activities
}

// handleGitCommandSingle processes git commands into a single activity with aggregated metadata
func (t *Tracker) handleGitCommandSingle(fields []string, workingDir string) *Activity {
	if len(fields) < 2 {
		return &Activity{
			Entity:     "git",
			EntityType: ActivityApp,
			Category:   "coding",
			Project:    t.detectProject(workingDir),
			Branch:     getGitBranch(workingDir),
			Timestamp:  time.Now(),
		}
	}

	gitSubcommand := fields[1]

	// Privacy-safe command name (just git + subcommand, no arguments that might contain secrets)
	entity := "git " + gitSubcommand

	switch gitSubcommand {
	case "commit", "push", "merge", "rebase":
		// For write operations, aggregate line changes from all affected files
		changes, err := getGitChangedFiles(workingDir)
		totalAdditions := 0
		totalDeletions := 0
		totalLines := 0

		if err == nil && len(changes) > 0 {
			for _, change := range changes {
				totalAdditions += change.LineAdditions
				totalDeletions += change.LineDeletions
				filePath := filepath.Join(workingDir, change.FilePath)
				if lines := getFileLines(filePath); lines != nil {
					totalLines += *lines
				}
			}
		}

		return &Activity{
			Entity:        entity,
			EntityType:    ActivityApp,
			Category:      "coding",
			Project:       t.detectProject(workingDir),
			Branch:        getGitBranch(workingDir),
			IsWrite:       true,
			Timestamp:     time.Now(),
			Lines:         &totalLines,
			LineAdditions: &totalAdditions,
			LineDeletions: &totalDeletions,
		}

	case "status", "log", "diff", "show":
		// Read operations
		return &Activity{
			Entity:     entity,
			EntityType: ActivityApp,
			Category:   "coding",
			Project:    t.detectProject(workingDir),
			Branch:     getGitBranch(workingDir),
			IsWrite:    false,
			Timestamp:  time.Now(),
		}

	default:
		// Generic git command
		return &Activity{
			Entity:     entity,
			EntityType: ActivityApp,
			Category:   "coding",
			Project:    t.detectProject(workingDir),
			Branch:     getGitBranch(workingDir),
			Timestamp:  time.Now(),
		}
	}
}

// isBuildTestCommand checks if command is a build/test operation
func (t *Tracker) isBuildTestCommand(cmdName string) bool {
	buildTestCommands := []string{
		"npm", "yarn", "pnpm", "bun",
		"cargo", "go", "mvn", "gradle",
		"make", "cmake", "ninja",
		"python", "pytest", "jest", "mocha",
		"tsc", "webpack", "vite", "rollup",
		"docker", "docker-compose",
		"terraform", "ansible",
	}

	for _, cmd := range buildTestCommands {
		if cmd == cmdName {
			return true
		}
	}
	return false
}

// handleBuildTestCommand processes build/test commands with context
func (t *Tracker) handleBuildTestCommand(fields []string, workingDir string) []*Activity {
	var activities []*Activity

	if len(fields) == 0 {
		return activities
	}

	cmdName := fields[0]
	subcommand := ""
	if len(fields) > 1 {
		subcommand = fields[1]
	}

	// Determine category based on subcommand
	category := "coding"
	if isTestCommand(subcommand) {
		category = "debugging"
	} else if isBuildCommand(subcommand) {
		category = "building"
	}

	// Try to detect language from project context
	language := t.detectProjectLanguage(workingDir)

	activity := &Activity{
		Entity:     cmdName + " " + subcommand,
		EntityType: ActivityApp,
		Category:   category,
		Language:   language,
		Project:    t.detectProject(workingDir),
		Branch:     getGitBranch(workingDir),
		Timestamp:  time.Now(),
	}

	activities = append(activities, activity)
	return activities
}

// handleBuildTestCommandSingle processes build/test commands into a single activity
func (t *Tracker) handleBuildTestCommandSingle(fields []string, workingDir string) *Activity {
	if len(fields) == 0 {
		return nil
	}

	cmdName := fields[0]
	subcommand := ""
	if len(fields) > 1 {
		subcommand = fields[1]
	}

	// Privacy-safe command name (just command + subcommand, no args that might contain secrets)
	entity := cmdName
	if subcommand != "" {
		entity = cmdName + " " + subcommand
	}

	// Determine category based on subcommand
	category := "coding"
	if isTestCommand(subcommand) {
		category = "debugging"
	} else if isBuildCommand(subcommand) {
		category = "building"
	}

	// Try to detect language from project context
	language := t.detectProjectLanguage(workingDir)

	return &Activity{
		Entity:     entity,
		EntityType: ActivityApp,
		Category:   category,
		Language:   language,
		Project:    t.detectProject(workingDir),
		Branch:     getGitBranch(workingDir),
		Timestamp:  time.Now(),
	}
}

// isTestCommand checks if subcommand is a test operation
func isTestCommand(subcommand string) bool {
	testCommands := []string{"test", "check", "verify", "spec", "jest", "mocha", "pytest"}
	for _, cmd := range testCommands {
		if cmd == subcommand {
			return true
		}
	}
	return false
}

// isBuildCommand checks if subcommand is a build operation
func isBuildCommand(subcommand string) bool {
	buildCommands := []string{"build", "compile", "install", "deploy", "publish", "dist", "bundle"}
	for _, cmd := range buildCommands {
		if cmd == subcommand {
			return true
		}
	}
	return false
}

// detectProjectLanguage detects primary language from project files
func (t *Tracker) detectProjectLanguage(workingDir string) string {
	// Check for language-specific project files
	languageFiles := map[string]string{
		"go.mod":           "Go",
		"package.json":     "JavaScript",
		"Cargo.toml":       "Rust",
		"pom.xml":          "Java",
		"requirements.txt": "Python",
		"setup.py":         "Python",
		"Gemfile":          "Ruby",
		"composer.json":    "PHP",
	}

	for file, language := range languageFiles {
		if _, err := os.Stat(filepath.Join(workingDir, file)); err == nil {
			return language
		}
	}

	return ""
}

// getDefaultLineNumber returns a default line number for file operations
func getDefaultLineNumber() *int {
	line := 1
	return &line
}

// getDefaultCursorPos returns a default cursor position
func getDefaultCursorPos() *int {
	pos := 1
	return &pos
}
