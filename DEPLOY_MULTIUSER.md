# Hướng dẫn Deploy Multi-User System lên VPS

## Tổng quan

Hướng dẫn này sẽ giúp bạn deploy DS2API với multi-user authentication lên VPS mà không cần setup lại từ đầu.

## Phương án Deploy

### Phương án 1: Update từ VPS hiện tại (Khuyến nghị)

Nếu bạn đã có DS2API đang chạy trên VPS, bạn chỉ cần update code và enable multi-user mode.

#### Bước 1: Backup dữ liệu hiện tại

```bash
# SSH vào VPS
ssh user@your-vps-ip

# Backup config.json
cd ~/ds2api
cp config.json config.json.backup.$(date +%Y%m%d_%H%M%S)

# Backup toàn bộ thư mục (optional)
cd ~
tar -czf ds2api_backup_$(date +%Y%m%d_%H%M%S).tar.gz ds2api/
```

#### Bước 2: Pull code mới từ Git

```bash
cd ~/ds2api

# Stash local changes nếu có
git stash

# Pull code mới
git pull origin custom-vibecode

# Hoặc nếu bạn đã fork và push lên repo của mình
git pull origin main
```

#### Bước 3: Build lại backend và frontend

```bash
# Build backend
go build -o ds2api ./cmd/ds2api

# Build frontend
cd webui
npm install
npm run build
cd ..
```

#### Bước 4: Cấu hình Multi-User Mode

```bash
# Tạo hoặc edit file .env
nano .env
```

Thêm các dòng sau:

```bash
# Enable multi-user mode
DS2API_MULTI_USER=true

# JWT secret (QUAN TRỌNG: Đổi thành secret mạnh)
DS2API_JWT_SECRET=$(openssl rand -hex 32)

# Admin password mặc định (optional)
DS2API_ADMIN_PASSWORD=your-strong-password-here

# Port (giữ nguyên nếu đã có)
PORT=5001
```

**Lưu ý:** Nếu bạn muốn generate JWT secret ngay:

```bash
# Generate và thêm vào .env
echo "DS2API_JWT_SECRET=$(openssl rand -hex 32)" >> .env
```

#### Bước 5: Restart service

```bash
# Restart DS2API service
sudo systemctl restart ds2api

# Kiểm tra status
sudo systemctl status ds2api

# Xem logs
sudo journalctl -u ds2api -f
```

#### Bước 6: Kiểm tra Multi-User Mode

```bash
# Test API endpoint
curl http://localhost:5001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}'

# Nếu trả về JSON (không phải 404) = multi-user đã enabled
```

#### Bước 7: Truy cập và đăng nhập

1. Mở browser: `https://yourdomain.com` (hoặc IP của VPS)
2. Bạn sẽ thấy giao diện đăng nhập/đăng ký mới
3. Đăng nhập với admin account:
   - Username: `admin`
   - Password: `admin123` (hoặc giá trị của `DS2API_ADMIN_PASSWORD`)

**QUAN TRỌNG:** Đổi password admin ngay sau lần đăng nhập đầu tiên!

---

### Phương án 2: Deploy từ đầu (Fresh Install)

Nếu bạn muốn setup VPS mới hoàn toàn.

#### Bước 1: Chuẩn bị VPS

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y git curl wget build-essential
```

#### Bước 2: Install Go

```bash
# Download Go 1.21.6
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

#### Bước 3: Install Node.js

```bash
# Install Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Verify
node --version
npm --version
```

#### Bước 4: Clone repository

```bash
cd ~
git clone https://github.com/your-username/deepapi.git ds2api
cd ds2api
git checkout custom-vibecode
```

#### Bước 5: Build project

```bash
# Build frontend
cd webui
npm install
npm run build
cd ..

# Build backend
go mod tidy
go build -o ds2api ./cmd/ds2api
```

#### Bước 6: Cấu hình

```bash
# Copy config example
cp config.example.json config.json

# Edit config
nano config.json
```

Thêm tài khoản DeepSeek và API keys của bạn vào `config.json`.

