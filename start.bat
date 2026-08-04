@echo off
chcp 65001 >nul
title HRMS 人力资源管理系统 - 一键启动
setlocal

echo ============================================
echo    HRMS 人力资源管理系统 - 一键启动
echo ============================================
echo.

REM 切换到脚本所在目录（项目根目录）
cd /d "%~dp0"

REM 1. 预检：检查 PostgreSQL 服务是否运行
echo [1/4] 检查 PostgreSQL 服务...
sc query postgresql-x64-18 >nul 2>&1
if errorlevel 1 (
    echo      [!] 未检测到 postgresql-x64-18 服务，请先启动 PostgreSQL！
) else (
    echo      [OK] PostgreSQL 服务运行中
)

REM 2. 预检：检查 go / npm
where go >nul 2>&1 || (echo      [!] 未找到 go，请安装 Go 1.25+ & goto :fail)
where npm >nul 2>&1 || (echo      [!] 未找到 npm，请安装 Node 20+ & goto :fail)
echo      [OK] Go 与 Node 环境就绪

REM 3. 启动后端（独立窗口，标题 HRMS-Backend）
echo [2/4] 启动后端服务 Go+Gin，监听 :8080 ...
start "HRMS-Backend" cmd /k "cd /d %~dp0backend && go run ./cmd/server"

REM 4. 启动前端（独立窗口，标题 HRMS-Frontend）
echo [3/4] 启动前端 Vite，监听 :5173 ...
start "HRMS-Frontend" cmd /k "cd /d %~dp0frontend && npm run dev"

REM 5. 等待服务就绪后自动打开浏览器
echo [4/4] 等待服务启动...
timeout /t 10 /nobreak >nul
start "" http://localhost:5173

echo.
echo ============================================
echo  启动完成！浏览器已打开 http://localhost:5173
echo.
echo  内置账号：
echo    admin     / admin123  (管理员)
echo    hr        / hr123     (人事)
echo    zhangwei  / 123456    (经理)
echo    liuyang   / 123456    (员工)
echo.
echo  停止服务：直接关闭 HRMS-Backend 与 HRMS-Frontend 两个窗口即可
echo ============================================
echo.
pause
exit /b 0

:fail
echo.
echo 环境检查未通过，请安装缺失组件后重试。
pause
exit /b 1
