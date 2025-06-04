#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="hackclub/terminal-wakatime"
BINARY_NAME="terminal-wakatime"
INSTALL_DIR="/usr/local/bin"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux";;
        Darwin*)    os="darwin";;
        CYGWIN*|MINGW*|MSYS*) os="windows";;
        *)          print_error "Unsupported operating system: $(uname -s)"; exit 1;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64";;
        arm64|aarch64)  arch="arm64";;
        *)              print_error "Unsupported architecture: $(uname -m)"; exit 1;;
    esac
    
    # Set platform-specific values
    if [ "$os" = "windows" ]; then
        PLATFORM="${os}-${arch}.exe"
        BINARY_NAME="${BINARY_NAME}.exe"
    else
        PLATFORM="${os}-${arch}"
    fi
    
    print_status "Detected platform: $PLATFORM"
}

# Function to get latest release version
get_latest_version() {
    print_status "Fetching latest release information..."
    
    # Try to get latest release from GitHub API
    if command -v curl >/dev/null 2>&1; then
        LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | cut -d '"' -f 4)
    elif command -v wget >/dev/null 2>&1; then
        LATEST_VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | cut -d '"' -f 4)
    else
        print_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
    
    if [ -z "$LATEST_VERSION" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi
    
    print_status "Latest version: $LATEST_VERSION"
}

# Function to download binary
download_binary() {
    local download_url="https://github.com/$REPO/releases/download/$LATEST_VERSION/terminal-wakatime-$PLATFORM"
    local temp_file="/tmp/$BINARY_NAME"
    
    print_status "Downloading from: $download_url"
    
    if command -v curl >/dev/null 2>&1; then
        curl -L "$download_url" -o "$temp_file"
    elif command -v wget >/dev/null 2>&1; then
        wget "$download_url" -O "$temp_file"
    else
        print_error "Neither curl nor wget is available"
        exit 1
    fi
    
    if [ ! -f "$temp_file" ]; then
        print_error "Download failed"
        exit 1
    fi
    
    chmod +x "$temp_file"
    print_success "Binary downloaded successfully"
}

# Function to install binary
install_binary() {
    local temp_file="/tmp/$BINARY_NAME"
    
    # Try to install to /usr/local/bin first
    if [ -w "$INSTALL_DIR" ] || sudo -n true 2>/dev/null; then
        print_status "Installing to $INSTALL_DIR..."
        if [ -w "$INSTALL_DIR" ]; then
            mv "$temp_file" "$INSTALL_DIR/$BINARY_NAME"
        else
            sudo mv "$temp_file" "$INSTALL_DIR/$BINARY_NAME"
        fi
        print_success "Installed to $INSTALL_DIR/$BINARY_NAME"
    else
        # Fallback to user's local bin directory
        local user_bin="$HOME/.local/bin"
        mkdir -p "$user_bin"
        mv "$temp_file" "$user_bin/$BINARY_NAME"
        print_success "Installed to $user_bin/$BINARY_NAME"
        
        # Check if user's bin is in PATH
        if [[ ":$PATH:" != *":$user_bin:"* ]]; then
            print_warning "Add $user_bin to your PATH by adding this line to your shell config:"
            echo "export PATH=\"$user_bin:\$PATH\""
        fi
    fi
}

# Function to detect shell
detect_shell() {
    if [ -n "$ZSH_VERSION" ]; then
        echo "zsh"
    elif [ -n "$BASH_VERSION" ]; then
        echo "bash"
    elif [ -n "$FISH_VERSION" ]; then
        echo "fish"
    else
        # Fallback to $SHELL environment variable
        basename "${SHELL:-bash}"
    fi
}

# Function to get shell config file
get_shell_config() {
    local shell_name="$1"
    case "$shell_name" in
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                echo "$HOME/.bashrc"
            else
                echo "$HOME/.bash_profile"
            fi
            ;;
        zsh)
            echo "$HOME/.zshrc"
            ;;
        fish)
            echo "$HOME/.config/fish/config.fish"
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

# Function to setup shell integration for a specific config file
setup_shell_config() {
    local config_file="$1"
    local shell_type="$2"
    
    # Create config directory if it doesn't exist (for fish)
    mkdir -p "$(dirname "$config_file")"
    
    # Check if already configured
    if grep -q "terminal-wakatime init" "$config_file" 2>/dev/null; then
        print_warning "Already configured in $config_file"
        return 1
    fi
    
    # Add integration based on shell type
    case "$shell_type" in
        fish)
            echo "" >> "$config_file"
            echo "# Terminal WakaTime integration" >> "$config_file"
            echo "terminal-wakatime init | source" >> "$config_file"
            ;;
        *)
            echo "" >> "$config_file"
            echo "# Terminal WakaTime integration" >> "$config_file"
            echo 'eval "$(terminal-wakatime init)"' >> "$config_file"
            ;;
    esac
    
    print_success "Added integration to $config_file"
    
    # Try to source the config file to make it immediately available
    case "$shell_type" in
        fish)
            # Fish uses a different sourcing method
            if [ "$FISH_VERSION" ]; then
                fish -c "source $config_file" 2>/dev/null || true
            fi
            ;;
        *)
            # For bash/zsh, try to source if we're in the right shell
            if [ "$BASH_VERSION" ] || [ "$ZSH_VERSION" ]; then
                source "$config_file" 2>/dev/null || true
                print_status "Sourced $config_file for immediate use"
            fi
            ;;
    esac
    
    return 0
}

