// Electron 主进程：创建主窗口、单实例锁、中文菜单、加载前端、初始化更新模块
const { app, BrowserWindow, Menu, shell, ipcMain, Notification } = require('electron')
const path = require('path')
const { pathToFileURL } = require('url')
const updater = require('./updater')
const { registerNotifyIpc } = require('./notify')

// 开发模式加载 vite dev server，生产模式直接加载线上前端（http 同源，避免 file:// 限制）
const DEV_URL = 'http://localhost:5173'
// 生产：线上前端地址（同源 http，无 file:// 下 module script CORS 问题，API 请求同源）
const PROD_URL = 'http://115.29.197.130'
// 用 app.getAppPath()（asar 根目录）拼路径，再用 pathToFileURL 显式转成 file:// URL；
// loadFile 直接传 asar 内路径在本机出现 URL 错乱（file:///C:/），故用 loadURL 显式给定
const WEB_INDEX = path.join(app.getAppPath(), 'build-web', 'index.html')

let mainWindow = null

// 单实例锁：重复启动时聚焦已有窗口，而不是再开一个实例
const gotLock = app.requestSingleInstanceLock()
if (!gotLock) {
  app.quit()
} else {
  app.on('second-instance', () => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore()
      mainWindow.focus()
    }
  })

  // Windows 通知/任务栏需要显式设置 AppUserModelID，否则通知不显示
  app.setAppUserModelId('com.hrms.desktop')

  app.whenReady().then(() => {
    createWindow()
    // 彻底移除默认/自定义应用菜单栏（「文件/视图/帮助」及功能栏），窗口更简洁；
    // 更新检查入口由渲染进程「个人中心→检查更新」按钮调用 window.hrmsUpdater.checkNow() 替代菜单项
    Menu.setApplicationMenu(null)
    // 系统通知 IPC（Windows 通知中心），依赖上方 setAppUserModelId
    registerNotifyIpc(ipcMain, Notification)
    // 注册 IPC 处理器并启动「随机间隔更新检查」机制
    updater.init(mainWindow)
  })
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1024,
    minHeight: 700,
    title: 'HRMS人力资源管理系统',
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true, // 渲染进程与主进程隔离
      nodeIntegration: false, // 渲染进程不开放 Node 能力
    },
  })

  // 外链（如更新下载打开外部浏览器）一律交给系统默认浏览器
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url)
    return { action: 'deny' }
  })

  // 调试：追踪加载/导航全过程
  mainWindow.webContents.on('did-navigate', (_e, url) => console.log('[did-navigate]', url))
  mainWindow.webContents.on('did-navigate-in-page', (_e, url) => console.log('[did-navigate-in-page]', url))
  mainWindow.webContents.on('did-fail-load', (_e, code, desc, url) => console.log('[did-fail-load]', code, desc, url))
  mainWindow.webContents.on('did-finish-load', () => console.log('[did-finish-load]', mainWindow.webContents.getURL()))

  console.log('[main] isPackaged=', app.isPackaged, 'WEB_INDEX=', WEB_INDEX, 'PROD_URL=', PROD_URL)
  if (app.isPackaged) {
    // 生产：直接加载线上前端（http 同源，避免 file:// 下 module script 被 Chromium CORS 阻止导致白屏；
    // 同时 API 请求同源，无跨域问题；更新机制读取的是服务器端新版本信息）
    mainWindow.loadURL(PROD_URL)
  } else {
    mainWindow.loadURL(DEV_URL)
  }

  // 主窗口关闭即退出应用
  mainWindow.on('closed', () => {
    mainWindow = null
    app.quit()
  })
}

// 兜底：所有窗口关闭时退出（Windows 下主窗口 closed 已直接 app.quit）
app.on('window-all-closed', () => {
  app.quit()
})
