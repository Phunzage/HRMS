import { useCallback, useRef } from 'react'
import { useAuthStore } from '../store/auth'
import { fetchRecentNotifications } from '../api/notification'

// 系统通知增量检测 Hook（桌面端 Electron 专用）：
// 轮询时对比「最近一条未读通知 id」与 localStorage 记录的 last_id，出现新 id 则弹 Windows 系统通知。
// 网页版浏览器无 window.hrmsUpdater，check() 直接返回，保持应用内铃铛行为，符合预期。
// localStorage key 带 username 后缀，多账号切换互不串号。
const STORAGE_PREFIX = 'hrms_last_notify_id'

export function useSystemNotify() {
  const username = useAuthStore((s) => s.user?.username)
  // 记录上次轮询时的用户名；登出置 null，用于识别「登录/切换账号」首次轮询：只记录不弹
  const lastUserRef = useRef<string | null>(null)

  const key = username ? `${STORAGE_PREFIX}_${username}` : STORAGE_PREFIX

  /** 轮询一次增量检测：有新通知时弹系统通知并记录最新 id */
  const check = useCallback(async () => {
    // 仅 Electron 桌面端可弹系统通知；纯浏览器跳过（不发起请求也不弹）
    if (!window.hrmsUpdater?.notify) return
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
    window.hrmsUpdater.notify({
      title: 'HRMS 新通知',
      body: recent[0]?.title ?? '',
    })
  }, [key, username])

  return check
}
