#!/bin/bash

echo "=== Terminal WakaTime Enhanced Metadata Demo ==="
echo ""

echo "Testing file editing with rich metadata..."
./terminal-wakatime track --command "vim pkg/tracker/tracker.go" --duration 5 --pwd "$(pwd)"

echo ""
echo "Testing git commit with file changes..."
./terminal-wakatime track --command "git commit -m 'add enhanced metadata'" --duration 3 --pwd "$(pwd)"

echo ""
echo "Testing build command with project context..."
./terminal-wakatime track --command "go test ./..." --duration 10 --pwd "$(pwd)"

echo ""
echo "Testing npm command with language detection..."
cd /tmp && mkdir test-js-project && cd test-js-project
echo '{"name": "test", "version": "1.0.0"}' > package.json
./terminal-wakatime track --command "npm test" --duration 5 --pwd "$(pwd)"
cd - && rm -rf /tmp/test-js-project

echo ""
echo "=== Demo Complete ==="
echo "Check your WakaTime dashboard to see the enhanced metadata!"
