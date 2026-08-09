# 任务批次 37：Android APK（Capacitor 包装）+ Android 端随机更新

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 35 已部署服务器（更新 API 支持 platform=android）；批次 36 做 Electron。本批次构建 **Android APK**（Capacitor 包装现有 React 应用，无需重写）并实现 Android 端更新机制。中文界面。

## 一、环境（已确认可用）
- Android Studio 已装：`C:\Program Files\Android\Android Studio`
- JDK：Studio 自带 JBR（`C:\Program Files\Android\Android Studio\jbr\bin\java.exe`，OpenJDK 21）
- SDK：`C:\Users\Phunzage\AppData\Local\Android\Sdk`（platforms: android-34/36.1，build-tools: 34/36）
- 环境变量：ANDROID_HOME 未设——**构建时显式设置**（`export ANDROID_HOME="C:\Users\Phunzage\AppData\Local\Android\Sdk"`、`export JAVA_HOME="C:\Program Files\Android\Android Studio\jbr"`、PATH 加 platform-tools）

## 二、Capacitor 集成

1. **安装**：`cd frontend && npm install @capacitor/core @capacitor/cli @capacitor/android`
2. **配置** `capacitor.config.ts`：
```ts
import type { CapacitorConfig } from '@capacitor/cli'
const config: CapacitorConfig = {
  appId: 'com.hrms.app',
  appName: 'HRMS人力资源管理系统',
  webDir: 'dist',
  server: { androidScheme: 'https' },
}
export default config
```
3. **初始化**：`npx cap init HRMS com.hrms.app --web-dir dist`（若配置已写则跳过）；`npx cap add android`
4. **前端 API 地址**：Android WebView 中请求 `http://115.29.197.130` 需允许明文流量（android:usesCleartextTraffic="true" 加到 AndroidManifest）或走 https（暂无证书）——**用 cleartext 方案**；前端 `.env.production` 设 `VITE_API_BASE=http://115.29.197.130`（与批次 36 一致）
5. **构建**：`npm run build`（生产）→ `npx cap sync android`
6. **出 APK**（gradle 构建，用 Studio JBR）：
```bash
export ANDROID_HOME="C:\Users\Phunzage\AppData\Local\Android\Sdk"
export JAVA_HOME="C:\Program Files\Android\Android Studio\jbr"
cd frontend/android && ./gradlew assembleRelease
```
- 产物：`frontend/android/app/build/outputs/apk/release/app-release.apk`
- 签名：release 用 debug keystore（开发期可接受）——**在 android/app/build.gradle 确认 signingConfig 用 debug 或生成自签 keystore**（简单起见 release 复用 debug 签名即可安装测试）

## 三、Android 端更新机制

### 方案：原生代码实现（Java/Kotlin，MainActivity 扩展）
1. 启动后 `onCreate` 中启动检查（延迟 5 秒）：
   - `fetch http://115.29.197.130/api/v1/update/check?platform=android&version=<versionName>&build=<versionCode>`
   - 版本来自 `BuildConfig.VERSION_NAME` / `VERSION_CODE`（android/app/build.gradle 里 versionName="1.0.0" versionCode=1）
2. **随机间隔轮询**：首次检查完成后，`Handler.postDelayed` 30~60 分钟随机（`30000 + new Random().nextInt(1800000)` 毫秒）排下次检查——与 Windows 端一致
3. **发现更新**：
   - 用 Android `NotificationCompat` 发通知（渠道 "updates"）：「发现新版本 v1.1.0」
   - 点击通知 → `DownloadManager` 下载 APK（url 来自 API）→ 完成广播 → 调 `PackageInstaller`/`Intent(ACTION_VIEW)` 打开 APK 触发安装（需 `REQUEST_INSTALL_PACKAGES` 权限，AndroidManifest 声明）
   - 或简单方案：`WebView` 渲染进程 JS 检测后提示——**选择原生通知+DownloadManager 方案**（离线也能提示）
4. **权限**：AndroidManifest 加 `<uses-permission android:name="android.permission.INTERNET"/>`、`REQUEST_INSTALL_PACKAGES`、`POST_NOTIFICATIONS`（Android 13+ 动态申请——MainActivity onCreate 请求）
5. **实现位置**：`frontend/android/app/src/main/java/com/hrms/app/` 下新建 `Updater.java`（或 Kotlin），MainActivity onCreate 调用 `Updater.start(this)`

## 四、打包与上传
1. `./gradlew assembleRelease` 产出 APK
2. 上传：`scp app-release.apk aliyun-server:/opt/hrms/updates/hrms-android-1.0.0.apk`
3. 登记版本：psql 插入 app_versions：platform=android, version=1.0.0, build=1, url=http://115.29.197.130/updates/hrms-android-1.0.0.apk（若批次 35 已做管理 API 则用 API）

## 五、验证
1. APK 构建成功、大小合理（几 MB~几十 MB）
2. `aapt dump badging app-release.apk`（SDK build-tools 里）确认 package/version/权限
3. 若本机有 Android 模拟器或连接的设备：安装测试启动（无则跳过，说明）
4. 更新 API：`curl "http://115.29.197.130/api/v1/update/check?platform=android&version=1.0.0&build=0"` → has_update:true（build=1 已登记）→ url 指向 apk
5. `curl -I http://115.29.197.130/updates/hrms-android-1.0.0.apk` → 200

## 汇报格式
Capacitor 集成步骤、gradle 构建命令与产物、Updater 实现（随机间隔/通知/下载安装）、APK 上传登记结果、验证输出、遗留问题（签名/图标/商店发布说明）。
