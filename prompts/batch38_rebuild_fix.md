# 任务批次 38：客户端白屏/验证码失败修复 + 重新打包

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户实测：Windows 白屏、Android 验证码加载失败。架构师已定位双根因并完成代码修改，你负责**验证修复 + 重新打包 + 上传登记**。

## 根因（架构师已确认）

1. **Windows 白屏**：`<script type="module" crossorigin>` 在 `file://` 协议下被 Chromium **强制 CORS 检查**（origin=null 永远失败）→ JS 从未执行 → 白屏。与服务器 CORS 无关（服务器 CORS 已加但不解决此问题）。
   - **修复已做**：`desktop/electron/main.js` 生产模式改 `loadURL(PROD_URL)`（`http://115.29.197.130`，同源 http，无 file:// 限制）
2. **Android 验证码失败**：Capacitor `androidScheme:'https'` 页面（`https://localhost`）fetch `http://` API = **混合内容**，WebView 的 `setMixedContentMode` 对 fetch/XHR 无效 → 请求被拦。
   - **修复已做**：`frontend/capacitor.config.ts` `androidScheme` 改为 `'http'`（本地 `http://localhost` 与线上 http API 同协议，无混合内容）

## 你的任务

### 1. 验证代码修改正确性
- 检查 `desktop/electron/main.js`：PROD_URL 定义、loadURL(PROD_URL) 分支
- 检查 `frontend/capacitor.config.ts`：androidScheme='http'
- 确认无误后进入打包

### 2. 重新打包 Windows（Electron）
```bash
cd desktop && npx electron-builder --win nsis
```
- 产物 `dist/hrms-desktop Setup 1.0.1.exe`（版本号升到 1.0.1，build 2——package.json version 改 1.0.1，让更新机制能识别新版本）
- 注意：main.js 引用线上 URL，**不需要重新构建前端**（build-web 仍打包但不再被加载，保留无妨）
- 本机快速验证：`npx electron .` 开发模式仍连 localhost:5173（跳过）；直接运行 win-unpacked 里的 exe 验证加载线上页面（无头跑或日志验证 did-navigate 到 http://115.29.197.130）

### 3. 重新打包 Android（Capacitor）
```bash
cd frontend
npm run build            # 生产构建（VITE_API_BASE=http://115.29.197.130 已在 .env.production）
npx cap sync android     # 同步配置（androidScheme=http 生效）
cd android
export ANDROID_HOME="C:\Users\Phunzage\AppData\Local\Android\Sdk"
export JAVA_HOME="/c/Program Files/Android/Android Studio/jbr"
./gradlew assembleRelease
```
- 版本升到 1.0.1（build.gradle versionName "1.0.1" versionCode 2）
- 产物 `app/build/outputs/apk/release/app-release.apk`
- 用 `unzip -p` 或 aapt 验证 AndroidManifest 里 application 有 `usesCleartextTraffic="true"`

### 4. 上传 + 登记新版本
```bash
scp desktop/dist/hrms-desktop\ Setup\ 1.0.1.exe aliyun-server:/opt/hrms/updates/hrms-win-1.0.1.exe
scp frontend/android/app/build/outputs/apk/release/app-release.apk aliyun-server:/opt/hrms/updates/hrms-android-1.0.1.apk
```
- 登记（自签 admin JWT 调 POST /api/v1/update/versions，或 psql 插入）：
  - win: version 1.0.1, build 2, url http://115.29.197.130/updates/hrms-win-1.0.1.exe
  - android: version 1.0.1, build 2, url http://115.29.197.130/updates/hrms-android-1.0.1.apk
- 服务器 psql 方法：`sudo -u postgres psql -d hrms -c "INSERT INTO app_versions (platform, version, build_number, url, release_notes, created_at, updated_at) VALUES ('win','1.0.1',2,'http://115.29.197.130/updates/hrms-win-1.0.1.exe','修复白屏：直接加载线上前端', now(), now());"`（android 同理）

### 5. 验证
1. `curl -s "http://115.29.197.130/api/v1/update/check?platform=win&build=1"` → has_update:true（新 1.0.1/build=2）
2. `curl -s "http://115.29.197.130/api/v1/update/check?platform=android&build=1"` → has_update:true
3. `curl -I http://115.29.197.130/updates/hrms-win-1.0.1.exe` → 200
4. 用本机浏览器打开 http://115.29.197.130/ 确认登录页 + 验证码正常（模拟两端加载的同一前端）
5. 汇报两端新包路径与大小

## 汇报格式
代码修改确认、Windows 打包产物（路径/大小/版本）、Android APK（路径/大小/版本/manifest 确认）、上传登记结果、更新 API 双平台验证、遗留问题。