```bash
# Tạo .env file
cat > .env << 'EOF'
DS2API_MULTI_USER=true
DS2API_JWT_SECRET=CHANGE_THIS_TO_RANDOM_SECRET
DS2API_ADMIN_PASSWORD=admin123
PORT=5001
EOF

# Generate JWT secret
sed -i "s/CHANGE_THIS_TO_RANDOM_SECRET/$(openssl rand -hex 32)/" .env
```

#### Bước 7: Tạo systemd service

```bash
sudo nano /etc/systemd/system/ds2api.service
```

Nội dung:

```ini
[Unit]
Description=DS2API Multi-User Service
After=network.target

[Service]
Type=simple
User=YOUR_USERNAME
WorkingDirectory=/home/YOUR_USERNAME/ds2api
EnvironmentFile=/home/YOUR_USERNAME/ds2api/.env
ExecStart=/home/YOUR_USERNAME/ds2api/ds2api
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Thay `YOUR_USERNAME` bằng username thực của bạn!**

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable ds2api

# Start service
sudo systemctl start ds2api

# Check status
sudo systemctl status ds2api
```

#### Bước 8: Setup Cloudflare Tunnel (Optional)

Nếu bạn muốn expose ra internet với domain:

```bash
# Install cloudflared
wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared-linux-amd64.deb

# Login
cloudflared tunnel login

# Create tunnel
cloudflared tunnel create ds2api

# Configure tunnel
mkdir -p ~/.cloudflared
nano ~/.cloudflared/config.yml
```

Nội dung `config.yml`:

```yaml
tunnel: YOUR_TUNNEL_ID
credentials-file: /home/YOUR_USERNAME/.cloudflared/YOUR_TUNNEL_ID.json

ingress:
  - hostname: yourdomain.com
    service: http://localhost:5001
  - service: http_status:404
```

```bash
# Create DNS record
cloudflared tunnel route dns ds2api yourdomain.com

# Run tunnel as service
sudo cloudflared service install
sudo systemctl start cloudflared
sudo systemctl enable cloudflared
```

---

## Kiểm tra sau khi Deploy

### 1. Kiểm tra service đang chạy

```bash
sudo systemctl status ds2api
```

### 2. Kiểm tra logs

```bash
# Real-time logs
sudo journalctl -u ds2api -f

# Last 100 lines
sudo journalctl -u ds2api -n 100
```

### 3. Kiểm tra database được tạo

```bash
ls -lh ~/ds2api/ds2api.db
```

Nếu file tồn tại = database đã được tạo thành công.

### 4. Test API endpoints

```bash
# Test register
curl -X POST http://localhost:5001/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'

# Test login
curl -X POST http://localhost:5001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'
```

### 5. Truy cập web UI

- Local: `http://localhost:5001`
- Public: `https://yourdomain.com`

---

## Troubleshooting

### Service không start

```bash
# Xem lỗi chi tiết
sudo journalctl -u ds2api -n 50 --no-pager

# Kiểm tra file binary
ls -lh ~/ds2api/ds2api

# Kiểm tra permissions
chmod +x ~/ds2api/ds2api
```

### Database không được tạo

```bash
# Kiểm tra quyền write
ls -ld ~/ds2api

# Tạo database manually
cd ~/ds2api
./ds2api
# Ctrl+C sau khi thấy "database initialized"
```

### Port 5001 đã được sử dụng

```bash
# Tìm process đang dùng port
sudo lsof -i :5001

# Kill process
sudo kill -9 <PID>

# Hoặc đổi port trong .env
echo "PORT=5002" >> .env
sudo systemctl restart ds2api
```

### Frontend không load

```bash
# Rebuild frontend
cd ~/ds2api/webui
rm -rf node_modules package-lock.json
npm install
npm run build
cd ..

# Restart service
sudo systemctl restart ds2api
```

### JWT token invalid

```bash
# Kiểm tra JWT_SECRET đã set chưa
grep JWT_SECRET ~/ds2api/.env

# Nếu chưa có, generate mới
echo "DS2API_JWT_SECRET=$(openssl rand -hex 32)" >> ~/ds2api/.env
sudo systemctl restart ds2api
```

