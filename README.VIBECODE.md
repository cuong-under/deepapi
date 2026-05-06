# DS2API - Vibecode Vietnam Edition

<p align="center">
  <img src="webui/public/ds2api-favicon.svg" width="128" height="128" alt="DS2API Vibecode icon" />
</p>

[![License](https://img.shields.io/github/license/CJackHwang/ds2api.svg)](LICENSE)
![Customized](https://img.shields.io/badge/customized-vibecode-00d9ff)
![Vietnamese](https://img.shields.io/badge/language-tiếng_việt-00d9ff)
![Theme](https://img.shields.io/badge/theme-cyberpunk-ff3399)

**Fork của [DS2API](https://github.com/CJackHwang/ds2api) với extensive customizations cho Vibecode Vietnam**

---

## 🎨 Customizations

### ✨ Features Mới
- **🔌 Plugin System** - Kiến trúc plugin hoàn chỉnh với permission management
- **📊 Analytics Dashboard** - Theo dõi token usage và chi phí real-time
- **💰 Multi-Currency** - Hỗ trợ USD và VND
- **🌐 Tiếng Việt** - Giao diện hoàn toàn bằng tiếng Việt

### 🎭 UI Theme
- **Cyberpunk/Neon Style** - Màu cyan/blue với glow effects
- **Custom Fonts** - Rajdhani và Fira Code
- **Animated Effects** - Grid background, scanlines, glowing borders
- **Vibecode Branding** - Custom footer và links

### 🛠️ Technical
- **~1,800 lines** backend code mới (Go)
- **~530 lines** Vietnamese translations
- **Complete theme overhaul** - Cyberpunk/Neon design
- **Plugin architecture** - Compile-time registration (Windows compatible)

---

## 🚀 Quick Start

### Installation

```bash
# Clone repository
git clone <your-fork-url>
cd ds2api

# Checkout custom branch
git checkout custom-vibecode

# Build
go build ./cmd/ds2api

# Run
./ds2api.exe
```

### Access

- **Admin UI:** http://localhost:5001/admin
- **API Endpoint:** http://localhost:5001/v1
- **Default Admin Key:** `admin` (change in production!)

---

## 📊 Analytics Dashboard

Truy cập **Thống kê Token** trong Admin UI để xem:

- **Overview Cards:** Total tokens, requests, cost, success rate
- **Time Periods:** Today, yesterday, last 7 days, last 30 days
- **Top Lists:** Top accounts, API keys, models
- **Cost Tracking:** Real-time cost calculation với pricing config

### Pricing Configuration

Edit `config.json`:

```json
{
  "pricing": {
    "currency": "USD",
    "models": {
      "deepseek-v4-flash": {
        "input_price_per_1m": 0.14,
        "output_price_per_1m": 0.28
      },
      "deepseek-v4-pro": {
        "input_price_per_1m": 0.55,
        "output_price_per_1m": 2.19
      }
    }
  }
}
```

---

## 🔌 Plugin System

### Available Plugins

1. **Analytics Plugin** - Token usage analytics
2. **Example Plugin** - Demo plugin for testing

### Create New Plugin

```bash
# 1. Create plugin directory
mkdir -p plugins/my-plugin

# 2. Create plugin.json
cat > plugins/my-plugin/plugin.json << EOF
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "My custom plugin",
  "author": "Your Name",
  "entry_point": "internal",
  "permissions": ["read:config"],
  "enabled": true
}
EOF

# 3. Create main.go
# See plugins/example/main.go for template

# 4. Register in router.go
# Add: _ "ds2api/plugins/my-plugin"

# 5. Build and test
go build ./cmd/ds2api
```

---

## 🔄 Update from Upstream

### Automatic Update (Recommended)

```bash
bash scripts/update-from-upstream.sh
```

Script sẽ:
- ✅ Fetch upstream changes
- ✅ Create automatic backup
- ✅ Merge với smart strategies
- ✅ Build và test
- ✅ Rollback nếu failed

### Manual Update

```bash
# 1. Fetch upstream
git fetch origin

# 2. Merge
git merge origin/main

# 3. Resolve conflicts (if any)
# See docs/UPDATE_PROCEDURE.md

# 4. Build and test
go build ./cmd/ds2api
```

### Rollback

```bash
bash scripts/rollback.sh
```

---

## 📚 Documentation

- **[CUSTOMIZATIONS_AUDIT.md](CUSTOMIZATIONS_AUDIT.md)** - Complete audit của tất cả customizations
- **[docs/UPDATE_PROCEDURE.md](docs/UPDATE_PROCEDURE.md)** - Chi tiết về update procedure
- **[API.md](API.md)** - API documentation (upstream)
- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** - Architecture overview (upstream)

---

## 🛡️ Merge Strategies

File `.gitattributes` định nghĩa merge strategies:

### Auto-Preserve (merge=ours)
- `config.json` - Your configuration
- `webui/src/locales/vi.json` - Vietnamese translations
- `webui/src/styles.css` - Cyberpunk theme
- `internal/plugin/*` - Plugin system
- `plugins/*` - Plugin implementations

### Manual Merge (merge=manual)
- `internal/server/router.go` - Core routing
- `internal/config/config.go` - Config structure
- `webui/src/layout/DashboardShell.jsx` - Layout với branding

---

## 🎯 Customization Inventory

### Backend (Go)
```
internal/plugin/                    1,049 lines
internal/httpapi/admin/analytics/     582 lines
plugins/analytics/                    171 lines
plugins/example/                       92 lines
Modified core files:                  ~100 lines
─────────────────────────────────────────────
Total:                              ~1,994 lines
```

### Frontend (React)
```
webui/src/features/analytics/         293 lines
webui/src/locales/vi.json            530 lines
webui/src/styles.css                 Complete rewrite
webui/src/layout/DashboardShell.jsx  Major modifications
webui/index.html                     Custom fonts/meta
─────────────────────────────────────────────
Total:                              ~1,000+ lines
```

---

## 🔧 Development

### Build

```bash
# Backend
go build ./cmd/ds2api

# Frontend
cd webui
npm install
npm run build
```

### Test

```bash
# Run server
./ds2api.exe

# Test analytics endpoint
curl http://localhost:5001/admin/analytics/overview \
  -H "Authorization: Bearer admin"

# Test plugin system
curl http://localhost:5001/admin/example/hello \
  -H "Authorization: Bearer admin"
```

---

## 🌟 Features Comparison

| Feature | Upstream | Vibecode Edition |
|---------|----------|------------------|
| OpenAI API Compatible | ✅ | ✅ |
| Claude API Compatible | ✅ | ✅ |
| Gemini API Compatible | ✅ | ✅ |
| Admin UI | ✅ | ✅ Cyberpunk Theme |
| Multi-Language | ✅ EN/ZH | ✅ EN/ZH/VI (Default: VI) |
| Analytics Dashboard | ❌ | ✅ |
| Plugin System | ❌ | ✅ |
| Cost Tracking | ❌ | ✅ |
| Multi-Currency | ❌ | ✅ USD/VND |
| Custom Branding | ❌ | ✅ Vibecode Vietnam |

---

## 🤝 Contributing

### To Upstream
Nếu bạn muốn contribute features về upstream DS2API:
1. Create PR to [CJackHwang/ds2api](https://github.com/CJackHwang/ds2api)
2. Follow upstream contribution guidelines

### To This Fork
Nếu bạn muốn contribute vào Vibecode edition:
1. Fork this repository
2. Create feature branch
3. Make changes
4. Submit PR to `custom-vibecode` branch

---

## 📞 Contact & Support

**Maintainer:** Cuongunder  
**Telegram:** [@tiensinhcc](https://t.me/tiensinhcc)  
**Community:** Vibecode Vietnam  
**Links:**
- [Tạo mail nhanh](https://tm.cuong.tech)
- [Skill hỗ trợ Vibecode](https://vibekit.codes)

---

## ⚠️ Disclaimer

Dự án này là fork của DS2API với customizations cho Vibecode Vietnam community. 

**Lưu ý:**
- Chỉ dùng cho mục đích học tập và nghiên cứu
- Không dùng cho mục đích thương mại
- Tuân thủ terms of service của DeepSeek
- Tự chịu trách nhiệm khi sử dụng

---

## 📄 License

Kế thừa license từ upstream: [LICENSE](LICENSE)

Customizations by Vibecode Vietnam © 2026

---

## 🙏 Credits

- **Upstream:** [DS2API by CJackHwang](https://github.com/CJackHwang/ds2api)
- **Customizations:** Cuongunder & Claude Sonnet 4
- **Community:** Vibecode Vietnam
- **Design:** Cyberpunk/Neon theme inspired by futuristic aesthetics

---

**Dự án phi lợi nhuận cho cộng đồng Vibecode Vietnam** 🇻🇳
