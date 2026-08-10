# 任务批次 38M-E：发布 v1.0.2 — 部署新前端 + 重新打包两端 + 登记版本

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。移动端适配已完成（批次 A-D，前端已改），现在**发布 v1.0.2**：部署新前端到服务器 → 重新打包 Android APK 与 Windows exe → 上传登记。中文界面。

## 关键背景
- 服务器：115.29.197.130（ssh aliyun-server），Nginx 静态托管 `/opt/hrms/web/`，更新包目录 `/opt/hrms/updates/`
- **Windows 桌面端 main.js 生产模式是 `loadURL('http://115.29.197.130')` 直接加载线上前端** → 只要线上 web 更新，已安装的 Windows 端**立即**用上新 UI（无需重装）；但 exe 安装包仍需重打（版本号/build 递增，供新装机与更新机制）
- Android 端前端**打进 APK** → 必须重新构建打包
- `.env.production` 已有 `VITE_API_BASE=http://115.29.197.130`

## 一、修复遗留小项（先做）
`frontend/src/pages/LeavePage.tsx` 请假事由 textarea 补 `w-full`（批次 D 遗留，daisyUI textarea 不随 Field 拉伸）

## 二、构建前端（生产）
```bash
cd frontend && npm run build
```
- 产物 `dist/`（含移动端适配 + VITE_API_BASE=http://115.29.197.130）

## 三、部署新前端到服务器（线上 web + Windows 端立即生效）
```bash
# 上传
scp -r frontend/dist/* aliyun-server:/tmp/web-new/
# 服务器替换（保留旧版备份）
ssh aliyun-server "sudo rm -rf /opt/hrms/web.bak && sudo cp -r /opt/hrms/web /opt/hrms/web.bak && sudo rm -rf /opt/hrms/web && sudo mkdir -p /opt/hrms/web && sudo cp -r /tmp/web-new/* /opt/hrms/web/ && sudo chown -R root:root /opt/hrms/web && ls /opt/hrms/web/"
```
- 验证：`curl -s http://115.29.197.130/ | grep -o 'assets/index-[^"]*\.js'` 应看到**新的 hash**（与本地 dist 一致）

## 四、重新打包 Android APK v1.0.2
1. 版本升级：`frontend/android/app/build.gradle`：
   - `versionName "1.0.2"`、`versionCode 3`
2. 同步 + 构建：
```bash
cd frontend
npx cap sync android
cd android
export ANDROID_HOME="C:\Users\Phunzage\AppData\Local\Android\Sdk"
export JAVA_HOME="/c/Program Files/Android/Android Studio/jbr"
./gradlew assembleRelease
```
3. 产物 `app/build/outputs/apk/release/app-release.apk`
4. 验证 APK 内 assets 含移动端适配代码：`unzip -p app-release.apk "assets/public/index.html" | grep -c "viewport"`；`unzip -p` 解出 assets/public/assets/*.js grep `-translate-x-full`（抽屉类）确认新前端已进包

## 五、重新打包 Windows exe v1.0.2
1. `desktop/package.json` version 改 `1.0.2`（build 3）
2. ```bash
cd desktop && npx electron-builder --win nsis
```
3. 产物 `dist/hrms-desktop Setup 1.0.2.exe`
4. 说明：exe 内 build-web 仍是旧前端，但**运行时 loadURL 线上**（新前端已部署），功能无碍；如需 build-web 同步可后续更新

## 六、上传 + 登记 v1.0.2
```bash
scp desktop/dist/hrms-desktop\ Setup\ 1.0.2.exe aliyun-server:/opt/hrms/updates/hrms-win-1.0.2.exe
scp frontend/android/app/build/outputs/apk/release/app-release.apk aliyun-server:/opt/hrms/updates/hrms-android-1.0.2.apk
```
登记（服务器 psql，注意表只有 created_at 无 updated_at）：
```sql
INSERT INTO app_versions (platform, version, build_number, url, release_notes, created_at)
VALUES
('win','1.0.2',3,'http://115.29.197.130/updates/hrms-win-1.0.2.exe','移动端全面适配', now()),
('android','1.0.2',3,'http://115.29.197.130/updates/hrms-android-1.0.2.apk','移动端全面适配', now());
```
（build_number 3 > 现有 2，更新机制会提示升级）

## 七、验证清单
1. 线上 web 新 hash：`curl -s http://115.29.197.130/ | grep -o 'assets/index-[^"]*\.js'` 与本地 `frontend/dist/assets/index-*.js` 一致
2. `curl "http://115.29.197.130/api/v1/update/check?platform=win&build=2"` → has_update:true（1.0.2/build3）
3. `curl "http://115.29.197.130/api/v1/update/check?platform=android&build=2"` → has_update:true
4. `curl -I http://115.29.197.130/updates/hrms-win-1.0.2.exe` → 200
5. `curl -I http://115.29.197.130/updates/hrms-android-1.0.2.apk` → 200
6. APK 内 assets 含新前端（抽屉类/移动断点）

## 汇报格式
LeavePage 修复、前端构建、线上部署（新旧 hash 对比）、APK/exe 路径与大小、上传登记、验证矩阵、遗留问题。
