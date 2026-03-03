# `terminal-wakatime`

Track your time in bash, zsh, fish, and PowerShell with WakaTime! Better alternative to the WakaTime desktop app. Built by [@zachlatta](https://github.com/zachlatta).

## Quick Setup (30 seconds)

```bash
curl -fsSL http://hack.club/terminal-wakatime.sh | bash
```

PowerShell:

```powershell
irm https://hack.club/terminal-wakatime.ps1 | iex
```

This installs `terminal-wakatime` to `~/.wakatime/terminal-wakatime` and adds shell init to your config (`~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`, or PowerShell profile).

That's it. Your terminal work in **bash, zsh, fish, and PowerShell** now gets properly tracked in WakaTime with correct project detection.

## The Problem with WakaTime Desktop App

You spend 3 hours coding, but WakaTime only shows 2 hours because:

- ❌ **Can't see terminal work** - Git commits, vim editing, testing, debugging = invisible
- ❌ **Wrong project names** - Everything becomes `<<LAST_PROJECT>>`
- ❌ **Misses development work** - `git commit`, `tmux` sessions don't count as coding time

**`terminal-wakatime` fixes all of this.** It properly tracks your terminal sessions with accurate project detection and file-level detail.

## Works With Your Existing Plugins

✅ **Vim plugin** still tracks detailed keystrokes and cursor movement  
✅ **VSCode plugin** still tracks file edits and project switching  
✅ **`terminal-wakatime`** adds the missing terminal sessions, git work, and project context

**No conflicts.** They work together to give you complete tracking.

## Before vs After

**Before `terminal-wakatime` (WakaTime Desktop App Only):**

```
Today's Coding Time: 2h 30m
├── VSCode: 2h 15m (my-website)
└── Terminal: 15m (<<LAST_PROJECT>>)
```

*Missing: 4+ hours of git work, vim editing, testing, and debugging*

**After `terminal-wakatime`:**

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

For Linux/MacOS (Bash):

```bash
curl -fsSL http://hack.club/tw.sh | bash
```

For Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/hackclub/terminal-wakatime/main/install.ps1 | iex
```

### Manual Install

Download from [releases](https://github.com/hackclub/terminal-wakatime/releases), then:

```bash
# Add to your shell config (~/.bashrc, ~/.zshrc, etc.)
eval "$(terminal-wakatime init)"
```

For PowerShell:

```powershell
terminal-wakatime init powershell | Invoke-Expression
```

You can also run the native PowerShell installer script directly:

```powershell
./install.ps1
```

### Package Managers

For all of your favorite package managers don't forget to activate the packge with the following in your shell config:

```bash
eval "$(terminal-wakatime init)"
```

## Go

```bash
# Go
go install github.com/hackclub/terminal-wakatime/cmd/terminal-wakatime@latest
```

### Nix

```bash
# Direct installation with flakes enabled
nix profile install github:hackclub/terminal-wakatime
```

For use in your own flake:

```nix
# In your flake.nix
{
  inputs.terminal-wakatime.url = "github:hackclub/terminal-wakatime";
  
  outputs = { self, nixpkgs, terminal-wakatime, ... }: {
    # Access the package as:
    # terminal-wakatime.packages.${system}.default
  };
}
```

## Configuration

**WakaTime API Key Setup:**

Optionally manually set your API key. If you're using [Hackatime](https://hackatime.hackclub.com), this is already done automatically by the [Hackatime setup script](https://hackatime.hackclub.com/my/wakatime_setup) and you can ignore this.

```bash
terminal-wakatime config --key YOUR_WAKATIME_KEY
```

**Basic Options:**

```bash
# Set custom project name for current directory
terminal-wakatime config --project my-awesome-project

# Test your setup
terminal-wakatime test
```

## How It Works

`terminal-wakatime` hooks into your shell to detect:

1. **When you start working** in a directory (project detection)
2. **What files you're editing** (vim, nano, code commands)  
3. **What tools you're using** (git, npm, python, etc.)

It sends this data to WakaTime using the same format as other plugins, so everything appears seamlessly in your dashboard.

## Editor Plugin Suggestions

When you use editors like `vim` or `code`, `terminal-wakatime` will suggest installing the dedicated plugin for better tracking:

```
💡 You're using Vim! Install vim-wakatime for detailed keystroke tracking:
   https://github.com/wakatime/vim-wakatime
   
   `terminal-wakatime` will still track your session time.
```

You can disable these suggestions:

```bash
terminal-wakatime config --disable-editor-suggestions
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
terminal-wakatime debug

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

**`terminal-wakatime`** hooks directly into your shell (Bash/Zsh/Fish/PowerShell) to track:

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

Built for Hack Club's [Hackatime](https://hackatime.hackclub.com) community, but works with standard WakaTime. Pull requests welcome!

```bash
git clone https://github.com/hackclub/terminal-wakatime
cd terminal-wakatime
go test ./...
```

## Support

- 🐛 [GitHub Issues](https://github.com/hackclub/terminal-wakatime/issues)
- 💬 [Hack Club Slack](https://hackclub.com/slack) `#hackatime-dev` channel
- 📖 [WakaTime Docs](https://wakatime.com/help)
