// 系统通知模块：注册 notify IPC，供渲染进程弹出 Windows 通知中心消息。
// 抽成独立模块便于主进程复用与单测打桩：ipcMain 与 Notification 由调用方注入，
// 测试时可注入 mock 断言「ipcMain.on('notify') 注册 + new Notification().show() 触发」。

const DEFAULT_TITLE = 'HRMS 新通知'

/**
 * 注册 notify IPC 处理器
 * @param {typeof import('electron').IpcMain} ipcMain 主进程 IPC 对象
 * @param {typeof import('electron').Notification} Notification 主进程原生通知类
 */
function registerNotifyIpc(ipcMain, Notification) {
  ipcMain.on('notify', (_event, payload = {}) => {
    if (!Notification.isSupported()) {
      console.warn('[notify] 当前平台不支持系统通知，已跳过')
      return
    }
    // 标题/内容截断到系统通知合理长度，非法 payload 兜底默认标题
    const title = String(payload.title || DEFAULT_TITLE).slice(0, 128)
    const body = String(payload.body || '').slice(0, 500)
    new Notification({ title, body, silent: false }).show()
  })
}

module.exports = { registerNotifyIpc }
