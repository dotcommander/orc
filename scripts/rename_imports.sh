#!/bin/bash
# Script to rename all import paths and references to use 'orc'

echo "Updating all references to use 'orc'..."

# Find all Go files and update imports
find . -name "*.go" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/refiner|github.com/dotcommander/orc|g'
find . -name "*.go" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/orchestrator|github.com/dotcommander/orc|g'

# Update go.mod imports in any go.mod files (for submodules if any)
find . -name "go.mod" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/refiner|github.com/dotcommander/orc|g'
find . -name "go.mod" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/orchestrator|github.com/dotcommander/orc|g'

# Update Markdown files
find . -name "*.md" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/refiner|github.com/dotcommander/orc|g'
find . -name "*.md" -type f -print0 | xargs -0 sed -i '' 's|github.com/vampirenirmal/orchestrator|github.com/dotcommander/orc|g'
find . -name "*.md" -type f -print0 | xargs -0 sed -i '' 's|/refiner|/orc|g'
find . -name "*.md" -type f -print0 | xargs -0 sed -i '' 's|Refiner|Orc|g'

# Update YAML files
find . -name "*.yaml" -o -name "*.yml" -type f -print0 | xargs -0 sed -i '' 's|refiner|orc|g'

# Update environment files
find . -name ".env*" -type f -print0 | xargs -0 sed -i '' 's|REFINER_|ORC_|g'
find . -name ".env*" -type f -print0 | xargs -0 sed -i '' 's|ORCHESTRATOR_|ORC_|g'

echo "All references updated successfully!"
echo "Remember to:"
echo "1. Update any external references or documentation"
echo "2. Update CI/CD configurations"
echo "3. Verify all environment variables now use ORC_ prefix"