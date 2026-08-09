// 更新检查与下载逻辑（随机时间拉取更新机制）
// 设计要点：
//   1. 应用启动后延迟 5 秒执行首次检查；
//   2. 每次检查完成后，生成 30~60 分钟随机间隔，用 setTimeout 排下次检查，
//      使所有客户端的拉取时间错开（避免同时请求服务器形成脉冲）；
//   3. 检查到新版本时向渲染进程推送 update:status 事件 + 系统通知；
//   4. 用户确认后下载完整安装包到「下载」目录，打开所在目录让用户手动安装。
const { app, Notification, shell, ipcMain } = require('electron')
const fs = require('fs')
const path = require('path')
const { URL } = require('url')

// 生产服务器地址（可用环境变量覆盖，便于本机测试指向其他环境）
const API_BASE = process.env.HRMS_UPDATE_API || 'http://115.29.197.130'
// 首次检查延迟：启动后 5 秒
const FIRST_CHECK_DELAY = 5 * 1000
// 随机间隔区间：30~60 分钟
const MIN_INTERVAL = 30 * 60 * 1000
const RANDOM_RANGE = 30 * 60 * 1000

let mainWindow = null
let checkTimer = null
let state = 'idle' // idle / checking / update-available / downloading / downloaded / error
let latest = null // 服务器返回的最新版本信息

// 当前版本信息：版本号 + 构建号（buildNumber 可用 HRMS_CLIENT_BUILD 覆盖，便于模拟旧版测试）
// 用 app.getAppPath() 定位 package.json：开发时为项目根目录，打包后为 asar 根目录，
// 相对 require 在 asar 内不可靠（Cannot find module '../../package.json'）
function currentVersion() {
  const pkg = require(path.join(app.getAppPath(), 'package.json'))
  const build = Number(process.env.HRMS_CLIENT_BUILD ?? pkg.buildNumber ?? 1)
  return { version: pkg.version, build }
}

// 向渲染进程推送更新状态事件（渲染进程右下角提示条据此渲染）
function emit(data) {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('update:status', data)
  }
}

// 安排下一次检查：30~60 分钟随机间隔 —— 「随机时间拉取」核心
function scheduleNext() {
  if (checkTimer) clearTimeout(checkTimer)
  const delay = MIN_INTERVAL + Math.floor(Math.random() * RANDOM_RANGE)
  const minutes = Math.round(delay / 60000)
  console.log(
    `[updater] 本次检查完成，下次更新检查约 ${minutes} 分钟后（${new Date(Date.now() + delay).toLocaleString('zh-CN')}）`,
  )
  checkTimer = setTimeout(check, delay)
}

// 执行一次更新检查（GET /api/v1/update/check?platform=win&version=&build=）
async function check() {
  const { version, build } = currentVersion()
  state = 'checking'
  emit({ status: 'checking', message: '正在检查更新…' })
  try {
    const url = `${API_BASE}/api/v1/update/check?platform=win&version=${encodeURIComponent(version)}&build=${build}`
    const res = await fetch(url, { signal: AbortSignal.timeout(15000) })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const body = await res.json()
    // 后端统一响应体 { code, message, data }
    if (body.code === 0 && body.data && body.data.has_update) {
      latest = body.data.latest
      state = 'update-available'
      emit({
        status: 'update-available',
        version: latest.version,
        build: latest.build_number,
        url: latest.url,
        releaseNotes: latest.release_notes,
      })
      // 系统通知（Windows 需先 setAppUserModelId，否则不弹）
      if (Notification.isSupported()) {
        new Notification({
          title: '发现新版本',
          body: `HRMS 新版本 v${latest.version} 已发布，点击下载更新`,
        }).show()
      }
    } else {
      latest = null
      state = 'idle'
      emit({ status: 'idle', message: '当前已是最新版本' })
    }
  } catch (e) {
    state = 'error'
    emit({ status: 'error', message: `检查更新失败：${e.message}` })
  } finally {
    scheduleNext()
  }
}

// 下载更新包（完整安装包）到「下载」目录，完成后打开所在目录提示手动安装
async function download() {
  if (!latest || !latest.url) {
    emit({ status: 'error', message: '暂无可用更新' })
    return { ok: false, message: '暂无可用更新' }
  }
  state = 'downloading'
  emit({ status: 'downloading', version: latest.version, percent: 0 })
  let lastPct = -1
  try {
    const dir = app.getPath('downloads')
    const fileName = path.basename(new URL(latest.url).pathname) || `hrms-win-${latest.version}.exe`
    const target = path.join(dir, fileName)
    const res = await fetch(latest.url)
    if (!res.ok || !res.body) throw new Error(`HTTP ${res.status}`)
    const total = Number(res.headers.get('content-length') || 0)
    const reader = res.body.getReader()
    const out = fs.createWriteStream(target)
    let received = 0
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      received += value.length
      if (!out.write(value)) await new Promise((resolve) => out.once('drain', resolve))
      if (total) {
        const pct = Math.round((received / total) * 100)
        // 仅在进度变化时推送，避免高频 IPC 淹没渲染进程
        if (pct !== lastPct) {
          lastPct = pct
          emit({ status: 'downloading', version: latest.version, percent: pct })
        }
      }
    }
    out.end()
    await new Promise((resolve, reject) => {
      out.on('finish', resolve)
      out.on('error', reject)
    })
    state = 'downloaded'
    emit({ status: 'downloaded', path: target, version: latest.version })
    console.log(`[updater] 安装包已下载：${target}`)
    // 打开所在目录，让用户运行安装包（简单可靠的手动安装方案）
    shell.openPath(dir)
    return { ok: true, path: target }
  } catch (e) {
    state = 'error'
    emit({ status: 'error', message: `下载失败：${e.message}` })
    return { ok: false, message: e.message }
  }
}

// 注册 IPC 处理器（渲染进程经 preload 白名单调用）
function registerIpc() {
  ipcMain.handle('update:get-info', () => ({ state, latest, ...currentVersion() }))
  ipcMain.handle('update:check-now', () => check())
  ipcMain.handle('update:download', () => download())
}

// 初始化：绑定主窗口、注册 IPC、启动首次检查（延迟 5 秒）
function init(win) {
  mainWindow = win
  registerIpc()
  const { version, build } = currentVersion()
  console.log(
    `[updater] API_BASE=${API_BASE}，当前版本 v${version} (build ${build})，${FIRST_CHECK_DELAY / 1000} 秒后首次检查`,
  )
  setTimeout(check, FIRST_CHECK_DELAY)
}

module.exports = { init, checkNow: () => check() }
