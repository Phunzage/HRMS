#!/usr/bin/env bash
# 删除一家公司实例（停容器 + 删数据库 + 删 Nginx conf + 删前端目录）
# 用法: sudo ./remove-company.sh <公司标识>
set -euo pipefail

COMPANY="${1:?用法: remove-company.sh <公司标识>}"
DOCKER_DIR="/opt/hrms/docker"
COMPOSE="$DOCKER_DIR/docker-compose.$COMPANY.yml"

echo "==> 停止并移除容器"
docker compose -p "hrms-$COMPANY" -f "$COMPOSE" down --rmi local 2>/dev/null || true
docker rm -f "hrms-backend-$COMPANY" >/dev/null 2>&1 || true

echo "==> 删除数据库 hrms_$COMPANY"
docker exec hrms-db psql -U postgres -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS hrms_$COMPANY;" || echo "    数据库删除失败(可能无权限/连接占用，可手动处理)"

echo "==> 删除 compose 与 Nginx conf"
rm -f "$COMPOSE" "/etc/nginx/sites-enabled/hrms-$COMPANY.conf"
rm -rf "/opt/hrms/web-$COMPANY"

nginx -t && systemctl reload nginx
echo "==> 完成: 公司 $COMPANY 已删除"
