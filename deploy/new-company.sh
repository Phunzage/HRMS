#!/usr/bin/env bash
# 多公司实例模板化：新增一家公司 = 一条命令（新库 + 新容器 + 新 Nginx server_name）
# 用法: sudo ./new-company.sh <公司标识> <域名>
# 例:   sudo ./new-company.sh comp_a a.phunzage.fun
# 数据隔离: 复用主数据库容器 hrms-db，但独立数据库 hrms_<公司标识>；后端镜像复用 hrms-backend:latest
set -euo pipefail

COMPANY="${1:?用法: new-company.sh <公司标识> <域名>}"
DOMAIN="${2:?用法: new-company.sh <公司标识> <域名>}"

# 校验公司标识只含字母数字下划线（防注入）
[[ "$COMPANY" =~ ^[A-Za-z0-9_-]+$ ]] || { echo "错误: 公司标识含非法字符: $COMPANY"; exit 1; }

DOCKER_DIR="/opt/hrms/docker"
COMPOSE="$DOCKER_DIR/docker-compose.$COMPANY.yml"
WEB_DIR="/opt/hrms/web-$COMPANY"
ENV_FILE="/opt/hrms/.env"
NETWORK="docker_hrms-net"        # 主 compose 项目目录名 docker → 网络 docker_hrms-net
IMAGE="hrms-backend:latest"
# 公司后端对外(仅本机)端口。注意: 用 HOST_PORT 而非 PORT，避免与 .env 里的 PORT=8080 冲突
HOST_PORT=""

test -f "$ENV_FILE" || { echo "错误: 缺少 $ENV_FILE"; exit 1; }
# 载入数据库密码/JWT 等敏感变量供 compose 插值（避免硬编码进生成文件）
set -a; . "$ENV_FILE"; set +a
docker image inspect "$IMAGE" >/dev/null 2>&1 || { echo "错误: 缺少镜像 $IMAGE，请先执行 docker tag <现有镜像> $IMAGE"; exit 1; }

# 取下一个空闲端口（8081 起，避开已有监听）
next_port() {
  local p=8081
  while ss -tlnH 2>/dev/null | awk '{print $4}' | grep -qE ":$p\$"; do
    p=$((p+1))
  done
  echo "$p"
}
HOST_PORT="$(next_port)"

echo "==> 公司 $COMPANY: 域名 $DOMAIN, 端口 $HOST_PORT"

echo "==> 创建数据库 hrms_$COMPANY"
if docker exec hrms-db psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='hrms_$COMPANY'" | grep -q 1; then
  echo "    数据库已存在，跳过创建"
else
  docker exec hrms-db psql -U postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE hrms_$COMPANY;"
fi

echo "==> 生成 compose 文件 $COMPOSE"
cat > "$COMPOSE" <<EOF
# 公司 $COMPANY 的独立后端（复用主数据库容器 hrms-db，独立数据库 hrms_$COMPANY）
# 由 new-company.sh 生成，勿手改。删除实例: docker compose -p hrms-$COMPANY -f $COMPOSE down && rm $COMPOSE
services:
  backend-$COMPANY:
    image: $IMAGE
    container_name: hrms-backend-$COMPANY
    restart: always
    environment:
      DB_HOST: hrms-db
      DB_PORT: "5432"
      DB_USER: postgres
      DB_PASSWORD: "\${DB_PASSWORD}"
      DB_NAME: hrms_$COMPANY
      JWT_SECRET: "\${JWT_SECRET}"
      PORT: "8080"
    env_file:
      - $ENV_FILE
    ports:
      - "127.0.0.1:$HOST_PORT:8080"
    networks:
      - default
    deploy:
      resources:
        limits:
          memory: 256M
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8080/api/v1/health >/dev/null 2>&1 || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 10
      start_period: 20s

volumes: {}

networks:
  default:
    external: true
    name: $NETWORK
EOF

echo "==> 启动后端容器"
cd "$DOCKER_DIR" && docker compose -p "hrms-$COMPANY" -f "$COMPOSE" up -d

echo "==> 等待后端健康"
for i in $(seq 1 30); do
  if docker exec hrms-backend-$COMPANY wget -qO- http://127.0.0.1:8080/api/v1/health >/dev/null 2>&1; then
    echo "    后端 healthy（第 ${i} 次探测）"
    break
  fi
  if [ "$i" = "30" ]; then echo "错误: 后端未在 60s 内健康"; exit 1; fi
  sleep 2
done

echo "==> 复制前端到 $WEB_DIR"
mkdir -p "$WEB_DIR"
cp -r /opt/hrms/web/* "$WEB_DIR/"

echo "==> 添加 Nginx server 块"
cat > "/etc/nginx/sites-enabled/hrms-$COMPANY.conf" <<EOF
server {
    listen 80;
    server_name $DOMAIN;
    client_max_body_size 50m;

    root $WEB_DIR;
    index index.html;

    # CORS：桌面端 file:// 与 Android WebView 跨源访问
    add_header Access-Control-Allow-Origin * always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Authorization, Content-Type" always;
    add_header Access-Control-Max-Age 86400 always;

    location /api/ {
        proxy_pass http://127.0.0.1:$HOST_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        if (\$request_method = OPTIONS) { return 204; }
    }

    location /updates/ {
        alias /opt/hrms/updates/;
        add_header Content-Disposition 'attachment; filename="\$1"';
    }

    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
EOF
nginx -t && systemctl reload nginx

echo "==> 完成: 公司 $COMPANY 已创建，访问 http://$DOMAIN（后端端口 $HOST_PORT）"
echo "    注意: 域名需先解析到本机并完成 ICP 备案才可公网访问；IP 直连仍走主服务"
