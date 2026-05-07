# Hướng dẫn Deploy DS2API lên VPS với Cloudflare Tunnel

## Yêu cầu VPS

- Ubuntu 20.04+ hoặc Debian 11+
- RAM: Tối thiểu 1GB (recommend 2GB)
- CPU: 1 core
- Disk: 10GB
- Root access hoặc sudo

## Bước 1: Chuẩn bị VPS

### 1.1. Cập nhật hệ thống

```bash
sudo apt update && sudo apt upgrade -y
```

### 1.2. Cài đặt Git

```bash
sudo apt install git -y
git --version
```

### 1.3. Cài đặt Go (v1.21+)

```bash
# Download Go
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz

# Extract
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
```

### 1.4. Cài đặt Node.js (v18+)

```bash
# Install nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
source ~/.bashrc

# Install Node.js
nvm install 18
nvm use 18

# Verify
node --version
npm --version
```

## Bước 2: Clone và Build Project

### 2.1. Clone repository

```bash
cd ~
git clone https://github.com/CJackHwang/ds2api.git
cd ds2api
```

**Nếu bạn đã push custom-vibecode branch lên GitHub:**

```bash
git checkout custom-vibecode
```

**Nếu chưa push (chỉ có local):**

Bạn cần push từ máy local trước:

```bash
# Trên máy local Windows
cd C:\Users\cuongunder\Downloads\sdasdsad\ds2api
git push origin custom-vibecode

# Sau đó trên VPS
git checkout custom-vibecode
```

### 2.2. Build frontend

```bash
cd webui
npm install
npm run build
cd ..
```

### 2.3. Build backend

```bash
go build -o ds2api ./cmd/ds2api
```

### 2.4. Tạo config.json

```bash
cp config.json.example config.json
nano config.json
```

Sửa config theo nhu cầu (thêm accounts, keys, etc.)

### 2.5. Set admin password

```bash
export DS2API_ADMIN_KEY="your-strong-password-here"
```

Hoặc thêm vào `~/.bashrc`:

```bash
echo 'export DS2API_ADMIN_KEY="your-strong-password-here"' >> ~/.bashrc
source ~/.bashrc
```

## Bước 3: Tạo Systemd Service

### 3.1. Tạo service file

```bash
sudo nano /etc/systemd/system/ds2api.service
```

Nội dung:

```ini
[Unit]
Description=DS2API Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/root/ds2api
Environment="DS2API_ADMIN_KEY=your-strong-password-here"
ExecStart=/root/ds2api/ds2api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

**Lưu ý:** Thay `your-strong-password-here` bằng password thật.

### 3.2. Enable và start service

```bash
sudo systemctl daemon-reload
sudo systemctl enable ds2api
sudo systemctl start ds2api
```

### 3.3. Kiểm tra status

```bash
sudo systemctl status ds2api
```

### 3.4. Xem logs

```bash
sudo journalctl -u ds2api -f
```

## Bước 4: Setup Cloudflare Tunnel

### 4.1. Cài đặt cloudflared

```bash
# Download cloudflared
wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb

# Install
sudo dpkg -i cloudflared-linux-amd64.deb

# Verify
cloudflared --version
```

### 4.2. Login Cloudflare

```bash
cloudflared tunnel login
```

Trình duyệt sẽ mở → Đăng nhập Cloudflare → Chọn domain → Authorize.

### 4.3. Tạo tunnel

```bash
cloudflared tunnel create ds2api
```

Lưu lại **Tunnel ID** (ví dụ: `abc123-def456-ghi789`)

### 4.4. Tạo config file

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

### 4.5. Tạo DNS record

```bash
cloudflared tunnel route dns ds2api ds2api.yourdomain.com
```

### 4.6. Chạy tunnel

**Test trước:**

```bash
cloudflared tunnel run ds2api
```

Nếu OK, Ctrl+C để dừng.

**Chạy như service:**

```bash
sudo cloudflared service install
sudo systemctl start cloudflared
sudo systemctl enable cloudflared
```

### 4.7. Kiểm tra tunnel status

```bash
sudo systemctl status cloudflared
cloudflared tunnel info ds2api
```

## Bước 5: Truy cập và Test

### 5.1. Truy cập web

Mở trình duyệt: `https://ds2api.yourdomain.com/admin`

- Username: (để trống)
- Password: `your-strong-password-here`

### 5.2. Test update system

1. Vào trang **Cập nhật hệ thống**
2. Bấm **Kiểm tra cập nhật**
3. Nếu có update → Bấm **Cài đặt ngay**
4. Đợi 1-2 phút → Server tự động restart
5. Refresh trang → Xong!

## Bước 6: Bảo trì

### 6.1. Xem logs

```bash
# DS2API logs
sudo journalctl -u ds2api -f

# Cloudflare Tunnel logs
sudo journalctl -u cloudflared -f
```

### 6.2. Restart service

```bash
sudo systemctl restart ds2api
sudo systemctl restart cloudflared
```

### 6.3. Update thủ công (nếu cần)

```bash
cd ~/ds2api
git pull origin custom-vibecode
cd webui && npm run build && cd ..
go build -o ds2api ./cmd/ds2api
sudo systemctl restart ds2api
```

### 6.4. Backup

```bash
# Backup config
cp ~/ds2api/config.json ~/ds2api-config-backup.json

# Backup database (nếu có)
cp -r ~/ds2api/data ~/ds2api-data-backup
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
# Xóa node_modules và rebuild
cd ~/ds2api/webui
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Lỗi: go build failed

```bash
# Update Go modules
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

### Lỗi: Git merge conflict khi update

```bash
# SSH vào VPS
cd ~/ds2api

# Xem conflict files
git status

# Resolve conflict thủ công
nano <conflict-file>

# Sau khi resolve
git add .
git commit -m "resolve merge conflict"

# Rebuild
cd webui && npm run build && cd ..
go build -o ds2api ./cmd/ds2api
sudo systemctl restart ds2api
```

## Security Best Practices

1. **Đổi admin password mạnh**
2. **Enable firewall:**
   ```bash
   sudo ufw allow 22/tcp
   sudo ufw enable
   ```
3. **Disable root SSH login** (sau khi tạo user thường)
4. **Enable auto-updates:**
   ```bash
   sudo apt install unattended-upgrades -y
   sudo dpkg-reconfigure -plow unattended-upgrades
   ```
5. **Backup định kỳ** config.json và data/

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

---

**Chúc bạn deploy thành công!** 🚀

Nếu gặp vấn đề, kiểm tra logs:
- DS2API: `sudo journalctl -u ds2api -f`
- Cloudflare: `sudo journalctl -u cloudflared -f`
