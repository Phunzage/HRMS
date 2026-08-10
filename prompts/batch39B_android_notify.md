# 任务批次 39B：Android 系统通知（前台+后台轮询推送）

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 39A 已完成：后端 `GET /notifications/recent` + Windows Electron 系统通知（IPC notify）。本批次实现 **Android 端系统通知**。中文界面。

## 需求
Android 上像微信一样弹系统通知：员工请假 → 上级（经理/HR）**即使不打开 App**也能在系统通知栏看到「新的请假申请待审批」。

## 方案

### 前台（App 打开时）：WebView 内 JS 轮询 + Capacitor 本地通知
1. 安装插件：`cd frontend && npm install @capacitor/local-notifications && npx cap sync android`
2. 渲染进程 `useSystemNotify` hook（批次 39A 已建）增强：检测到新通知时，**Electron 环境用 window.hrmsUpdater.notify，Capacitor 环境用 LocalNotifications.schedule**：
```ts
import { Capacitor } from '@capacitor/core'
import { LocalNotifications } from '@capacitor/local-notifications'
// 判断：Capacitor.isNativePlatform() 为 true 时用 LocalNotifications
if (Capacitor.isNativePlatform()) {
  await LocalNotifications.schedule({
    notifications: [{ id: latestId, title: 'HRMS 新通知', body: recent[0].title, smallIcon: 'ic_launcher' }],
  })
}
```
- **权限**：Android 13+ 需动态请求 POST_NOTIFICATIONS——`LocalNotifications.requestPermissions()`（MainActivity 已声明权限，运行时请求一次；App 启动时或首次检测到通知时调用）
- 通知渠道：schedule 时用 `channelId`（如 'hrms'）——插件默认渠道可配置，或先不指定用默认
- 点击通知打开 App：LocalNotifications.addListener('localNotificationReceived') 可后续增强（先做基础弹出）

### 后台（App 退到后台/锁屏）：原生层轮询（WebView 会暂停，JS 轮询失效）
**关键**：Android 后台 WebView 停止执行 JS，必须原生层独立轮询。方案：

1. **原生 Service**（`frontend/android/app/src/main/java/com/hrms/app/NotifyService.java`）：
   - 前台服务（`startForeground` + 常驻通知「HRMS 运行中」）——**需要用户感知，但可靠**
   - 或 WorkManager 周期任务（最小 15 分钟间隔，系统可能延迟）——省电但不即时
   - **选择**：用 `WorkManager` 周期任务（每 15 分钟）+ 启动时立即检查一次；为即时性，MainActivity onResume 也检查
2. Service 逻辑：
   - 从 SharedPreferences 读 token（登录时存：前端 localStorage 的值——**注意**：WebView localStorage 与原生 SharedPreferences 不通，需通过 Capacitor 插件桥接或简单方案见下）
   - 轮询 `GET http://115.29.197.130/api/v1/notifications/recent?limit=1`（带 Authorization）
   - 对比上次 id → 有新 → `NotificationManager` 发系统通知（复用 POST_NOTIFICATIONS）
3. **token 桥接问题（关键）**：登录 token 存在 WebView localStorage，原生层拿不到。方案：
   - 写一个 Capacitor 插件 `TokenStore`（`registerPlugin` + Android 实现读写 SharedPreferences）：前端登录成功后 `TokenStore.set({token})`；Service 从 SharedPreferences 读 token
   - **实现**：`frontend/src/api/tokenBridge.ts` 封装 + 原生 `TokenStorePlugin.java` + `TokenStore.java`（Capacitor 插件注册）；前端 auth store 在 token 变更时同步写入
4. 周期任务注册：MainActivity.onCreate 里 `WorkManager.enqueueUniquePeriodicWork(...)`（15 分钟周期，token 存在才真正请求）

### 简化决策（避免过度复杂）
- 后台轮询最小可行：WorkManager 15 分钟周期 + onResume 即时检查；前台由 WebView JS 轮询（30s）覆盖
- 若 WorkManager 实现复杂，可先只做**前台**（App 打开/后台短时间），后台长时推送作为增强——但用户核心诉求是"不打开也能收到"，所以后台 Service 是必须的。**优先完成 WorkManager 后台轮询**
- 通知点击打开 App：PendingIntent 指向 MainActivity（默认即如此）

## 三、完成后必须执行
1. `cd frontend && npm run build` 0 错误
2. `npx cap sync android`
3. gradle 构建：`export ANDROID_HOME="C:\Users\Phunzage\AppData\Local\Android\Sdk" && export JAVA_HOME="/c/Program Files/Android/Android Studio/jbr" && cd android && ./gradlew assembleRelease`
4. 版本升 `1.0.4` / versionCode 5（build.gradle）
5. 上传 + 登记（hrms-android-1.0.4.apk，build_number 5，release_notes「系统级推送通知」）
6. 验证：
   - aapt badging 确认版本（**用 Windows 路径**，MSYS 路径 aapt 打不开）
   - 代码走查：TokenStore 插件注册、NotifyService/WorkManager 存在、前端 LocalNotifications 调用
   - 若本机有模拟器/设备可实测（无则说明，靠代码走查）
7. 停掉进程，端口释放

## 汇报格式
插件安装、TokenStore 桥接、前台 JS 通知、后台 WorkManager Service、版本构建上传登记、验证结果、遗留（权限/点击跳转/后台周期精度）。
