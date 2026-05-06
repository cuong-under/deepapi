# DS2API Update Procedure - Vibecode Vietnam Edition

**Last Updated:** 2026-05-07  
**Maintainer:** Cuongunder (@tiensinhcc)  
**Project:** DS2API Fork with Vibecode Vietnam Customizations

---

## Overview

Dự án này là fork của [DS2API](https://github.com/CJackHwang/ds2api) với extensive customizations:
- Plugin System Infrastructure
- Analytics Dashboard
- Cyberpunk/Neon UI Theme
- Vietnamese Translations
- Vibecode Vietnam Branding

Document này hướng dẫn cách update từ upstream mà vẫn preserve tất cả customizations.

---

## Quick Start

### Update từ Upstream (Recommended)

```bash
# 1. Chạy update script
bash scripts/update-from-upstream.sh

# 2. Test thoroughly
./ds2api.exe

# 3. Nếu có vấn đề, rollback
bash scripts/rollback.sh
```

---

## Detailed Procedure

### Step 1: Preparation

**Before updating, ensure:**
- ✅ All changes are committed
- ✅ Server is stopped
- ✅ You're on `custom-vibecode` branch
- ✅ You have recent backup

```bash
# Check current branch
git branch --show-current
# Should show: custom-vibecode

# Check for uncommitted changes
git status

# Commit if needed
git add -A
git commit -m "chore: prepare for upstream update"
```

### Step 2: Run Update Script

```bash
bash scripts/update-from-upstream.sh
```

**The script will:**
1. ✅ Check prerequisites
2. ✅ Fetch upstream changes
3. ✅ Show what will be merged
4. ✅ Create automatic backup branch
5. ✅ Merge with smart strategies
6. ✅ Build and test
7. ✅ Report results

### Step 3: Handle Conflicts (If Any)

**Common conflicts and resolutions:**

#### Conflict 1: `internal/server/router.go`
```go
<<<<<<< HEAD (ours - keep this)
// Plugin manager initialization
pluginsDir := filepath.Join(filepath.Dir(config.ConfigPath()), "plugins")
pluginManager := plugin.NewManager(pluginsDir, pluginDeps)
=======
// Upstream changes
>>>>>>> origin/main (theirs - review and merge)
```

**Resolution:** Keep plugin manager initialization, merge upstream changes around it.

#### Conflict 2: `internal/config/config.go`
```go
<<<<<<< HEAD (ours - keep this)
type Config struct {
    // ... existing fields
    Pricing PricingConfig `json:"pricing,omitempty"`
}
=======
// Upstream may add new fields
>>>>>>> origin/main
```

**Resolution:** Keep `Pricing` field, add any new upstream fields.

#### Conflict 3: `webui/src/layout/DashboardShell.jsx`
```jsx
<<<<<<< HEAD (ours - keep this)
{/* Vibecode Vietnam Branding */}
<footer className="mt-8 border-t border-cyan-500/30 pt-6">
  <div className="text-center">
    <p className="text-sm text-cyan-400/60">
      Dự án phi lợi nhuận cho cộng đồng
    </p>
    <p className="text-lg font-bold text-cyan-400 glow-cyan">
      VIBECODE VIETNAM
    </p>
  </div>
</footer>
=======
// Upstream changes
>>>>>>> origin/main
```

**Resolution:** Keep Vibecode branding, merge upstream layout changes.

### Step 4: Resolve Conflicts Manually

```bash
# 1. Edit conflicted files
code internal/server/router.go

# 2. Stage resolved files
git add internal/server/router.go

# 3. Continue merge
git commit

# 4. Build and test
go build ./cmd/ds2api
./ds2api.exe
```

### Step 5: Test Everything

**Critical tests:**

1. **Server Startup**
   ```bash
   ./ds2api.exe
   # Check logs for:
   # - [plugin] registered plugin factory name=analytics
   # - [plugin] loaded plugin from registry name=analytics
   # - [analytics] pricing initialized
   ```

2. **Analytics Dashboard**
   ```bash
   # Open browser: http://localhost:5001/admin
   # Login with admin key
   # Navigate to "Thống kê Token"
   # Verify data displays correctly
   ```

3. **Plugin System**
   ```bash
   curl http://localhost:5001/admin/analytics/overview \
     -H "Authorization: Bearer admin"
   # Should return JSON with token stats
   ```

4. **UI Theme**
   - Check cyberpunk theme loads (cyan colors, glow effects)
   - Check Vietnamese translations work
   - Check Vibecode branding in footer

5. **Build Frontend**
   ```bash
   cd webui
   npm run build
   # Check for errors
   ```

### Step 6: Rollback (If Needed)

**If tests fail:**

```bash
# Option 1: Use rollback script
bash scripts/rollback.sh

# Option 2: Manual rollback
git reset --hard backup-YYYYMMDD-HHMMSS
```

---

## Merge Strategies Explained

### Auto-Preserve (merge=ours)

These files are **never overwritten** by upstream:

- `config.json` - Your accounts, API keys, pricing
- `webui/src/locales/vi.json` - Vietnamese translations
- `webui/src/styles.css` - Cyberpunk theme
- `webui/tailwind.config.js` - Custom colors
- `webui/index.html` - Custom fonts, meta tags
- `webui/public/ds2api-favicon.svg` - Custom logo
- `webui/src/layout/DashboardShell.jsx` - Vibecode branding
- `internal/plugin/*` - Plugin system
- `plugins/*` - Plugin implementations
- `internal/httpapi/admin/analytics/*` - Analytics feature

### Manual Merge (merge=manual)

These files require **careful review**:

- `internal/server/router.go` - Core routing (high conflict risk)
- `internal/config/config.go` - Config structure (medium risk)
- `internal/httpapi/admin/handler.go` - Admin routes (medium risk)
- `cmd/ds2api/main.go` - Main entry point (low risk)

**Why manual?** These files integrate customizations with core code. Upstream may change surrounding code, requiring careful merge.

---

## Troubleshooting

### Issue 1: Build Fails After Update

**Symptoms:**
```
# ds2api/internal/plugin
undefined: SomeType
```

**Solution:**
```bash
# Check if upstream changed dependencies
go mod tidy

# Rebuild
go build ./cmd/ds2api
```

### Issue 2: Plugin System Not Loading

**Symptoms:**
```
[plugin] failed to load plugin from registry
```

**Solution:**
```bash
# Check plugin imports in router.go
grep "_ \"ds2api/plugins" internal/server/router.go

# Should see:
# _ "ds2api/plugins/analytics"
# _ "ds2api/plugins/example"
```

### Issue 3: Analytics Returns 0 Data

**Symptoms:**
- Dashboard shows all zeros
- No token stats

**Solution:**
```bash
# Check pricing config loaded
grep "pricing" config.json

# Check analytics handler initialized
grep "analytics.*initialized" logs
```

### Issue 4: UI Theme Broken

**Symptoms:**
- No cyan colors
- No glow effects
- Default theme showing

**Solution:**
```bash
# Rebuild frontend
cd webui
npm run build

# Check styles.css preserved
git diff origin/main webui/src/styles.css
# Should show extensive differences (cyberpunk theme)
```

### Issue 5: Vietnamese Not Default

**Symptoms:**
- UI shows English by default

**Solution:**
```bash
# Check i18n.jsx
grep "defaultLanguage" webui/src/i18n.jsx
# Should be: defaultLanguage: 'vi'

# Rebuild frontend
cd webui && npm run build
```

---

## Maintenance Schedule

### Weekly
- ✅ Check for upstream updates
- ✅ Review upstream changelog

### Monthly
- ✅ Update from upstream (if changes available)
- ✅ Test all features thoroughly
- ✅ Update documentation if needed

### Quarterly
- ✅ Review and cleanup backup branches
- ✅ Audit customizations for obsolete code
- ✅ Consider contributing features back to upstream

---

## Backup Strategy

### Automatic Backups

Update script creates automatic backups:
```
backup-20260507-143022  (created by update script)
backup-20260506-091544  (previous update)
backup-20260505-162133  (older update)
```

### Manual Backups

Before major changes:
```bash
# Create named backup
git branch backup-before-major-refactor

# List backups
git branch | grep backup
```

### Cleanup Old Backups

```bash
# Delete backups older than 30 days
git branch | grep "backup-2026" | while read branch; do
  git branch -D $branch
done
```

---

## Emergency Procedures

### Complete System Failure

**If everything breaks:**

```bash
# 1. Stop server
pkill ds2api

# 2. Rollback to last known good state
git reset --hard backup-YYYYMMDD-HHMMSS

# 3. Clean build
rm -rf ds2api.exe
go clean -cache
go build ./cmd/ds2api

# 4. Rebuild frontend
cd webui
rm -rf node_modules dist
npm install
npm run build

# 5. Test
./ds2api.exe
```

### Lost Customizations

**If customizations accidentally deleted:**

```bash
# 1. Check reflog
git reflog

# 2. Find commit before deletion
git log --all --oneline | grep "Vibecode"

# 3. Restore from commit
git checkout <commit-hash> -- <file>

# 4. Commit restoration
git commit -m "restore: Recover lost customizations"
```

---

## Contact & Support

**Maintainer:** Cuongunder  
**Telegram:** @tiensinhcc  
**Community:** Vibecode Vietnam  

**For issues:**
1. Check this document first
2. Review `CUSTOMIZATIONS_AUDIT.md`
3. Contact via Telegram

---

## Changelog

### 2026-05-07
- ✅ Initial update procedure documentation
- ✅ Created update and rollback scripts
- ✅ Established merge strategies
- ✅ Documented common conflicts

---

## Appendix: File Inventory

### Custom Files (Safe - No Conflicts)
```
internal/plugin/                    (1,049 lines)
internal/httpapi/admin/analytics/   (582 lines)
plugins/analytics/                  (171 lines)
plugins/example/                    (92 lines)
webui/src/features/analytics/       (293 lines)
webui/src/locales/vi.json          (530 lines)
CUSTOMIZATIONS_AUDIT.md
scripts/update-from-upstream.sh
scripts/rollback.sh
docs/UPDATE_PROCEDURE.md
```

### Modified Files (Conflict Risk)
```
internal/server/router.go           (🔴 HIGH)
internal/config/config.go           (🟡 MEDIUM)
webui/src/styles.css               (🔴 HIGH)
webui/src/layout/DashboardShell.jsx (🔴 HIGH)
webui/index.html                   (🟡 MEDIUM)
webui/tailwind.config.js           (🟡 MEDIUM)
cmd/ds2api/main.go                 (🟢 LOW)
```

---

**End of Document**
