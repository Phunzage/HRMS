#!/usr/bin/env bash
# HRMS 一键启动脚本（git-bash / WSL 使用）
# 用法: ./start.sh    （Ctrl+C 可同时停止前后端）
set -e
cd "$(dirname "$0")"

echo "============================================"
echo "   HRMS 人力资源管理系统 - 一键启动"
echo "============================================"

# 检查 PostgreSQL
if sc query postgresql-x64-18 >/dev/null 2>&1 || net start 2>/dev/null | grep -qi postgres; then
    echo "[OK] PostgreSQL 服务运行中"
else
    echo "[!] 未检测到 PostgreSQL 服务，请先启动"
fi

# 启动后端
echo "[1/2] 启动后端 :8080 ..."
(cd backend && go run ./cmd/server) &
BACK_PID=$!

# 启动前端
echo "[2/2] 启动前端 :5173 ..."
(cd frontend && npm run dev) &
FRONT_PID=$!

# 退出时清理两个进程
trap 'echo; echo "正在停止服务..."; kill $BACK_PID $FRONT_PID 2>/dev/null; exit 0' INT TERM

# 等待就绪后打开浏览器
sleep 10
cmd //c start "" http://localhost:5173 2>/dev/null || true

echo "浏览器已打开 http://localhost:5173"
echo "内置账号: admin/admin123(管理员)  hr/hr123(人事)  zhangwei/123456(经理)  liuyang/123456(员工)"
echo "Ctrl+C 停止所有服务"

wait
