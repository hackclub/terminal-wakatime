# Terminal WakaTime

A WakaTime plugin for tracking coding time in terminal environments. This plugin monitors terminal activity across multiple shells (Bash, Zsh, Fish, etc.) and sends activity data to WakaTime for time tracking and analytics.

## Overview

Terminal WakaTime is designed as a one-line command that can be added to any shell configuration to automatically track terminal-based coding activity. It detects when you're working on files and sends them to wakatime-cli for time tracking.

## Features

- **Multi-shell Support**: Works with Bash, Zsh, Fish, and other common shells
- **Simple Setup**: Single command setup for any shell
- **Smart Activity Detection**: Files, apps, and remote connections
- **Editor Integration**: Suggests dedicated plugins for better tracking
- **Auto-dependency Management**: Downloads and manages wakatime-cli automatically

## Quick Start

Add this line to your shell configuration file:

```bash
# For Bash (~/.bashrc) or Zsh (~/.zshrc)
eval "$(terminal-wakatime init)"

# For Fish (~/.config/fish/config.fish)
terminal-wakatime init | source
```

## User Story: A Day in the Terminal

Here's how Terminal WakaTime tracks a typical development session, showing the heartbeats sent and plugin interactions:

### Initial Setup
```bash
# User opens terminal
$ cd ~/projects/my-web-app
```
**Heartbeat 1**: `category=browsing, entity=/Users/dev/projects/my-web-app, project=my-web-app`

### Project Work Begins
```bash
$ git status
# On branch feature/auth
# Changes not staged for commit:
#   modified:   src/auth.js
```
**Heartbeat 2**: `category=browsing, entity=git, project=my-web-app, branch=feature/auth`

```bash
$ npm test
# Running tests...
# ✓ 23 tests passed
```
**Heartbeat 3**: `category=building, entity=npm, project=my-web-app` (wakatime-cli detects npm as building)

### File Editing - The Vim Integration
```bash
$ vim src/auth.js
```

**🔄 Plugin Interplay Begins:**

1. **Terminal WakaTime detects vim launch**:
   - **Heartbeat 4**: `category=coding, entity=/Users/dev/projects/my-web-app/src/auth.js, project=my-web-app, language=JavaScript, editor=vim`
   
2. **One-time notification appears** (only shown once per day):
   ```
   💡 Tip: You're using Vim! For detailed tracking including keystrokes, 
      cursor movement, and mode changes, install vim-wakatime:
      
      https://github.com/wakatime/vim-wakatime
      
      Terminal WakaTime will continue tracking your session.
   ```

3. **If vim-wakatime is NOT installed**:
   - Terminal WakaTime tracks session duration and file focus
   - **Heartbeat 5**: `category=coding, entity=/Users/dev/projects/my-web-app/src/auth.js, project=my-web-app, language=JavaScript` (every 2 minutes while active)

4. **If vim-wakatime IS installed**:
   - **Handoff occurs**: Terminal WakaTime detects vim-wakatime and reduces its tracking
   - **Vim-wakatime takes over**: Sends detailed heartbeats with cursor position, mode changes, keystrokes
   - **Vim heartbeats**: `category=coding, entity=/Users/dev/projects/my-web-app/src/auth.js, project=my-web-app, language=JavaScript, lines=156, lineno=42, cursorpos=15`
   - **Terminal WakaTime**: Only tracks vim session start/end times

### Exiting Vim
```bash
# User saves and exits vim (:wq)
$ 
```
**Heartbeat 6**: `category=coding, entity=/Users/dev/projects/my-web-app/src/auth.js, project=my-web-app, language=JavaScript, is_write=true` (file save detected)

### Build Process
```bash
$ npm run build
# Building for production...
# Build completed in 12.3s
```
**Heartbeat 7**: `category=building, entity=npm, project=my-web-app` (wakatime-cli handles categorization)

### Git Operations
```bash
$ git add src/auth.js
$ git commit -m "Add JWT token validation"
```
**Heartbeat 8**: `category=code reviewing, entity=git, project=my-web-app, branch=feature/auth`

### Directory Navigation
```bash
$ cd ../database-service
$ ls -la
```
**Heartbeat 9**: `category=browsing, entity=/Users/dev/projects/database-service, project=database-service`

### AI Coding Assistant
```bash
$ amp
# Amp starts up, user works on code generation and refactoring
```
**Heartbeat 10**: `category=coding, entity=amp, entity-type=app, project=database-service`

