# Multi-User Authentication System

## Tổng quan

DS2API hiện hỗ trợ hệ thống multi-user authentication với các tính năng:

- ✅ Đăng ký tài khoản công khai (self-service)
- ✅ Đăng nhập với JWT token
- ✅ User isolation (mỗi user chỉ thấy accounts/keys của mình)
- ✅ Role-based access control (admin vs user)
- ✅ SQLite database với bcrypt password hashing
- ✅ Session management với token expiration

## Kích hoạt Multi-User Mode

### Bước 1: Thiết lập biến môi trường

Tạo hoặc chỉnh sửa file `.env`:

```bash
# Bật multi-user mode
DS2API_MULTI_USER=true

# JWT secret (QUAN TRỌNG: Đổi thành secret mạnh của bạn)
DS2API_JWT_SECRET=your-very-strong-secret-here-change-this

# Admin password mặc định (tùy chọn, mặc định: admin123)
DS2API_ADMIN_PASSWORD=your-admin-password
```

### Bước 2: Khởi động server

```bash
# Từ source code
go run ./cmd/ds2api

# Hoặc từ binary đã build
./ds2api
```

Server sẽ tự động:
- Tạo database `ds2api.db` trong cùng thư mục với `config.json`
- Chạy migrations để tạo tables
- Tạo user admin mặc định nếu chưa có user nào

### Bước 3: Đăng nhập admin

**Thông tin đăng nhập mặc định:**
- Username: `admin`
- Password: `admin123` (hoặc giá trị của `DS2API_ADMIN_PASSWORD`)

**QUAN TRỌNG:** Đổi password admin ngay sau lần đăng nhập đầu tiên!

## API Endpoints

### Authentication

#### 1. Đăng ký user mới (Public)

```bash
POST /api/auth/register
Content-Type: application/json

{
  "username": "john",
  "email": "john@example.com",
  "password": "password123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 604800,
  "user": {
    "id": 2,
    "username": "john",
    "email": "john@example.com",
    "role": "user"
  }
}
```

#### 2. Đăng nhập (Public)

```bash
POST /api/auth/login
Content-Type: application/json

{
  "username": "john",
  "password": "password123"
}
```

**Response:** Giống như register

#### 3. Đăng xuất (Authenticated)

```bash
POST /api/auth/logout
Authorization: Bearer <token>
```

#### 4. Lấy thông tin user hiện tại (Authenticated)

```bash
GET /api/auth/me
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": 2,
  "username": "john",
  "email": "john@example.com",
  "role": "user"
}
```

#### 5. Refresh token (Authenticated)

```bash
POST /api/auth/refresh
Authorization: Bearer <token>
```

### User Accounts Management

#### 1. Liệt kê accounts của user (Authenticated)

```bash
GET /api/user/accounts
Authorization: Bearer <token>
```

**Response:**
```json
{
  "accounts": [
    {
      "id": 1,
      "user_id": 2,
      "name": "My Account",
      "remark": "Personal account",
      "email": "myaccount@deepseek.com",
      "mobile": "",
      "password": "encrypted",
      "proxy_id": "",
      "created_at": 1715097600
    }
  ],
  "total": 1
}
```

**Lưu ý:** 
- User thường chỉ thấy accounts của mình
- Admin thấy tất cả accounts

#### 2. Tạo account mới (Authenticated)

```bash
POST /api/user/accounts
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "My Account",
  "remark": "Personal account",
  "email": "myaccount@deepseek.com",
  "mobile": "",
  "password": "deepseek_password",
  "proxy_id": ""
}
```

#### 3. Lấy thông tin account (Authenticated)

```bash
GET /api/user/accounts/{id}
Authorization: Bearer <token>
```

#### 4. Cập nhật account (Authenticated)

```bash
PUT /api/user/accounts/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Updated Name",
  "remark": "Updated remark",
  "email": "updated@deepseek.com",
  "mobile": "",
  "password": "new_password",
  "proxy_id": ""
}
```

