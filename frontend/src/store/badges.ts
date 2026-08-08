import { create } from 'zustand'

// 导航栏角标字段（与 GET /badges 返回一致）
interface BadgeFields {
  pending_approvals: number // 待我审批任务数
  unread_announcements: number // 未读公告数
  unread_notifications: number // 未读站内通知数
}

interface BadgesState extends BadgeFields {
  lastUpdated: number | null // 最近一次轮询成功时间戳
  /** 覆盖式更新角标（轮询/页面联动后整包替换） */
  setBadges: (badges: Partial<BadgeFields>) => void
  /** 审批角标减一（单条标记已读/处理完） */
  decrementApproval: () => void
  /** 公告角标减一（单条公告已读） */
  decrementAnnouncement: () => void
  /** 通知角标减一（铃铛单条已读） */
  decrementNotification: () => void
}

// 全局导航栏角标状态（不持久化：每次登录后由轮询即时拉取）
export const useBadgeStore = create<BadgesState>((set) => ({
  pending_approvals: 0,
  unread_announcements: 0,
  unread_notifications: 0,
  lastUpdated: null,
  setBadges: (badges) => set((s) => ({ ...s, ...badges, lastUpdated: Date.now() })),
  decrementApproval: () => set((s) => ({ pending_approvals: Math.max(0, s.pending_approvals - 1) })),
  decrementAnnouncement: () => set((s) => ({ unread_announcements: Math.max(0, s.unread_announcements - 1) })),
  decrementNotification: () => set((s) => ({ unread_notifications: Math.max(0, s.unread_notifications - 1) })),
}))
