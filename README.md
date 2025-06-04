# Terminal WakaTime for Hackatime

Stop losing coding time to invisible terminal work.

## Quick Setup (30 seconds)

```bash
curl -fsSL http://hack.club/tw.sh | sh
```

That's it. Everything in your terminal now counts toward your Hackatime.

## What This Solves

You spend hours in terminal doing real coding work - git commits, vim editing, running tests, debugging - but Hackatime only sees "2 hours of coding" when you actually coded for 6 hours.

**Terminal WakaTime fills the gap.** Now your terminal work gets properly tracked with correct project names and file details.

## Works With Your Existing Plugins

✅ **Vim plugin** still tracks detailed keystrokes and cursor movement  
✅ **VSCode plugin** still tracks file edits and project switching  
✅ **Terminal WakaTime** adds the missing terminal sessions, git work, and project context

**No conflicts.** They work together to give you complete tracking.

## Before vs After

**Before Terminal WakaTime:**
```
Today's Coding Time: 2h 30m
├── VSCode: 2h 15m (my-website)
└── Unknown: 15m
```

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

**Hackatime API Key Setup:**
```bash
terminal-wakatime config --key YOUR_HACKATIME_KEY
```
Get your key from: [hackatime.hackclub.com/settings](https://hackatime.hackclub.com/settings)

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

It sends this data to Hackatime using the same format as other plugins, so everything appears seamlessly in your dashboard.

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

## Privacy

- No file contents are ever sent
- Only file paths, timestamps, and metadata
- All data encrypted in transit
- Same privacy model as other Hackatime plugins

## Contributing

Built for Hack Club's Hackatime community. Pull requests welcome!

```bash
git clone https://github.com/hackclub/terminal-wakatime
cd terminal-wakatime
go test ./...
```

## Support

- 🐛 [GitHub Issues](https://github.com/hackclub/terminal-wakatime/issues)
- 💬 [Hack Club Slack](https://hackclub.com/slack) #hackatime channel
- 📖 [Hackatime Docs](https://hackatime.hackclub.com/help)
