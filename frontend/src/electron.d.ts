// Electron 预加载脚本（desktop/electron/preload.js）经 contextBridge 暴露到渲染进程的全局对象类型声明。
// 仅桌面端（Electron 环境）存在；纯浏览器网页版中 window.hrmsUpdater 为 undefined。
interface HrmsUpdaterApi {
  /** 弹出系统通知（Windows 通知中心），主进程新建原生 Notification */
  notify: (payload: { title: string; body?: string }) => void
  /** 获取当前更新状态（state / 服务器最新版本 / 本机版本） */
  getUpdateInfo: () => Promise<unknown>
  /** 手动触发一次更新检查 */
  checkNow: () => Promise<unknown>
  /** 下载更新包并打开所在目录提示手动安装 */
  downloadUpdate: () => Promise<unknown>
  /** 订阅更新状态事件，返回取消订阅函数 */
  onUpdateStatus: (cb: (data: unknown) => void) => () => void
}

interface Window {
  hrmsUpdater?: HrmsUpdaterApi
}
