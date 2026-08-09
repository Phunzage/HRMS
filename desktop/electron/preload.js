// 预加载脚本：通过 contextBridge 安全地向渲染进程暴露更新相关 API
// 渲染进程只能调用这些白名单方法，无法直接访问 Node/Electron 能力
const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('hrmsUpdater', {
  // 获取当前更新状态（state / 服务器最新版本 / 本机版本）
  getUpdateInfo: () => ipcRenderer.invoke('update:get-info'),
  // 手动触发一次更新检查（菜单「检查更新」调用）
  checkNow: () => ipcRenderer.invoke('update:check-now'),
  // 下载更新包并打开所在目录提示手动安装
  downloadUpdate: () => ipcRenderer.invoke('update:download'),
  // 订阅更新状态事件，返回取消订阅函数
  onUpdateStatus: (cb) => {
    const listener = (_event, data) => cb(data)
    ipcRenderer.on('update:status', listener)
    return () => ipcRenderer.removeListener('update:status', listener)
  },
})
