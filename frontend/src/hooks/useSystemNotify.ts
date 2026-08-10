import { useCallback, useRef } from 'react'
import { Capacitor } from '@capacitor/core'
import { LocalNotifications } from '@capacitor/local-notifications'
import { useAuthStore } from '../store/auth'
import { fetchRecentNotifications } from '../api/notification'

// 系统通知增量检测 Hook（桌面端 Electron + Android 原生）：
// 轮询时对比「最近一条未读通知 id」与 localStorage 记录的 last_id，出现新 id 则弹系统通知。
// - 桌面端（Electron）：window.hrmsUpdater.notify 弹 Windows 通知中心；
// - Android 原生（Capacitor）：LocalNotifications.schedule 弹系统通知栏（同微信行为）；
// - 纯浏览器两者都没有，check() 直接返回，保持应用内铃铛行为，符合预期。
// localStorage key 带 username 后缀，多账号切换互不串号。
const STORAGE_PREFIX = 'hrms_last_notify_id'

// Android 13+ 动态通知权限：会话内只请求一次，避免每 30s 轮询反复弹权限框
let permissionRequested = false

export function useSystemNotify() {
  const username = useAuthStore((s) => s.user?.username)
  // 记录上次轮询时的用户名；登出置 null，用于识别「登录/切换账号」首次轮询：只记录不弹
  const lastUserRef = useRef<string | null>(null)

  const key = username ? `${STORAGE_PREFIX}_${username}` : STORAGE_PREFIX

  /** 轮询一次增量检测：有新通知时弹系统通知并记录最新 id */
  const check = useCallback(async () => {
    // Android 原生平台走 Capacitor；否则仅 Electron 桌面端可弹系统通知（纯浏览器跳过，不发起请求也不弹）
    const isNative = Capacitor.isNativePlatform()
    if (!isNative && !window.hrmsUpdater?.notify) return
    // 未登录不弹（登出时记下状态，下次登录按新会话处理）
    if (!username) {
      lastUserRef.current = null
      return
    }

    let recent: { id: number; title: string }[] = []
    try {
      recent = await fetchRecentNotifications(1)
    } catch {
      // 轮询失败静默，不影响角标主流程
      return
    }
    const latestId = recent[0]?.id
    if (!latestId) return // 无未读通知，无增量可弹

    // 登录/切换账号后首次轮询：只把当前最新 id 记为基线，不弹历史通知
    const isFreshLogin = lastUserRef.current !== username
    lastUserRef.current = username
    if (isFreshLogin) {
      localStorage.setItem(key, String(latestId))
      return
    }

    const lastId = Number(localStorage.getItem(key) || 0)
    if (latestId === lastId) return // 无新通知，避免重复弹

    localStorage.setItem(key, String(latestId))

    if (isNative) {
      // Android 13+ 需动态申请 POST_NOTIFICATIONS，首次弹窗前请求一次
      if (!permissionRequested) {
        permissionRequested = true
        const perm = await LocalNotifications.requestPermissions()
        if (perm.display !== 'granted') return // 未授权不弹（用户可在系统设置开启）
      }
      await LocalNotifications.schedule({
        notifications: [
          {
            id: latestId,
            title: 'HRMS 新通知',
            body: recent[0]?.title ?? '',
            smallIcon: 'ic_launcher',
          },
        ],
      })
    } else {
      window.hrmsUpdater?.notify({
        title: 'HRMS 新通知',
        body: recent[0]?.title ?? '',
      })
    }
  }, [key, username])

  return check
}
