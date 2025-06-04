# Terminal WakaTime

**The better way to track your coding time in WakaTime.**

Stop losing hours of real coding work because WakaTime's desktop app can't see what you're doing in your terminal.

## Quick Setup (30 seconds)

```bash
curl -fsSL http://hack.club/tw.sh | sh
```

That's it. Your terminal work in **Bash, Zsh, and Fish** now gets properly tracked in WakaTime with correct project detection.

## The Problem with WakaTime Desktop App

You spend 6 hours coding, but WakaTime only shows 2 hours because:

- ❌ **Can't see terminal work** - Git commits, vim editing, testing, debugging = invisible
- ❌ **Wrong project names** - Everything becomes "Terminal" or "Unknown" 
- ❌ **Misses file editing** - `vim src/app.js` doesn't count as coding time
- ❌ **No shell integration** - Your actual development workflow is ignored

**Terminal WakaTime fixes all of this.** It properly tracks your terminal sessions with accurate project detection and file-level detail.

## Works With Your Existing Plugins

✅ **Vim plugin** still tracks detailed keystrokes and cursor movement  
✅ **VSCode plugin** still tracks file edits and project switching  
✅ **Terminal WakaTime** adds the missing terminal sessions, git work, and project context

**No conflicts.** They work together to give you complete tracking.

## Before vs After

**Before Terminal WakaTime (WakaTime Desktop App Only):**
```
Today's Coding Time: 2h 30m
├── VSCode: 2h 15m (my-website)
└── Terminal: 15m (Unknown)
```
*Missing: 4+ hours of git work, vim editing, testing, and debugging*

**After Terminal WakaTime:**
```
Today's Coding Time: 6h 45m
├── VSCode: 2h 15m (my-website) 
├── Terminal: 3h 30m (my-website)
│   ├── Git operations: 45m
│   ├── Vim editing: 2h 10m  
│   └── Testing/debugging: 35m
└── Terminal: 1h (hackclub-bot)
    └── Python scripting: 1h
```
*Now tracking your complete development workflow with correct project names*

## What Gets Tracked

**Files & Editing:**
- `vim src/app.js` → Tracks file editing time in correct project
- `nano README.md` → Counts toward your coding time
- File saves and project switching

**Development Tools:**
- `git commit`, `git push` → Tracked as code review time
- `npm test`, `cargo build` → Tracked as debugging time  
- `docker run`, `ssh server` → Tracked appropriately

**Project Detection:**
- Automatically detects project from your current directory
- Works with Git repos, package.json, Cargo.toml, etc.
- No more "Unknown Project" in your stats

## Installation Options

### Quick Install (Recommended)
```bash
curl -fsSL http://hack.club/tw.sh | sh
```

### Manual Install
Download from [releases](https://github.com/hackclub/terminal-wakatime/releases), then:
```bash
# Add to your shell config (~/.bashrc, ~/.zshrc, etc.)
eval "$(terminal-wakatime init)"
```

### Package Managers
```bash
# Go
go install github.com/hackclub/terminal-wakatime/cmd/terminal-wakatime@latest
```

## Configuration

**WakaTime API Key Setup:**
```bash
terminal-wakatime config --key YOUR_WAKATIME_KEY
```
Get your key from: [wakatime.com/api-key](https://wakatime.com/api-key)

**Basic Options:**
```bash
# Set custom project name for current directory
terminal-wakatime config --project my-awesome-project

# Test your setup
terminal-wakatime test
```

## How It Works

Terminal WakaTime hooks into your shell to detect:
1. **When you start working** in a directory (project detection)
2. **What files you're editing** (vim, nano, code commands)  
3. **What tools you're using** (git, npm, python, etc.)

It sends this data to WakaTime using the same format as other plugins, so everything appears seamlessly in your dashboard.

## Editor Plugin Suggestions

When you use editors like `vim` or `code`, Terminal WakaTime will suggest installing the dedicated plugin for better tracking:

```
💡 You're using Vim! Install vim-wakatime for detailed keystroke tracking:
   https://github.com/wakatime/vim-wakatime
   
   Terminal WakaTime will still track your session time.
```

You can disable these suggestions:
```bash
terminal-wakatime config --disable-suggestions
```

## Troubleshooting

**Not tracking activity?**
```bash
# Check if properly installed
echo $PROMPT_COMMAND  # Should show terminal-wakatime

# Verify API key
terminal-wakatime config --show

# Test connection
terminal-wakatime test
```

**Wrong project names?**
```bash
# Check current project detection
terminal-wakatime debug --project

# Manually set project for this directory
terminal-wakatime config --project my-project
```

**Issues with dependencies?**
```bash
# Check wakatime-cli status
terminal-wakatime deps --status

# Reinstall if needed
terminal-wakatime deps --reinstall
```

## Why Not Just Use WakaTime Desktop App?

**WakaTime Desktop App** only tracks window focus - it has no idea what you're actually doing in your terminal. When you're deep in a coding session doing `git commits`, `vim editing`, `npm test`, it just sees "Terminal app is open" with no context.

**Terminal WakaTime** hooks directly into your shell (Bash/Zsh/Fish) to track:
- ✅ Actual commands and file editing
- ✅ Correct project detection from your current directory  
- ✅ Real coding time vs just having terminal open
- ✅ Works alongside your existing editor plugins

## Privacy

- No file contents are ever sent
- Only file paths, timestamps, and metadata
- All data encrypted in transit
- Same privacy model as other WakaTime plugins

## Contributing

Built for Hack Club's Hackatime community, but works with standard WakaTime. Pull requests welcome!

```bash
git clone https://github.com/hackclub/terminal-wakatime
cd terminal-wakatime
go test ./...
```

## Support

- 🐛 [GitHub Issues](https://github.com/hackclub/terminal-wakatime/issues)
- 💬 [Hack Club Slack](https://hackclub.com/slack) #hackatime channel
- 📖 [WakaTime Docs](https://wakatime.com/help)
