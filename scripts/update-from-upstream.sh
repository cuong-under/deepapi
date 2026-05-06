#!/bin/bash
# DS2API Update Script - Vibecode Vietnam Edition
# Updates DS2API from upstream while preserving customizations

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Banner
echo -e "${CYAN}"
echo "╔════════════════════════════════════════════════════════════╗"
echo "║         DS2API Update Script - Vibecode Vietnam           ║"
echo "║              Preserve Customizations on Update            ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "internal" ]; then
    echo -e "${RED}❌ Error: Not in DS2API root directory${NC}"
    echo "Please run this script from the DS2API root directory"
    exit 1
fi

# Check if we're on custom-vibecode branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "custom-vibecode" ]; then
    echo -e "${YELLOW}⚠️  Warning: Not on custom-vibecode branch${NC}"
    echo "Current branch: $CURRENT_BRANCH"
    read -p "Switch to custom-vibecode branch? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git checkout custom-vibecode
    else
        echo -e "${RED}Aborted${NC}"
        exit 1
    fi
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}⚠️  Warning: You have uncommitted changes${NC}"
    git status --short
    echo
    read -p "Commit changes before updating? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git add -A
        read -p "Enter commit message: " COMMIT_MSG
        git commit -m "$COMMIT_MSG"
    else
        echo -e "${YELLOW}Continuing with uncommitted changes...${NC}"
    fi
fi

# Fetch upstream
echo -e "${BLUE}📡 Fetching upstream changes...${NC}"
git fetch origin

# Show what will be merged
echo
echo -e "${CYAN}📋 Changes in upstream since last update:${NC}"
UPSTREAM_COMMITS=$(git log HEAD..origin/main --oneline | head -20)
if [ -z "$UPSTREAM_COMMITS" ]; then
    echo -e "${GREEN}✅ Already up to date!${NC}"
    exit 0
fi

echo "$UPSTREAM_COMMITS"
echo
echo -e "${YELLOW}Total commits to merge: $(git rev-list --count HEAD..origin/main)${NC}"
echo

# Ask for confirmation
read -p "Continue with merge? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Update cancelled${NC}"
    exit 0
fi

# Create backup branch
BACKUP_BRANCH="backup-$(date +%Y%m%d-%H%M%S)"
echo -e "${BLUE}💾 Creating backup branch: $BACKUP_BRANCH${NC}"
git branch $BACKUP_BRANCH

# Merge upstream
echo -e "${BLUE}🔄 Merging upstream changes...${NC}"
if git merge origin/main --no-edit; then
    echo -e "${GREEN}✅ Merge successful!${NC}"
else
    echo -e "${RED}⚠️  Merge conflicts detected${NC}"
    echo
    echo -e "${YELLOW}Conflicted files:${NC}"
    git diff --name-only --diff-filter=U
    echo
    echo -e "${CYAN}Resolution steps:${NC}"
    echo "1. Review conflicts in the files above"
    echo "2. Edit conflicted files to resolve"
    echo "3. Run: git add <file>"
    echo "4. Run: git commit"
    echo
    echo -e "${YELLOW}Common conflicts and how to resolve:${NC}"
    echo "- internal/server/router.go: Keep plugin manager initialization"
    echo "- internal/config/config.go: Keep PricingConfig struct"
    echo "- webui/src/layout/DashboardShell.jsx: Keep Vibecode branding"
    echo
    echo -e "${BLUE}Backup branch created: $BACKUP_BRANCH${NC}"
    echo "To rollback: git reset --hard $BACKUP_BRANCH"
    exit 1
fi

# Build and test
echo
echo -e "${BLUE}🔨 Building project...${NC}"
if go build ./cmd/ds2api; then
    echo -e "${GREEN}✅ Build successful${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    echo "Rolling back..."
    git reset --hard $BACKUP_BRANCH
    exit 1
fi

# Test server startup
echo
echo -e "${BLUE}🧪 Testing server startup...${NC}"
timeout 3 ./ds2api.exe > /dev/null 2>&1 || true
if [ -f "ds2api.exe" ]; then
    echo -e "${GREEN}✅ Server binary created${NC}"
else
    echo -e "${RED}❌ Server binary not found${NC}"
    exit 1
fi

# Summary
echo
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                  ✅ Update Complete!                       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo
echo -e "${CYAN}📊 Summary:${NC}"
echo "- Merged $(git rev-list --count $BACKUP_BRANCH..HEAD) commits from upstream"
echo "- Backup branch: $BACKUP_BRANCH"
echo "- Current branch: $(git branch --show-current)"
echo
echo -e "${YELLOW}⚠️  Next steps:${NC}"
echo "1. Test all features thoroughly:"
echo "   - Analytics dashboard"
echo "   - Plugin system"
echo "   - Vietnamese translations"
echo "   - Cyberpunk theme"
echo "2. If everything works: git branch -D $BACKUP_BRANCH"
echo "3. If issues found: git reset --hard $BACKUP_BRANCH"
echo
echo -e "${CYAN}🚀 To start server: ./ds2api.exe${NC}"
echo -e "${CYAN}📝 To view changes: git log $BACKUP_BRANCH..HEAD${NC}"
echo