---

## Migration từ Single-User sang Multi-User

### Bước 1: Backup config.json

```bash
cp ~/ds2api/config.json ~/ds2api/config.json.backup
```

### Bước 2: Enable multi-user mode

```bash
echo "DS2API_MULTI_USER=true" >> ~/ds2api/.env
echo "DS2API_JWT_SECRET=$(openssl rand -hex 32)" >> ~/ds2api/.env
sudo systemctl restart ds2api
```

### Bước 3: Đăng nhập admin và tạo accounts mới

1. Truy cập web UI
2. Đăng nhập với admin (username: `admin`, password: `admin123`)
3. Vào tab "Accounts"
4. Thêm các DeepSeek accounts từ config.json cũ vào database

### Bước 4: Test với API keys mới

1. Vào tab "API Keys"
2. Tạo API key mới
3. Test với API key mới

### Bước 5: Dần dần migrate

- Giữ config.json cho backward compatibility
- Tạo accounts mới trong database
- Sau khi stable, có thể xóa accounts trong config.json

---

## Bảo mật

### 1. Đổi JWT secret

```bash
# Generate secret mới
NEW_SECRET=$(openssl rand -hex 32)

# Update .env
sed -i "s/DS2API_JWT_SECRET=.*/DS2API_JWT_SECRET=$NEW_SECRET/" ~/ds2api/.env

# Restart
sudo systemctl restart ds2api
```

### 2. Đổi admin password

Sau khi đăng nhập admin lần đầu, đổi password ngay.

### 3. Enable firewall

```bash
# Allow SSH
sudo ufw allow 22/tcp

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status
```

### 4. Disable root SSH

```bash
sudo nano /etc/ssh/sshd_config
# Đổi: PermitRootLogin no

sudo systemctl restart sshd
```

### 5. Auto-updates

```bash
sudo apt install unattended-upgrades -y
sudo dpkg-reconfigure -plow unattended-upgrades
```

---

## Monitoring

### Xem logs real-time

```bash
sudo journalctl -u ds2api -f
```

### Kiểm tra resource usage

```bash
# CPU và RAM
htop

# Disk
df -h

# Memory
free -h
```

### Kiểm tra database size

```bash
du -h ~/ds2api/ds2api.db
```

---

## Backup và Restore

### Backup

```bash
# Backup database
cp ~/ds2api/ds2api.db ~/ds2api/ds2api.db.backup.$(date +%Y%m%d_%H%M%S)

# Backup config
cp ~/ds2api/config.json ~/ds2api/config.json.backup.$(date +%Y%m%d_%H%M%S)

# Backup .env
cp ~/ds2api/.env ~/ds2api/.env.backup.$(date +%Y%m%d_%H%M%S)

# Full backup
cd ~
tar -czf ds2api_full_backup_$(date +%Y%m%d_%H%M%S).tar.gz ds2api/
```

### Restore

```bash
# Stop service
sudo systemctl stop ds2api

# Restore database
cp ~/ds2api/ds2api.db.backup.YYYYMMDD_HHMMSS ~/ds2api/ds2api.db

# Restore config
cp ~/ds2api/config.json.backup.YYYYMMDD_HHMMSS ~/ds2api/config.json

# Start service
sudo systemctl start ds2api
```

---

## Kết luận

Bạn đã hoàn thành deploy DS2API với multi-user authentication! 

**Checklist:**
- ✅ Backend đã build và chạy
- ✅ Frontend đã build
- ✅ Multi-user mode enabled
- ✅ Database được tạo
- ✅ Admin account hoạt động
- ✅ Service tự động khởi động khi boot
- ✅ Cloudflare Tunnel (optional)

**Next steps:**
1. Đổi admin password
2. Tạo user accounts
3. Tạo API keys
4. Test API endpoints
5. Monitor logs

**Support:**
- GitHub Issues: https://github.com/cuong-under/deepapi/issues
- Documentation: `MULTI_USER.md`
- Implementation: `IMPLEMENTATION_SUMMARY.md`