### Docker Build
```bash
$ docker build -t myapp .
# Step 1/12 : FROM node:16  
# Step 2/12 : WORKDIR /app
# ...
# Successfully built abc123def456
```
**Heartbeat 11**: `category=building, entity=docker, entity-type=app, project=database-service`

### Multiple Editors
```bash
$ code package.json    # VS Code opens
```
**Heartbeat 12**: `category=coding, entity=/Users/dev/projects/database-service/package.json, project=database-service, language=JSON, editor=vscode`

**Notification** (if vscode-wakatime not installed):
```
💡 Tip: You're using VS Code! Install the official WakaTime extension
   for comprehensive tracking: https://github.com/wakatime/vscode-wakatime
```

### Session Summary

Over this 30-minute terminal session, Terminal WakaTime sent **12 heartbeats** to wakatime-cli:
- **Simple file tracking**: Just sends file paths to wakatime-cli
- **wakatime-cli does the rest**: Detects languages, projects, categories automatically  
- **Editor integration**: Smart notifications for better editor-specific tracking
- **No complex logic**: Follows WakaTime plugin best practice of "keep it simple"

### Plugin Coordination Benefits

**Without vim-wakatime**: Terminal WakaTime provides basic session tracking
- ✅ Session start/end times
- ✅ File being edited  
- ✅ Project context
- ❌ Detailed editing activity

**With vim-wakatime**: Intelligent cooperation
- ✅ **Terminal WakaTime**: Session orchestration, command tracking, project context
- ✅ **Vim WakaTime**: Detailed editing metrics, keystrokes, cursor movement, modes  
- ✅ **No duplicate tracking**: Plugins coordinate to avoid double-counting
- ✅ **Seamless experience**: User gets comprehensive tracking without configuration

This creates the **most accurate time tracking possible** by combining terminal-level session management with editor-specific detailed activity monitoring.

## How It Works

### Architecture

Following official WakaTime plugin best practices, Terminal WakaTime implements the standard pattern:

1. **Detects Terminal Events**: Monitors when you're working on files in the terminal
2. **Calls wakatime-cli intelligently**: Following the official WakaTime plugin pattern:
   - Calls wakatime-cli if **file changes** OR **2+ minutes pass** OR **file save**
   - Lets wakatime-cli handle rate limiting, deduplication, and API communication
3. **Lets wakatime-cli handle everything**: Language detection, project detection, metadata extraction

### Data Flow

```
Terminal Event → Terminal WakaTime → wakatime-cli → WakaTime API
```

### Core Events (Following WakaTime Standards)

- **File Changed**: When you switch to editing a different file (`vim file.js`, `nano config.json`)
- **File Modified**: When you're actively editing a file (keystrokes, file changes)  
- **File Saved**: When file contents are written to disk (`Ctrl+S`, `:w`, etc.)
- **App Usage**: When you use coding tools (`amp`, `claude`, `cursor`, interactive sessions)
- **Domain Work**: When you work on remote systems (`ssh server.com`, database connections)

### What Gets Tracked

Terminal WakaTime detects different activity types and sends them to wakatime-cli:

**Files** (entity-type: file):
- `vim src/app.js` → `wakatime-cli --entity /path/src/app.js`
- `nano ~/.bashrc` → `wakatime-cli --entity /home/user/.bashrc`

**Apps** (entity-type: app):  
- `amp` → `wakatime-cli --entity-type app --entity amp --category coding`
- `claude code` → `wakatime-cli --entity-type app --entity claude --category coding`
- `psql` → `wakatime-cli --entity-type app --entity psql --category coding`
- `node` → `wakatime-cli --entity-type app --entity node --category coding`

**Domains** (entity-type: domain):
- `ssh user@server.com` → `wakatime-cli --entity-type domain --entity server.com`
- `mysql -h db.company.com` → `wakatime-cli --entity-type domain --entity db.company.com`

**wakatime-cli handles**: Project detection, language detection, time tracking, API communication.

## Installation

### 🚀 Quick Install (Recommended for Beginners)

**One command installs everything:**

```bash
curl -fsSL http://hack.club/tw.sh | sh
```

This script will:
- ✅ Auto-detect your OS and architecture  
- ✅ Download the latest version
- ✅ Install the binary to the right location
- ✅ Set up shell integration automatically
- ✅ Guide you through API key setup

