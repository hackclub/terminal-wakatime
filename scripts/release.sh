#!/bin/bash
set -e

# Release script for terminal-wakatime
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh v1.0.0

if [ $# -eq 0 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.0.0"
    exit 1
fi

VERSION=$1

# Validate version format (should start with 'v')
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version should be in format vX.Y.Z (e.g., v1.0.0)"
    exit 1
fi

# Check if we're on main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "Error: Must be on main branch to create release"
    exit 1
fi

# Check if working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: Working directory is not clean. Please commit or stash changes."
    exit 1
fi

# Pull latest changes
echo "Pulling latest changes..."
git pull origin main

# Check if tag already exists
if git tag -l | grep -q "^$VERSION$"; then
    echo "Error: Tag $VERSION already exists"
    exit 1
fi

# Run tests to make sure everything works
echo "Running tests..."
make test

# Build to verify it compiles
echo "Building..."
make build

# Show current version
echo "Current version:"
./terminal-wakatime version

echo ""
echo "Creating release $VERSION..."
echo "This will:"
echo "1. Create and push git tag $VERSION"
echo "2. Trigger GitHub Actions to build and create release"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 1
fi

# Create and push tag
echo "Creating tag $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"

echo "Pushing tag..."
git push origin "$VERSION"

echo ""
echo "✓ Release $VERSION created!"
echo "✓ GitHub Actions will build binaries and create release"
echo "✓ Check: https://github.com/hackclub/terminal-wakatime/actions"
echo "✓ Release: https://github.com/hackclub/terminal-wakatime/releases"
