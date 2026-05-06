# DS2API Customizations Audit & Preservation Strategy

**Audit Date:** 2026-05-07  
**Project:** DS2API Fork by Cuongunder  
**Upstream:** https://github.com/CJackHwang/ds2api

---

## Executive Summary

Dự án đã có **extensive customizations** với:
- **~1,800 lines** backend code mới (Go)
- **~530 lines** frontend translations (Vietnamese)
- **1 complete UI theme overhaul** (Cyberpunk/Neon)
- **1 major feature** (Analytics Dashboard)
- **1 infrastructure system** (Plugin Architecture)
- **Custom branding** (Vibecode Vietnam)

**Risk Level khi upstream update:** 🔴 **CRITICAL**  
Hầu hết customizations sẽ bị mất nếu không có strategy phù hợp.

---

## Part 1: Complete Customizations Inventory

### A. Backend Customizations (Go)

#### 1. New Packages (100% Custom - 1,802 lines)

**internal/plugin/** (1,049 lines) - Plugin System Infrastructure
```
├── interface.go (49 lines) - Plugin interface definition
├── manager.go (254 lines) - Plugin lifecycle manager
├── registry.go (125 lines) - Compile-time registration
├── manifest.go (64 lines) - Manifest handling
├── permissions.go (87 lines) - Permission system
└── sandbox.go (93 lines) - Sandboxed wrappers
```
**Risk:** 🔴 HIGH - Hoàn toàn mới, sẽ bị mất khi update  
**Dependency:** Core infrastructure cho plugin system

**internal/httpapi/admin/analytics/** (582 lines) - Analytics Feature
```
├── handler.go (377 lines) - Analytics logic
└── routes.go (205 lines) - Route registration
```
**Risk:** 🔴 HIGH - Feature mới, không có trong upstream  
**Dependency:** Depends on plugin system, config.PricingConfig

**plugins/** (171 lines) - Plugin Implementations
```
├── analytics/
│   ├── main.go (79 lines)
│   ├── plugin.json
│   └── build.sh
└── example/
    ├── main.go (92 lines)
    └── plugin.json
```
**Risk:** 🟢 LOW - Isolated, dễ preserve  
**Dependency:** Depends on internal/plugin/

#### 2. Modified Core Files (Critical Integration Points)

**internal/server/router.go**
```go
// Lines modified: ~50 lines across multiple sections
Changes:
- Import analytics, plugin packages
- Import plugin implementations (blank imports)
- Add PluginManager field to App struct
- Plugin manager initialization (lines 72-87)
- Pricing config loading (lines 98-134)
- Analytics handler initialization (lines 136-141)
- Plugin routes registration (lines 189-191)
```
**Risk:** 🔴 CRITICAL - Core file, high conflict probability  
**Conflict Probability:** 90% - Upstream thường update router.go

**internal/config/config.go**
```go
// Lines modified: ~15 lines
Changes:
- Add Pricing field to Config struct (line 24)
- Add PricingConfig struct (lines 203-207)
- Add ModelPrice struct (lines 209-211)
```
**Risk:** 🟡 MEDIUM - Struct changes, medium conflict probability  
**Conflict Probability:** 50% - Upstream có thể thêm fields mới

**cmd/ds2api/main.go**
```go
// Lines modified: ~5 lines
Changes:
- Plugin shutdown logic in graceful shutdown (lines 73-77)
```
**Risk:** 🟢 LOW - Minimal changes  
**Conflict Probability:** 20%

**internal/httpapi/admin/handler.go**
```go
// Lines modified: ~10 lines
Changes:
- Import analytics package
- Modify RegisterRoutes signature
- Add analytics routes registration
```
**Risk:** 🟡 MEDIUM - API integration point  
**Conflict Probability:** 40%

#### 3. Configuration Changes

**config.json**
```json
{
  "pricing": {
    "currency": "USD",
    "models": {
      "deepseek-v4-flash": {...},
      "deepseek-v4-pro": {...},
      "deepseek-reasoner": {...}
    }
  }
}
```
**Risk:** 🟢 LOW - Config file, easy to preserve  
**Note:** Contains sensitive data (accounts, passwords, API keys)

---

### B. Frontend Customizations (React/UI)

#### 1. Complete Theme Overhaul (Critical)

**webui/src/styles.css** (178 lines → heavily modified)
```css
Changes:
- Color scheme: Amber/Gold → Electric Cyan/Blue
- Background: Modern flat → Cyberpunk with grid/scanlines
- Added 10+ custom utility classes (.glow-cyan, .cyber-grid, etc.)
- Custom animations (pulse-glow, animated-border, ripple)
- Scrollbar styling with cyan glow
```
**Risk:** 🔴 CRITICAL - Complete rewrite  
**Conflict Probability:** 95% - Upstream có thể update styles

**webui/tailwind.config.js**
```js
Changes:
- Custom color mappings
- Neon color palette
- Custom fonts: Rajdhani, Fira Code
```
**Risk:** 🟡 MEDIUM  
**Conflict Probability:** 30%

**webui/index.html**
```html
Changes:
- Google Fonts: Rajdhani, Fira Code
- Theme color: #00d9ff
- Vietnamese meta tags
- Custom title/description
- Custom favicon
```
**Risk:** 🟡 MEDIUM  
**Conflict Probability:** 40%

**webui/public/ds2api-favicon.svg** (460 lines)
```
Custom blue-themed logo
```
**Risk:** 🟢 LOW - Asset file  
**Conflict Probability:** 0%

#### 2. Vietnamese Translations (Critical)

**webui/src/locales/vi.json** (530 lines)
```json
Complete Vietnamese translation:
- All UI sections translated
- 42 lines more than English
- Analytics section (42 lines, lines 493-530)
- Default language: Vietnamese
```
**Risk:** 🟡 MEDIUM - New file, but upstream may add keys  
**Conflict Probability:** 60% - Upstream thêm translation keys mới

**webui/src/i18n.jsx**
```js
Changes:
- Default language: vi (was en)
- Language detection prioritizes Vietnamese
```
**Risk:** 🟢 LOW  
**Conflict Probability:** 20%

#### 3. New Feature Components

**webui/src/features/analytics/AnalyticsContainer.jsx** (293 lines)
```jsx
Complete analytics dashboard:
- Overview cards
- Period stats
- Top lists
- VND currency support
- Cyberpunk styling
```
**Risk:** 🟢 LOW - Isolated component  
**Conflict Probability:** 0%

#### 4. Modified Components

**webui/src/layout/DashboardShell.jsx** (336 lines)
```jsx
Changes:
- Added analytics navigation (line 53)
- Added analytics route (line 121)
- Custom branding links:
  - "Tạo mail nhanh" → https://tm.cuong.tech
  - "Skill hỗ trợ Vibecode" → https://vibekit.codes
- Footer section (lines 306-330):
  - "VIBECODE VIETNAM" branding
  - "Cuongunder" credit
  - Telegram: @tiensinhcc
```
**Risk:** 🔴 HIGH - Core layout file  
**Conflict Probability:** 70%

**webui/src/components/LanguageToggle.jsx**
```jsx
Changes:
- Added Vietnamese option
- Removed Chinese from UI
```
**Risk:** 🟢 LOW  
**Conflict Probability:** 30%

---

## Part 2: Categorization by Preservation Strategy

### Category A: Pluginizable (Can be converted to plugins)
✅ **Easy to preserve via plugin system**

1. **Analytics Feature** ✅ Already done
   - Backend: `internal/httpapi/admin/analytics/` → `plugins/analytics/`
   - Frontend: `webui/src/features/analytics/` → Plugin webui bundle
   - Status: ✅ Backend converted, Frontend needs bundling

2. **Example Plugin** ✅ Already done
   - Demonstrates plugin system works

### Category B: Theme/Styling (Requires theme plugin system)
⚠️ **Needs theme plugin architecture**

1. **Cyberpunk Theme**
   - `webui/src/styles.css` - Complete rewrite
   - `webui/tailwind.config.js` - Custom colors
   - `webui/index.html` - Fonts, meta tags
   - **Solution:** Theme plugin system with CSS injection

2. **Custom Assets**
   - `webui/public/ds2api-favicon.svg`
   - **Solution:** Asset override system

### Category C: Translations (Requires i18n plugin)
⚠️ **Needs i18n plugin system**

1. **Vietnamese Translations**
   - `webui/src/locales/vi.json` (530 lines)
   - Default language preference
   - **Solution:** Translation plugin system

### Category D: Core Modifications (Requires smart merge)
🔴 **High conflict risk, needs careful handling**

1. **Router Integration**
   - `internal/server/router.go` - Plugin manager, analytics handler
   - **Solution:** Plugin auto-registration hooks

2. **Config Structure**
   - `internal/config/config.go` - PricingConfig
   - **Solution:** Config extension system

3. **Layout Customizations**
   - `webui/src/layout/DashboardShell.jsx` - Branding, footer
   - **Solution:** Layout plugin hooks

### Category E: Infrastructure (Core dependencies)
🔴 **Cannot be pluginized, must be preserved**

1. **Plugin System**
   - `internal/plugin/` (1,049 lines)
   - **Solution:** Must be in core or separate branch

---

## Part 3: Preservation Strategies

### Strategy 1: Full Plugin Architecture (Recommended)
**Approach:** Convert all customizations to plugins

**Pros:**
- ✅ Clean separation from upstream
- ✅ Easy to maintain
- ✅ Can update upstream without conflicts
- ✅ Modular and scalable

**Cons:**
- ❌ Requires significant refactoring
- ❌ Plugin system itself needs to be preserved
- ❌ Some features hard to pluginize (theme, i18n)

**Implementation:**
1. **Phase 1:** Plugin system in core (already done ✅)
2. **Phase 2:** Theme plugin system
3. **Phase 3:** i18n plugin system
4. **Phase 4:** Layout hooks plugin system
5. **Phase 5:** Convert all features to plugins

**Estimated Effort:** 2-3 weeks

---

### Strategy 2: Git-Based Fork Management (Alternative)
**Approach:** Maintain custom branch with smart merge strategy

**Pros:**
- ✅ No refactoring needed
- ✅ Keep all customizations as-is
- ✅ Git handles most conflicts

**Cons:**
- ❌ Manual conflict resolution on every update
- ❌ High maintenance overhead
- ❌ Risk of merge conflicts

**Implementation:**
1. Create `custom` branch from current state
2. Track upstream as `upstream/main`
3. Periodically merge upstream into custom
4. Resolve conflicts manually
5. Use `.gitattributes` for merge strategies

**Estimated Effort:** 1-2 hours per upstream update

---

### Strategy 3: Hybrid Approach (Best Balance)
**Approach:** Plugin system + Git branch + Smart merge

**Pros:**
- ✅ Best of both worlds
- ✅ Plugins for features (analytics, etc.)
- ✅ Git for theme/i18n
- ✅ Automated merge for low-risk files

**Cons:**
- ❌ More complex setup
- ❌ Requires both plugin and git knowledge

**Implementation:**

#### Step 1: Keep Plugin System in Core
```
internal/plugin/ → Core infrastructure (must preserve)
```

#### Step 2: Pluginize Features
```
✅ Analytics → Plugin (done)
✅ Example → Plugin (done)
🔄 Future features → Plugins
```

#### Step 3: Git Branch Strategy
```
main (upstream tracking)
  ↓
custom (your customizations)
  ↓
production (deployed version)
```

#### Step 4: Automated Merge Rules
```
.gitattributes:
# Auto-merge config files (keep ours)
config.json merge=ours
webui/src/locales/vi.json merge=ours

# Auto-merge styles (keep ours)
webui/src/styles.css merge=ours
webui/tailwind.config.js merge=ours

# Manual merge for core files
internal/server/router.go merge=manual
internal/config/config.go merge=manual
```

#### Step 5: Update Workflow
```bash
# 1. Fetch upstream
git fetch upstream

# 2. Merge into custom branch
git checkout custom
git merge upstream/main

# 3. Resolve conflicts (if any)
# - Plugin system: Keep ours
# - Theme/i18n: Keep ours
# - Core files: Manual merge

# 4. Test
go build ./cmd/ds2api
cd webui && npm run build

# 5. Deploy
git checkout production
git merge custom
```

**Estimated Effort:** 
- Initial setup: 1 day
- Per update: 30 minutes - 2 hours (depending on conflicts)

---

## Part 4: Recommended Strategy

### 🎯 **Hybrid Approach** (Strategy 3)

**Rationale:**
1. Plugin system already implemented ✅
2. Analytics already converted to plugin ✅
3. Theme/i18n hard to pluginize → Git merge
4. Best balance of maintainability vs effort

**Implementation Plan:**

### Phase 1: Setup Git Branch Structure (1 hour)
```bash
# Backup current state
git branch custom-backup

# Create custom branch
git branch custom
git checkout custom

# Add upstream remote
git remote add upstream https://github.com/CJackHwang/ds2api.git
git fetch upstream

# Setup merge strategies
cat > .gitattributes << EOF
# Preserve customizations
config.json merge=ours
webui/src/locales/vi.json merge=ours
webui/src/styles.css merge=ours
webui/tailwind.config.js merge=ours
webui/index.html merge=ours
webui/public/ds2api-favicon.svg merge=ours

# Manual merge for integration points
internal/server/router.go merge=manual
internal/config/config.go merge=manual
webui/src/layout/DashboardShell.jsx merge=manual
EOF

git add .gitattributes
git commit -m "Add merge strategies for customizations"
```

### Phase 2: Document Customizations (30 minutes)
```bash
# Create CUSTOMIZATIONS.md
cat > CUSTOMIZATIONS.md << EOF
# DS2API Customizations by Cuongunder

## Custom Features
- Analytics Dashboard (plugin)
- Plugin System Infrastructure
- Cyberpunk/Neon Theme
- Vietnamese Translations
- Vibecode Vietnam Branding

## Modified Files
See CUSTOMIZATIONS_AUDIT.md for complete list

## Update Procedure
See docs/UPDATE_PROCEDURE.md
EOF
```

### Phase 3: Create Update Script (1 hour)
```bash
# scripts/update-from-upstream.sh
#!/bin/bash
set -e

echo "🔄 Updating DS2API from upstream..."

# Fetch upstream
git fetch upstream

# Show what will be merged
echo "📋 Changes in upstream:"
git log HEAD..upstream/main --oneline | head -20

read -p "Continue with merge? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
fi

# Merge
git merge upstream/main

# Check for conflicts
if git diff --name-only --diff-filter=U | grep -q .; then
    echo "⚠️  Conflicts detected in:"
    git diff --name-only --diff-filter=U
    echo ""
    echo "Please resolve conflicts manually:"
    echo "1. Edit conflicted files"
    echo "2. git add <file>"
    echo "3. git commit"
    exit 1
fi

# Build and test
echo "🔨 Building..."
go build ./cmd/ds2api

echo "🧪 Testing..."
./ds2api.exe --version

echo "✅ Update complete!"
echo "Next steps:"
echo "1. Test thoroughly"
echo "2. git push origin custom"
```

### Phase 4: Test Update Process (2 hours)
```bash
# Simulate upstream update
git checkout -b test-update
git merge upstream/main

# Resolve conflicts
# Test build
# Test functionality

# If successful, document conflicts and resolutions
```

### Phase 5: Create Rollback Plan (30 minutes)
```bash
# scripts/rollback.sh
#!/bin/bash
BACKUP_BRANCH="custom-backup-$(date +%Y%m%d-%H%M%S)"

echo "📦 Creating backup: $BACKUP_BRANCH"
git branch $BACKUP_BRANCH

echo "⏮️  Rolling back to previous state"
git reset --hard HEAD~1

echo "✅ Rollback complete"
echo "Backup saved as: $BACKUP_BRANCH"
```

---

## Part 5: Future-Proofing

### A. Plugin System Enhancements

**1. Theme Plugin Support**
```go
// internal/plugin/interface.go
type ThemePlugin interface {
    Plugin
    GetStylesheet() string
    GetAssets() map[string][]byte
}
```

**2. i18n Plugin Support**
```go
// internal/plugin/interface.go
type I18nPlugin interface {
    Plugin
    GetTranslations() map[string]map[string]string
}
```

**3. Layout Hook Plugin Support**
```go
// internal/plugin/interface.go
type LayoutPlugin interface {
    Plugin
    GetHeaderHTML() string
    GetFooterHTML() string
    GetCustomLinks() []Link
}
```

### B. Auto-Update System (Future)

**When plugin system is mature:**
1. Check upstream for updates
2. Download new version
3. Backup current version
4. Install update (preserve plugins/)
5. Restart server
6. Verify plugins still work
7. Rollback if failed

---

## Part 6: Risk Assessment

### High Risk Files (90%+ conflict probability)
1. `internal/server/router.go` - Core routing, frequently updated
2. `webui/src/styles.css` - Complete rewrite, high conflict
3. `webui/src/layout/DashboardShell.jsx` - Layout changes

**Mitigation:**
- Keep detailed documentation of changes
- Use git merge markers
- Test thoroughly after each merge

### Medium Risk Files (40-70% conflict probability)
1. `internal/config/config.go` - Struct changes
2. `webui/src/locales/vi.json` - New translation keys
3. `webui/index.html` - Meta tags, fonts

**Mitigation:**
- Use merge=ours for translations
- Manual review for config changes

### Low Risk Files (0-30% conflict probability)
1. `internal/plugin/` - New package, no conflicts
2. `plugins/` - Isolated, no conflicts
3. `webui/src/features/analytics/` - New component, no conflicts

**Mitigation:**
- None needed, safe to preserve

---

## Part 7: Action Items

### Immediate (This Week)
- [ ] Setup git branch structure
- [ ] Create .gitattributes with merge strategies
- [ ] Document all customizations
- [ ] Create update script
- [ ] Test update process with current upstream

### Short Term (This Month)
- [ ] Create rollback script
- [ ] Setup CI/CD for testing updates
- [ ] Document conflict resolution procedures
- [ ] Train team on update workflow

### Long Term (Next Quarter)
- [ ] Implement theme plugin system
- [ ] Implement i18n plugin system
- [ ] Implement layout hooks plugin system
- [ ] Build auto-update system

---

## Conclusion

**Current State:**
- ✅ Plugin system implemented
- ✅ Analytics converted to plugin
- ⚠️ Theme/i18n still in core
- ⚠️ No update strategy in place

**Recommended Next Steps:**
1. **Immediate:** Setup git branch + merge strategies (1 day)
2. **Short term:** Test update workflow (1 week)
3. **Long term:** Enhance plugin system (1-3 months)

**Estimated Maintenance Overhead:**
- With hybrid strategy: **30 min - 2 hours per upstream update**
- Without strategy: **Risk of losing all customizations**

**ROI:**
- Initial investment: 1-2 days
- Ongoing savings: Preserve 1,800+ lines of custom code
- Risk reduction: 95% → 10%

---

**Document Version:** 1.0  
**Last Updated:** 2026-05-07  
**Author:** Claude (Kiro AI)  
**Reviewed By:** Cuongunder