# Function to find the best config file for a shell type
find_best_config() {
    local shell_type="$1"
    local best_file=""
    local best_size=0
    
    case "$shell_type" in
        bash)
            local configs=("$HOME/.bashrc" "$HOME/.bash_profile")
            ;;
        zsh)
            local configs=("$HOME/.zshrc")
            ;;
        fish)
            local configs=("$HOME/.config/fish/config.fish")
            ;;
    esac
    
    # Find the config file with the most content
    for config_file in "${configs[@]}"; do
        if [ -f "$config_file" ]; then
            local file_size=$(wc -c < "$config_file" 2>/dev/null || echo 0)
            if [ "$file_size" -gt "$best_size" ]; then
                best_size="$file_size"
                best_file="$config_file"
            fi
        fi
    done
    
    # If no existing files found, use the primary default
    if [ -z "$best_file" ]; then
        case "$shell_type" in
            bash)
                best_file="$HOME/.bashrc"
                ;;
            zsh)
                best_file="$HOME/.zshrc"
                ;;
            fish)
                best_file="$HOME/.config/fish/config.fish"
                ;;
        esac
    fi
    
    echo "$best_file"
}

# Function to setup shell integration for all available shells
setup_shell_integration() {
    print_status "Setting up shell integration for all supported shells..."
    
    local configured_count=0
    local shell_types=("bash" "zsh" "fish")
    
    for shell_type in "${shell_types[@]}"; do
        local config_file=$(find_best_config "$shell_type")
        
        # Check if this config file exists and has content, or if it's a primary config
        local should_configure=false
        if [ -f "$config_file" ]; then
            local file_size=$(wc -c < "$config_file" 2>/dev/null || echo 0)
            if [ "$file_size" -gt 0 ]; then
                should_configure=true
                print_status "Found active $shell_type config: $config_file (${file_size} bytes)"
            fi
        else
            # Always configure primary shell configs even if they don't exist
            case "$config_file" in
                "$HOME/.bashrc"|"$HOME/.zshrc"|"$HOME/.config/fish/config.fish")
                    should_configure=true
                    print_status "Will create $shell_type config: $config_file"
                    ;;
            esac
        fi
        
        if [ "$should_configure" = true ]; then
            if setup_shell_config "$config_file" "$shell_type"; then
                ((configured_count++))
            fi
        fi
    done
    
    if [ $configured_count -eq 0 ]; then
        print_warning "No shell configurations were modified (already configured or no shells found)"
    else
        print_success "Configured $configured_count shell(s)"
        print_status "Shell integration is now active in your current session!"
    fi
}

# Function to guide API key setup
setup_api_key() {
    # Check if API key is already configured
    if command -v terminal-wakatime >/dev/null 2>&1; then
        local api_key_status=$(terminal-wakatime config --show 2>/dev/null | grep "API Key:" | cut -d' ' -f3-)
        if [ "$api_key_status" != "(not set)" ] && [ -n "$api_key_status" ]; then
            print_success "API key is already configured!"
            print_success "Setup complete! Terminal WakaTime is ready to use."
            return
        fi
    fi
    
    print_status "Setting up WakaTime API key..."
    echo ""
    echo "To complete setup, you need a WakaTime API key:"
    echo "1. Visit: https://wakatime.com/api-key"
    echo "2. Copy your API key"
    echo "3. Run: terminal-wakatime config --key YOUR_API_KEY"
    echo ""
    print_success "Setup complete! Run the config command above to start tracking."
}

# Main installation flow
main() {
    echo "🚀 Terminal WakaTime Installer"
    echo "=============================="
    echo ""
    
    # Check if already installed
    if command -v terminal-wakatime >/dev/null 2>&1; then
        print_warning "Terminal WakaTime is already installed at: $(which terminal-wakatime)"
        read -p "Do you want to reinstall? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 0
        fi
    fi
    
    detect_platform
    get_latest_version
    download_binary
    install_binary
    setup_shell_integration
    setup_api_key
    
    echo ""
    print_success "Installation completed successfully! 🎉"
}

# Run installer
main "$@"
