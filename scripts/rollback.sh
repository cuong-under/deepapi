#!/bin/bash
# DS2API Rollback Script - Vibecode Vietnam Edition
# Rollback to previous state if update fails

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Banner
echo -e "${RED}"
echo "╔════════════════════════════════════════════════════════════╗"
echo "║         DS2API Rollback Script - Vibecode Vietnam         ║"
echo "║              Restore Previous Working State               ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "internal" ]; then
    echo -e "${RED}❌ Error: Not in DS2API root directory${NC}"
    exit 1
fi

# List available backup branches
echo -e "${CYAN}📋 Available backup branches:${NC}"
BACKUP_BRANCHES=$(git branch | grep "backup-" | sed 's/^[ *]*//')

if [ -z "$BACKUP_BRANCHES" ]; then
    echo -e "${YELLOW}No backup branches found${NC}"
    echo
    echo "Backup branches are created automatically by update-from-upstream.sh"
    echo "Format: backup-YYYYMMDD-HHMMSS"
    exit 1
fi

echo "$BACKUP_BRANCHES"
echo

# Get current branch
CURRENT_BRANCH=$(git branch --show-current)
echo -e "${BLUE}Current branch: $CURRENT_BRANCH${NC}"
echo

# Ask which backup to restore
read -p "Enter backup branch name to restore (or 'latest' for most recent): " BACKUP_BRANCH

if [ "$BACKUP_BRANCH" = "latest" ]; then
    BACKUP_BRANCH=$(echo "$BACKUP_BRANCHES" | tail -1)
    echo -e "${CYAN}Selected: $BACKUP_BRANCH${NC}"
fi

# Verify backup branch exists
if ! git rev-parse --verify "$BACKUP_BRANCH" > /dev/null 2>&1; then
    echo -e "${RED}❌ Error: Backup branch '$BACKUP_BRANCH' not found${NC}"
    exit 1
fi

# Show what will be lost
echo
echo -e "${YELLOW}⚠️  Warning: This will discard all changes since backup${NC}"
echo
echo -e "${CYAN}Changes that will be lost:${NC}"
git log --oneline $BACKUP_BRANCH..HEAD | head -10
echo

# Confirm rollback
read -p "Are you sure you want to rollback? This cannot be undone! (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo -e "${YELLOW}Rollback cancelled${NC}"
    exit 0
fi

# Create safety backup of current state
SAFETY_BACKUP="safety-backup-$(date +%Y%m%d-%H%M%S)"
echo -e "${BLUE}💾 Creating safety backup: $SAFETY_BACKUP${NC}"
git branch $SAFETY_BACKUP

# Perform rollback
echo -e "${BLUE}⏮️  Rolling back to $BACKUP_BRANCH...${NC}"
git reset --hard $BACKUP_BRANCH

# Clean untracked files
echo -e "${BLUE}🧹 Cleaning untracked files...${NC}"
git clean -fd

# Rebuild
echo -e "${BLUE}🔨 Rebuilding project...${NC}"
if go build ./cmd/ds2api; then
    echo -e "${GREEN}✅ Build successful${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    echo "Restoring to safety backup..."
    git reset --hard $SAFETY_BACKUP
    exit 1
fi

# Summary
echo
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                 ✅ Rollback Complete!                      ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo
echo -e "${CYAN}📊 Summary:${NC}"
echo "- Rolled back to: $BACKUP_BRANCH"
echo "- Safety backup created: $SAFETY_BACKUP"
echo "- Current branch: $(git branch --show-current)"
echo
echo -e "${YELLOW}⚠️  Next steps:${NC}"
echo "1. Test that everything works"
echo "2. If satisfied, delete safety backup: git branch -D $SAFETY_BACKUP"
echo "3. If issues persist, contact support"
echo
echo -e "${CYAN}🚀 To start server: ./ds2api.exe${NC}"
echo
