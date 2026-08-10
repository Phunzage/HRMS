# 任务批次 39C：修复权限重启重置 + Windows 端系统通知（重打包）

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。两个问题，根因已由架构师定位，按方案修复。

## 问题 1：角色权限每次更新/重启后被重置

**根因**：`backend/internal/database/seed.go` 的 `seedRolePermissions`：
- 对每个角色遍历默认权限 codes，**缺失就补全**（约 234-242 行）
- hr 默认配置写死了 `leave:approve` / `attendance_correction:approve`（188-203 行）
- 用户在角色管理界面**取消** hr 的审批权限 → role_permissions 表删行 → 下次启动 seed 发现"缺失"→ **重新补回** → 权限被还原

### 修复：seed 权限只初始化首次，不覆盖后续修改
把「缺失补全」逻辑改为「**角色从未配置过任何权限时才初始化默认权限**」：
1. 对每个角色，先查 `SELECT count(*) FROM role_permissions WHERE role_id = ?`
2. 若 count > 0（该角色已被配置过，无论谁配的）→ **完全跳过**（不补全、不清理）
3. 若 count == 0（全新角色）→ 写入该角色默认权限
4. **移除「清理」逻辑**（248 行附近 Delete 那段）——不再强制与种子一致
5. 注释更新：说明「默认权限仅在角色首次创建时写入；之后由 admin 在角色管理维护，重启不还原」

验证：
- 删除 hr 的 leave:approve 关联 → 重启后端 → hr 权限**保持被删状态**（不再还原）✅
- 新建一个角色（无任何权限）→ 重启 → 不会被 seed 写入（非种子角色本就无默认）✅
- 内置角色 admin/hr/manager/employee 首次初始化仍生效（现有库已初始化过，count>0 跳过，保持现状）✅
- `go build ./...`、`go test ./...` 全绿

## 问题 2：Windows 桌面端收不到系统通知

**根因**：批次 39A 的 Electron 代码（`desktop/electron/notify.js` + main.js IPC + preload notify）已改，但 **exe 未重新打包**——用户桌面上是旧 exe（10:27 打包，notify.js 11:58 才创建），主进程无 notify IPC、preload 无 notify → 前端 `window.hrmsUpdater?.notify` undefined → 不弹窗。

### 修复：重新打包 Windows exe v1.0.5
1. `desktop/package.json` version 改 `1.0.5`（build 6，高于 android 1.0.4/build5 保持单调递增即可，或独立计数——**按平台各自递增**：win 从 build 3 → 4）
   - 检查服务器 app_versions 里 win 当前最大 build_number，+1
2. ```bash
cd desktop && npx electron-builder --win nsis
```
3. 产物 `dist/hrms-desktop Setup 1.0.5.exe`
4. **验证 exe 含 notify**：`npx asar list dist/win-unpacked/resources/app.asar | grep notify` 应看到 notify.js
5. 本机快速验证：运行 win-unpacked 的 exe（或 `npx electron .` 开发模式连本地后端）→ 制造一条新通知（如另一账号提交请假给当前账号）→ 30s 内 Windows 通知中心弹出「新的请假申请待审批」
6. 上传 `hrms-win-1.0.5.exe` 到 `/opt/hrms/updates/` + psql 登记（build_number = win 现有 +1，url 指向新文件，release_notes「Windows系统通知」）
7. `curl "http://115.29.197.130/api/v1/update/check?platform=win&build=旧值"` → has_update:true

### 顺带确认（不做大改，仅检查）
- Android v1.0.4（39B）已含通知插件与前台 JS 轮询——若 Android 也收不到，检查：线上前端 JS 是否含 LocalNotifications（已确认含）；用户是否授权通知权限（Android 13+）；APK 内插件（已确认有 capacitor）。**若 Android 真机仍收不到**，先确认用户是否拒绝过权限，再考虑 TokenStore/WorkManager 是否正常。

## 汇报格式
seed 修复前后行为对比（取消权限→重启→保持取消）、exe 重打包产物与 asar 验证、Windows 通知实测、版本登记、测试结果。
