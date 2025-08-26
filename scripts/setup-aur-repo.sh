#!/bin/bash

# Script to set up the AUR repository
# Usage: ./scripts/setup-aur-repo.sh

set -euo pipefail

echo "Setting up AUR repository for projector-cli..."

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed."
    echo "Please install it first: https://cli.github.com/"
    exit 1
fi

# Check if user is authenticated
if ! gh auth status &> /dev/null; then
    echo "Error: Not authenticated with GitHub CLI."
    echo "Please run: gh auth login"
    exit 1
fi

echo "Creating AUR repository: joelgrimberg/projector-cli-aur"

# Create the repository
gh repo create joelgrimberg/projector-cli-aur \
    --public \
    --description "AUR package for projector-cli - automatically updated by CI/CD" \
    --clone

echo "Repository created successfully!"
echo ""
echo "Next steps:"
echo "1. Add AUR_TOKEN secret to your main repository:"
echo "   - Go to: https://github.com/joelgrimberg/projector/settings/secrets/actions"
echo "   - Add new secret: AUR_TOKEN"
echo "   - Value: Create a Personal Access Token with 'repo' scope"
echo ""
echo "2. The workflow will now automatically update this AUR repository with each release!"
echo ""
echo "3. Users can install with: yay -S projector-cli"