**Get your API key:** Visit [wakatime.com/api-key](https://wakatime.com/api-key) and copy your key.

That's it! Restart your terminal and start coding. 🎉

---

### Alternative Install Methods

<details>
<summary>📦 Package Managers</summary>

```bash
# Homebrew (macOS/Linux)
brew install hackclub/tap/terminal-wakatime

# Go (if you have Go installed)
go install github.com/hackclub/terminal-wakatime/cmd/terminal-wakatime@latest
```
</details>

<details>
<summary>💻 Manual Installation</summary>

#### Download Binary

1. Go to [Releases](https://github.com/hackclub/terminal-wakatime/releases/latest)
2. Download the file for your platform:
   - **Linux**: `terminal-wakatime-linux-amd64`
   - **macOS Intel**: `terminal-wakatime-darwin-amd64` 
   - **macOS Apple Silicon**: `terminal-wakatime-darwin-arm64`
   - **Windows**: `terminal-wakatime-windows-amd64.exe`

3. Make it executable and move to your PATH:
   ```bash
   chmod +x terminal-wakatime-*
   sudo mv terminal-wakatime-* /usr/local/bin/terminal-wakatime
   ```

#### Shell Setup

Add this line to your shell config file:

```bash
# For Bash (~/.bashrc) or Zsh (~/.zshrc)
eval "$(terminal-wakatime init)"

# For Fish (~/.config/fish/config.fish)  
terminal-wakatime init | source
```

#### API Key Setup

```bash
terminal-wakatime config --key YOUR_API_KEY
```

Get your API key from: [wakatime.com/api-key](https://wakatime.com/api-key)

</details>

### 🔍 Verify Installation

```bash
terminal-wakatime test
```

This will verify your installation and API connection.

## Automatic Dependencies

Terminal WakaTime automatically downloads and manages wakatime-cli:

- **Auto-installs** wakatime-cli from GitHub releases on first run
- **Keeps updated** - checks for new versions daily
- **Cross-platform** - works on Linux, macOS, Windows (x64/ARM)
- **Secure** - HTTPS downloads with checksum verification

```bash
# Check dependency status
terminal-wakatime deps --status

# Force reinstall if needed
terminal-wakatime deps --reinstall
```

## Configuration

### Basic Configuration

```bash
# Set API key
terminal-wakatime config --key YOUR_API_KEY

# Set custom project name
terminal-wakatime config --project my-terminal-project

# Configure heartbeat display frequency (wakatime-cli handles actual rate limiting)
terminal-wakatime config --heartbeat-frequency 120
```

### Advanced Configuration

Configuration file: `~/.wakatime.cfg`

```ini
[settings]
api_key = YOUR_API_KEY
debug = false
hidefilenames = false
exclude =
    COMMIT_EDITMSG$
    PULLREQ_EDITMSG$
    MERGE_MSG$
    TAG_EDITMSG$
include =
    .*
include_only_with_project_file = false
```

### Shell-specific Options

```bash
# Bash/Zsh: Enable command timing
export TERMINAL_WAKATIME_COMMAND_TIMING=true

# Fish: Enable detailed process monitoring  
set -x TERMINAL_WAKATIME_PROCESS_DETAILS true

# Set minimum command duration to track (default: 2 seconds)
export TERMINAL_WAKATIME_MIN_COMMAND_TIME=2
```

## Usage

Once installed and configured, Terminal WakaTime runs automatically in the background. It follows the standard WakaTime plugin pattern:

1. **Detects terminal activity**: Files, apps, remote connections
2. **Calls wakatime-cli with appropriate flags**: `--entity`, `--entity-type`, `--category`
3. **wakatime-cli handles everything else**: project detection, language detection, sending to API

The plugin sends its version info to wakatime-cli via the `--plugin` flag for proper attribution.

### Manual Commands

```bash
# Send a manual heartbeat
terminal-wakatime heartbeat --entity /path/to/file

# Check status and recent activity
terminal-wakatime status

# View local configuration
terminal-wakatime config --show

# Test connection to WakaTime API
terminal-wakatime test

# Enable debug logging
terminal-wakatime config --debug true
```

## Editor Plugin Integration

Terminal WakaTime intelligently detects when you're using text editors and suggests dedicated WakaTime plugins for enhanced tracking.

### Smart Editor Detection

When you run commands like `vim`, `emacs`, `nano`, `code`, or other editors, Terminal WakaTime will:

1. **Detect the editor launch** and track basic session information
2. **Show a helpful notification** suggesting the dedicated plugin (only once per editor per day)
3. **Continue tracking** the session duration and files opened
4. **Provide installation links** for the specific editor plugin

### Supported Editor Notifications

- **Vim/Neovim**: Suggests [vim-wakatime](https://github.com/wakatime/vim-wakatime) for detailed keystroke and mode tracking
- **Emacs**: Suggests [wakatime-mode](https://github.com/wakatime/wakatime-mode) for comprehensive Emacs integration  
- **VS Code**: Suggests [vscode-wakatime](https://github.com/wakatime/vscode-wakatime) for precise file and project tracking
- **Sublime Text**: Suggests [sublime-wakatime](https://github.com/wakatime/sublime-wakatime) for enhanced Sublime integration
- **Atom**: Suggests [atom-wakatime](https://github.com/wakatime/atom-wakatime) for Atom-specific features
- **IntelliJ/PyCharm**: Suggests [jetbrains-wakatime](https://github.com/wakatime/jetbrains-wakatime) for IDE integration

### Example Notification

```
💡 Tip: You're using Vim! For more detailed tracking including keystrokes, 
   cursor movement, and mode changes, try the official vim-wakatime plugin:
   
   https://github.com/wakatime/vim-wakatime
   
   Terminal WakaTime will continue tracking your session duration.
   (This message appears once per day per editor)
```

### Configuration

```bash
# Disable editor plugin suggestions
terminal-wakatime config --disable-editor-suggestions true
```

## Development

### Project Structure

```
terminal-wakatime/
├── cmd/terminal-wakatime/     # Main application entry point
├── pkg/
│   ├── config/               # Configuration management
│   ├── shell/                # Shell integration and hooks
│   ├── monitor/              # Activity monitoring and event processing
│   ├── tracker/              # WakaTime heartbeat generation
│   └── wakatime/             # WakaTime CLI integration
├── scripts/                  # Build and deployment scripts
└── tests/                    # Test fixtures and integration tests
```

### Building

```bash
# Build for current platform
go build -o terminal-wakatime ./cmd/terminal-wakatime

# Build for all platforms
make build-all

# Run tests
go test ./...

# Run with race detection
go test -race ./...
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Test with race detection
go test -race ./...
```

Testing uses mocked wakatime-cli for reliability. See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed testing guidelines.

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Ensure tests pass: `go test ./...`
5. Format code: `go fmt ./...`
6. Submit a pull request

### Coding Standards

- Follow standard Go conventions and `gofmt` formatting
- Write tests for new functionality
- Include documentation for exported functions
- Use meaningful variable and function names
- Handle errors explicitly

## Troubleshooting

### Common Issues

**Plugin not tracking activity**:
```bash
# Check if hooks are properly installed
echo $PROMPT_COMMAND  # Bash
echo $precmd_functions  # Zsh

# Verify API key is set
terminal-wakatime config --show
```

**WakaTime CLI issues**:
```bash
# Check WakaTime CLI status
terminal-wakatime deps --status

# Reinstall WakaTime CLI
terminal-wakatime deps --reinstall

# Test WakaTime CLI directly
~/.wakatime/wakatime-cli-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) --version
```

**Installation failures**:
```bash
# Check network connectivity
curl -I https://api.github.com/repos/wakatime/wakatime-cli/releases/latest

# Force reinstall if needed
terminal-wakatime deps --reinstall

# Use system package manager as fallback
# Homebrew: brew install wakatime-cli
# APT: apt install wakatime-cli (if available)
```

**Debug logging**:
```bash
# Enable verbose logging
terminal-wakatime config --debug true

# View logs
tail -f ~/.wakatime/terminal-wakatime.log
```

### Debug Information

```bash
# System information
terminal-wakatime debug --system

# Shell environment
terminal-wakatime debug --shell

# Recent heartbeats
terminal-wakatime debug --heartbeats
```

## Privacy

Terminal WakaTime respects your privacy:

- **No file contents** are ever transmitted
- Only metadata (file paths, timestamps, languages) is sent
- File paths can be obfuscated via configuration
- All data transmission uses HTTPS
- Local activity is stored encrypted

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- 📧 Email: [support@wakatime.com](mailto:support@wakatime.com)
- 🐛 Issues: [GitHub Issues](https://github.com/hackclub/terminal-wakatime/issues)
- 💬 Discussion: [WakaTime Community](https://github.com/wakatime/wakatime/discussions)
- 📖 Documentation: [wakatime.com/help](https://wakatime.com/help)

## Related Projects

- [WakaTime CLI](https://github.com/wakatime/wakatime-cli) - Core command-line tool
- [WakaTime Plugins](https://wakatime.com/plugins) - Editor and IDE plugins
- [WakaTime API](https://wakatime.com/developers) - API documentation
