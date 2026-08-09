# 任务批次 35：服务器部署 — 环境初始化 + 后端部署 + 更新检查 API

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。项目开发阶段完成，现在进入生产部署。你负责**服务器端**全部工作（通过 SSH 操作远端服务器）。

## 服务器信息（SSH 密钥已配置，别名 aliyun-server，无需密码）
- Ubuntu 24.04.4 LTS，2 核 / 1.6G 内存（已有 2G swap）/ 40G 磁盘（33G 可用）
- 全新状态：无 Go / PostgreSQL / Nginx
- 入口：http://115.29.197.130（IP 直连，暂不绑域名）

## 一、服务器初始化（SSH 执行）

### 1. 安装基础软件
```bash
ssh aliyun-server "sudo apt-get update -y && sudo apt-get install -y postgresql postgresql-contrib nginx curl ca-certificates"
```
- PostgreSQL 24.04 自带版本（17/18 均可，够用）
- 启动并设置自启：`sudo systemctl enable --now postgresql nginx`

### 2. Go 安装（1.25 或更高）
```bash
ssh aliyun-server "cd /tmp && curl -sL https://go.dev/dl/go1.25.7.linux-amd64.tar.gz -o go.tar.gz && sudo tar -C /usr/local -xzf go.tar.gz && echo 'export PATH=/usr/local/go/bin:\$PATH' | sudo tee /etc/profile.d/go.sh && source /etc/profile.d/go.sh && go version"
```

### 3. PostgreSQL 配置
```bash
# 设 postgres 用户密码（沿用 3143）并建 hrms 库
sudo -u postgres psql -c "ALTER USER postgres PASSWORD '3143';"
sudo -u postgres psql -c "CREATE DATABASE hrms ENCODING 'UTF8';"
```
- **低内存调优**（1.6G）：编辑 /etc/postgresql/*/main/postgresql.conf：
  - shared_buffers = 256MB、effective_cache_size = 512MB、work_mem = 8MB、maintenance_work_mem = 64MB
  - 重启：`sudo systemctl restart postgresql`

### 4. Nginx 配置（反向代理 + 静态托管）
创建 `/etc/nginx/sites-available/hrms`：
```nginx
server {
    listen 80;
    server_name _;
    client_max_body_size 50m;

    # 前端静态文件（后续客户端也走这里）
    root /opt/hrms/web;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # 更新包下载目录（静态文件）
    location /updates/ {
        alias /opt/hrms/updates/;
        add_header Content-Disposition 'attachment; filename="$1"';
    }

    # SPA 路由回退
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```
- `sudo ln -s /etc/nginx/sites-available/hrms /etc/nginx/sites-enabled/`，删默认 site，`sudo nginx -t && sudo systemctl reload nginx`

## 二、后端构建与部署

1. **本机构建 Linux 版**：`cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o hrms-server ./cmd/server`
2. **上传**：`scp backend/hrms-server aliyun-server:/tmp/`
3. **服务器部署**：
```bash
ssh aliyun-server "
sudo mkdir -p /opt/hrms/{web,updates,logs}
sudo mv /tmp/hrms-server /opt/hrms/
# .env（沿用本机 backend/.env 内容，JWT_SECRET 保持相同以便旧 token 兼容）
sudo tee /opt/hrms/.env <<'EOF'
<从 backend/.env 读取，注意转义>
EOF
"
```
4. **systemd 服务** `/etc/systemd/system/hrms.service`：
```ini
[Unit]
Description=HRMS Backend
After=network.target postgresql.service
[Service]
WorkingDirectory=/opt/hrms
EnvironmentFile=/opt/hrms/.env
ExecStart=/opt/hrms/hrms-server
Restart=always
RestartSec=3
User=root
[Install]
WantedBy=multi-user.target
```
- `sudo systemctl enable --now hrms`，`curl http://127.0.0.1:8080/api/v1/health` 验证
- `curl http://115.29.197.130/api/v1/health` 验证 Nginx 链路

5. **前端构建上传**：`cd frontend && npm run build && scp -r dist/* aliyun-server:/tmp/web/ && ssh aliyun-server "sudo cp -r /tmp/web/* /opt/hrms/web/"`
- 验证 `curl http://115.29.197.130/` 返回 index.html

## 三、更新检查 API（后端新增，本机开发后重新部署）

### 模型（models/update.go）
```go
// AppVersion 客户端版本记录
type AppVersion struct {
    ID          uint   `gorm:"primaryKey" json:"id"`
    Platform    string `gorm:"size:16;uniqueIndex:idx_plat_ver;not null" json:"platform"` // win / android
    Version     string `gorm:"size:32;uniqueIndex:idx_plat_ver;not null" json:"version"`  // 如 1.0.0
    BuildNumber int    `gorm:"not null" json:"build_number"`                             // 递增构建号
    URL         string `gorm:"size:512;not null" json:"url"`                             // 下载地址（绝对 URL）
    ReleaseNotes string `gorm:"size:1000" json:"release_notes"`
    CreatedAt   time.Time `json:"created_at"`
}
```
AutoMigrate 注册。

### API
1. `GET /api/v1/update/check?platform=win&version=1.0.0&build=5`（公开，无需 JWT——客户端启动时调用）
   - 查该平台 build_number 最大的记录
   - 若 build > 客户端 build → `{has_update:true, latest:{platform,version,build_number,url,release_notes}}`；否则 `{has_update:false}`
2. `GET /api/v1/update/latest?platform=win`（公开）：返回最新版本信息
3. 管理接口（admin + update:write 权限点，可选——本批次先做 check/latest，管理上传后续批次做或简单做）：
   - `POST /api/v1/update/versions`（admin）：登记新版本（url 指向 /updates/ 下的文件）
   - `GET /api/v1/update/versions?platform=`（admin）：版本列表
   - 权限点：`update:read` / `update:write` 加给 admin

### 更新包存放
- 服务器 `/opt/hrms/updates/`：如 `hrms-win-1.0.1.exe`、`hrms-android-1.0.1.apk`
- URL：`http://115.29.197.130/updates/hrms-win-1.0.1.exe`

## 四、安全加固（服务器）
- UFW 只放行 22/80/443：`sudo ufw allow 22/tcp && sudo ufw allow 80/tcp && sudo ufw allow 443/tcp && sudo ufw --force enable`
- PostgreSQL 只监听 localhost（默认即如此，确认）
- Nginx 关闭 server_tokens：`sudo sed -i 's/# server_tokens off;/server_tokens off;/' /etc/nginx/nginx.conf`

## 五、验证清单（全部执行后汇报）
1. `curl http://115.29.197.130/api/v1/health` → ok
2. `curl http://115.29.197.130/` → index.html
3. 注册 win 版本 1.0.0 → `curl "http://115.29.197.130/api/v1/update/check?platform=win&version=1.0.0&build=1"` → has_update:false；build=0 → has_update:true 且 url 正确
4. `curl http://115.29.197.130/updates/` 目录可访问（或 403 但路径存在）
5. 数据库 hrms 库、seed 账号（admin/admin123）能通过公网登录（POST /api/v1/auth/captcha + login）
6. 服务器资源：`free -h` 确认 PG + Go 共存正常

## 汇报格式
服务器初始化步骤结果、部署产物路径、systemd/nginx 配置要点、更新 API 契约与实测、安全加固结果、公网访问验证。
