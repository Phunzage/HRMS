import { Capacitor, registerPlugin } from '@capacitor/core'

// TokenStore 桥接：登录 token 存于 WebView localStorage，原生层（Android 后台 Service）读不到，
// 经 Capacitor 插件写入 SharedPreferences，供 NotifyWorker 轮询时携带 Authorization。
// 非原生平台（纯浏览器 / Electron）注册 web 端空实现，调用静默成功，不影响网页版行为。

interface TokenStorePlugin {
  /** 写入 token（登录成功 / 刷新令牌后） */
  set(options: { token: string }): Promise<void>
  /** 读取 token（原生层用，前端一般不直接调） */
  get(): Promise<{ token: string }>
  /** 清空 token（退出登录） */
  clear(): Promise<void>
}

// web 端空实现：浏览器/桌面端无原生 TokenStore 插件，所有调用 no-op
const TokenStore = registerPlugin<TokenStorePlugin>('TokenStore', {
  web: () => ({
    async set() {
      /* 非原生平台无 SharedPreferences，忽略 */
    },
    async get() {
      return { token: '' }
    },
    async clear() {
      /* 忽略 */
    },
  }),
})

/** 将 token 变更同步到原生层：有 token 写入，无 token（登出）清空 */
export function syncTokenToNative(token: string | null): void {
  if (!Capacitor.isNativePlatform()) return
  if (token) {
    TokenStore.set({ token }).catch(() => undefined)
  } else {
    TokenStore.clear().catch(() => undefined)
  }
}

export default TokenStore
