# Hướng dẫn Deploy DS2API lên VPS với Cloudflare Tunnel

## Yêu cầu VPS

- Ubuntu 20.04+ hoặc Debian 11+
- RAM: Tối thiểu 1GB (khuyến nghị 2GB)
- CPU: 1 core
- Disk: 10GB
- Root access hoặc sudo

## Phương pháp 1: Deploy tự động (Khuyến nghị)

### Bước 1: Tải và chạy script deploy

```bash
# Tải script deploy
wget https://raw.githubusercontent.com/CJackHwang/ds2api/custom-vibecode/deploy.sh

# Chạy với quyền root
sudo bash deploy.sh
```

Script sẽ tự động:
- Cài đặt Git, Go 1.21.6, Node.js 18
- Clone repository từ GitHub
- Checkout branch `custom-vibecode`
- Build frontend (React + Vite)
- Build backend (Go)
- Tạo systemd service
- Khởi động service
- Hỏi có muốn cài Cloudflare Tunnel không

### Bước 2: Cấu hình

Sau khi script chạy xong, chỉnh sửa config:

```bash
cd ~/ds2api
nano config.json
```

Thêm tài khoản DeepSeek và API keys của bạn.

### Bước 3: Khởi động lại service

```bash
sudo systemctl restart ds2api
```

## Phương pháp 2: Deploy thủ công

### Bước 1: Cài đặt dependencies

```bash
# Cập nhật hệ thống
sudo apt update && sudo apt upgrade -y

# Cài Git
sudo apt install git -y

# Cài Go 1.21.6
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
source /etc/profile
rm go1.21.6.linux-amd64.tar.gz

# Cài Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo bash -
sudo apt install -y nodejs
```

### Bước 2: Clone repository

```bash
cd ~
git clone https://github.com/CJackHwang/ds2api.git
cd ds2api
git checkout custom-vibecode
```

### Bước 3: Build ứng dụng

```bash
# Build frontend
cd webui
npm install
npm run build
cd ..

# Build backend
go build -o ds2api ./cmd/ds2api
```

### Bước 4: Cấu hình

```bash
cp config.json.example config.json
nano config.json
```

Điền thông tin tài khoản DeepSeek và API keys.

### Bước 5: Tạo systemd service

```bash
sudo nano /etc/systemd/system/ds2api.service
```

Nội dung:

```ini
[Unit]
Description=DS2API - DeepSeek API Gateway
Documentation=https://github.com/CJackHwang/ds2api
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/root/ds2api

# Environment variables
Environment="DS2API_ADMIN_KEY=YOUR_STRONG_PASSWORD_HERE"
Environment="PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"

# Start command
ExecStart=/root/ds2api/ds2api

# Restart policy
Restart=always
RestartSec=5
StartLimitInterval=0

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=ds2api

# Security hardening
NoNewPrivileges=true
PrivateTmp=true

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

**Lưu ý:** Thay `YOUR_STRONG_PASSWORD_HERE` bằng mật khẩu mạnh của bạn.

### Bước 6: Khởi động service

```bash
sudo systemctl daemon-reload
sudo systemctl enable ds2api
sudo systemctl start ds2api
sudo systemctl status ds2api
```

## Cài đặt Cloudflare Tunnel

### Bước 1: Cài cloudflared

```bash
wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared-linux-amd64.deb
rm cloudflared-linux-amd64.deb
```

### Bước 2: Đăng nhập Cloudflare

```bash
cloudflared tunnel login
```

Trình duyệt sẽ mở → Đăng nhập Cloudflare → Chọn domain → Authorize.

### Bước 3: Tạo tunnel

```bash
cloudflared tunnel create ds2api
```

Lưu lại **Tunnel ID** (ví dụ: `abc123-def456-ghi789`)

### Bước 4: Tạo config file

```bash
mkdir -p ~/.cloudflared
nano ~/.cloudflared/config.yml
```

Nội dung:

```yaml
tunnel: abc123-def456-ghi789
credentials-file: /root/.cloudflared/abc123-def456-ghi789.json

ingress:
  - hostname: ds2api.yourdomain.com
    service: http://localhost:5001
  - service: http_status:404
```

**Thay thế:**
- `abc123-def456-ghi789` → Tunnel ID của bạn
- `ds2api.yourdomain.com` → Domain của bạn

### Bước 5: Tạo DNS record

```bash
cloudflared tunnel route dns ds2api ds2api.yourdomain.com
```

### Bước 6: Chạy tunnel như service

```bash
sudo cloudflared service install
sudo systemctl start cloudflared
sudo systemctl enable cloudflared
sudo systemctl status cloudflared
```

## Quản lý Service

### Xem logs

```bash
# DS2API logs
sudo journalctl -u ds2api -f

