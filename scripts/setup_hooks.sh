#!/bin/bash
# HotPlex Git Hooks Installer
# Links scripts from /scripts to .git/hooks for a consistent dev experience

set -e

# Get repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
HOOK_SOURCE_DIR="$REPO_ROOT/scripts"
HOOK_TARGET_DIR="$REPO_ROOT/.git/hooks"

# ANSI colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔗 Installing HotPlex Git Hooks...${NC}"

HOOKS=("pre-commit" "commit-msg" "pre-push")

for hook in "${HOOKS[@]}"; do
    if [ -f "$HOOK_SOURCE_DIR/$hook" ]; then
        # Ensure executable
        chmod +x "$HOOK_SOURCE_DIR/$hook"
        # Create symbolic link
        ln -sf "$HOOK_SOURCE_DIR/$hook" "$HOOK_TARGET_DIR/$hook"
        echo -e "${GREEN}✅ Linked: $hook${NC}"
    else
        echo "⚠️  Skip: $hook (not found in $HOOK_SOURCE_DIR)"
    fi
done

echo -e "${BLUE}Done! Hooks are now active.${NC}"
