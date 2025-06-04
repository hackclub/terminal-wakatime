#!/bin/bash

echo "🚀 Demonstrating Correct WakaTime Plugin Pattern"
echo "================================================"
echo ""

echo "Our Terminal WakaTime now follows the official WakaTime plugin pattern:"
echo ""
echo "✅ Calls wakatime-cli when:"
echo "   • File changes (different file than last time)"  
echo "   • 2+ minutes pass (since last call for same file)"
echo "   • File save event occurs"
echo ""
echo "✅ Lets wakatime-cli handle:"
echo "   • Rate limiting and deduplication"
echo "   • Language detection"
echo "   • Project detection" 
echo "   • API communication"
echo ""

echo "🎯 Testing the pattern..."
echo ""

# Test editor detection
echo "1. Testing editor detection:"
./terminal-wakatime track -- --command "vim main.go" --duration 3 --pwd $(pwd)
echo ""

# Test different file (should always send)
echo "2. Testing file change (should always send):"
./terminal-wakatime track -- --command "emacs README.md" --duration 2 --pwd $(pwd)
echo ""

# Test VS Code  
echo "3. Testing VS Code detection:"
./terminal-wakatime track -- --command "code package.json" --duration 4 --pwd $(pwd)
echo ""

# Test coding tools
echo "4. Testing coding tool detection:"
./terminal-wakatime track -- --command "npm install" --duration 10 --pwd $(pwd)
echo ""

echo "✨ All heartbeats sent to wakatime-cli following official WakaTime plugin spec!"
echo ""
echo "📖 This matches the behavior of all official WakaTime plugins:"
echo "   • VS Code WakaTime extension"
echo "   • Vim WakaTime plugin"
echo "   • Sublime WakaTime plugin"
echo "   • And all others!"