# Cloudflare Tunnel logs
sudo journalctl -u cloudflared -f
```

### Restart service

```bash
sudo systemctl restart ds2api
sudo systemctl restart cloudflared
```

### Stop service

```bash
sudo systemctl stop ds2api
sudo systemctl stop cloudflared
```

## Cập nhật hệ thống

### Cách 1: Qua Admin UI (Khuyến nghị)

1. Truy cập `https://ds2api.yourdomain.com/admin`
2. Đăng nhập bằng `DS2API_ADMIN_KEY`
3. Vào trang **Cập nhật hệ thống**
4. Bấm **Kiểm tra cập nhật**
5. Nếu có update → Bấm **Cài đặt ngay**
6. Đợi 1-2 phút → Server tự động restart
7. Refresh trang → Xong!

**Lưu ý:** Hệ thống sẽ tự động:
- Backup phiên bản hiện tại
- Merge code mới từ upstream
- Giữ lại custom features (Analytics, Vibecode theme)
- Rebuild frontend và backend
- Restart server
- Chỉ giữ 5 bản backup mới nhất

### Cách 2: Thủ công qua SSH

```bash
cd ~/ds2api
git pull origin custom-vibecode
cd webui && npm run build && cd ..
go build -o ds2api ./cmd/ds2api
sudo systemctl restart ds2api
```

## Troubleshooting

### Lỗi: Port 5001 đã được sử dụng

```bash
# Tìm process đang dùng port
sudo lsof -i :5001

# Kill process
sudo kill -9 <PID>
```

### Lỗi: npm build failed

```bash
cd ~/ds2api/webui
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Lỗi: go build failed

```bash
cd ~/ds2api
go mod tidy
go build -o ds2api ./cmd/ds2api
```

### Lỗi: Cloudflare Tunnel không kết nối

```bash
# Kiểm tra config
cat ~/.cloudflared/config.yml

# Restart tunnel
sudo systemctl restart cloudflared

# Xem logs
sudo journalctl -u cloudflared -n 50
```

### Lỗi: Service không start

```bash
# Xem logs chi tiết
sudo journalctl -u ds2api -n 100 --no-pager

# Kiểm tra file binary
ls -lh ~/ds2api/ds2api

# Kiểm tra config
cat ~/ds2api/config.json
```

## Bảo mật

### 1. Đổi admin password mạnh

Trong `/etc/systemd/system/ds2api.service`:

```ini
Environment="DS2API_ADMIN_KEY=your-very-strong-password-here"
```

Sau đó:

```bash
sudo systemctl daemon-reload
sudo systemctl restart ds2api
```

### 2. Enable firewall

```bash
sudo ufw allow 22/tcp
sudo ufw enable
```

**Lưu ý:** Không cần mở port 5001 vì dùng Cloudflare Tunnel.

### 3. Disable root SSH login

```bash
sudo nano /etc/ssh/sshd_config
```

Thay đổi:

```
PermitRootLogin no
```

Restart SSH:

```bash
sudo systemctl restart sshd
```

### 4. Enable auto-updates

```bash
sudo apt install unattended-upgrades -y
sudo dpkg-reconfigure -plow unattended-upgrades
```

### 5. Backup định kỳ

```bash
# Backup config
cp ~/ds2api/config.json ~/ds2api-config-backup-$(date +%Y%m%d).json

# Backup database (nếu có)
cp -r ~/ds2api/data ~/ds2api-data-backup-$(date +%Y%m%d)
```

## Monitoring

### Kiểm tra uptime

```bash
sudo systemctl status ds2api
```

### Kiểm tra resource usage

```bash
htop
```

### Kiểm tra disk space

```bash
df -h
```

### Kiểm tra memory

```bash
free -h
```

## Truy cập

Sau khi deploy xong:

- **Local:** `http://localhost:5001/admin`
- **Public:** `https://ds2api.yourdomain.com/admin`

Đăng nhập bằng password đã set trong `DS2API_ADMIN_KEY`.

---

**Chúc bạn deploy thành công!** 🚀

Nếu gặp vấn đề, kiểm tra logs:
- DS2API: `sudo journalctl -u ds2api -f`
- Cloudflare: `sudo journalctl -u cloudflared -f`