#### 5. Xóa account (Authenticated)

```bash
DELETE /api/user/accounts/{id}
Authorization: Bearer <token>
```

### API Keys Management

#### 1. Liệt kê API keys của user (Authenticated)

```bash
GET /api/user/keys
Authorization: Bearer <token>
```

**Response:**
```json
{
  "keys": [
    {
      "id": 1,
      "user_id": 2,
      "api_key": "sk-abc123...",
      "name": "My API Key",
      "remark": "For production",
      "created_at": 1715097600
    }
  ],
  "total": 1
}
```

#### 2. Tạo API key mới (Authenticated)

```bash
POST /api/user/keys
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "My API Key",
  "remark": "For production"
}
```

**Response:** API key sẽ được tự động generate

#### 3. Cập nhật API key (Authenticated)

```bash
PUT /api/user/keys/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Updated Name",
  "remark": "Updated remark"
}
```

#### 4. Xóa API key (Authenticated)

```bash
DELETE /api/user/keys/{id}
Authorization: Bearer <token>
```

## Security Features

### 1. Password Hashing
- Sử dụng bcrypt với cost factor 10
- Passwords không bao giờ được lưu dưới dạng plaintext

### 2. JWT Token
- Token expires sau 7 ngày
- Token được lưu trong database để có thể revoke
- Sử dụng HMAC-SHA256 signature

### 3. User Isolation
- Mỗi user chỉ có thể truy cập accounts/keys của mình
- Admin có thể xem tất cả resources
- Ownership check ở database level với foreign keys

### 4. Session Management
- Sessions được lưu trong database
- Expired sessions tự động bị reject
- Logout xóa session khỏi database

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

### User Accounts Table
```sql
CREATE TABLE user_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT,
    remark TEXT,
    email TEXT,
    mobile TEXT,
    password TEXT,
    proxy_id TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### User API Keys Table
```sql
CREATE TABLE user_api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    api_key TEXT UNIQUE NOT NULL,
    name TEXT,
    remark TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### Sessions Table
```sql
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## Backward Compatibility

Multi-user mode là **opt-in**. Nếu không set `DS2API_MULTI_USER=true`, hệ thống sẽ hoạt động như cũ với single admin authentication.

## Migration từ Single-User

1. Backup `config.json` hiện tại
2. Set `DS2API_MULTI_USER=true` trong `.env`
3. Khởi động server (database sẽ được tạo tự động)
4. Đăng nhập với admin account
5. Tạo accounts và API keys mới trong database
6. Dần dần migrate từ config.json sang database

## Troubleshooting

### Database không được tạo
- Kiểm tra quyền write trong thư mục chứa `config.json`
- Xem logs để biết lỗi cụ thể

### Token expired
- Token có thời hạn 7 ngày
- Sử dụng `/api/auth/refresh` để refresh token
- Hoặc đăng nhập lại

### Access denied
- Kiểm tra token có hợp lệ không
- Kiểm tra user có quyền truy cập resource không
- Admin có thể truy cập tất cả resources

### Password không đúng
- Passwords phân biệt hoa thường
- Minimum 8 ký tự khi đăng ký

## Best Practices

1. **Đổi JWT secret ngay lập tức** - Không sử dụng default secret
2. **Đổi admin password** - Đổi password mặc định sau lần đăng nhập đầu tiên
3. **Sử dụng HTTPS** - Luôn deploy với HTTPS trong production
4. **Backup database** - Định kỳ backup file `ds2api.db`
5. **Rotate tokens** - Khuyến khích users refresh token định kỳ
6. **Monitor sessions** - Cleanup expired sessions định kỳ

## Roadmap

- [ ] Frontend UI cho registration/login
- [ ] Password reset via email
- [ ] Two-factor authentication (2FA)
- [ ] API rate limiting per user
- [ ] User activity logs
- [ ] Admin dashboard để quản lý users
