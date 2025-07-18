#!/bin/bash
# Script to rename all import paths from refiner to orchestrator

echo "Updating import paths from refiner to orchestrator..."

# Find all Go files and update imports
find . -name "*.go" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/refiner|github.com/vampirenirmal/orchestrator|g'

# Update go.mod imports in any go.mod files (for submodules if any)
find . -name "go.mod" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/refiner|github.com/vampirenirmal/orchestrator|g'

# Update Markdown files
find . -name "*.md" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/refiner|github.com/vampirenirmal/orchestrator|g'
find . -name "*.md" -type f -print0 | xargs -0 sed -i '' 's|/refiner|/orchestrator|g'
find . -name "*.md" -type f -print0 | xargs -0 sed -i '' 's|Refiner|The Orchestrator|g'
find . -name "*.md" -type f -print0 | xargs -0 sed -i '' 's|refiner|orc|g'

# Update YAML files
find . -name "*.yaml" -o -name "*.yml" -type f -print0 | xargs -0 sed -i '' 's|refiner|orchestrator|g'

# Update environment files
find . -name ".env*" -type f -print0 | xargs -0 sed -i '' 's|REFINER_|ORCHESTRATOR_|g'

echo "Import paths updated successfully!"
echo "Remember to:"
echo "1. Rename the project directory from 'refiner' to 'orchestrator'"
echo "2. Update any external references or documentation"
echo "3. Update CI/CD configurations"