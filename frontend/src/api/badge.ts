import client from './client'

// 导航栏角标汇总（GET /badges，当前用户）
export interface Badges {
  pending_approvals: number // 待我审批任务数
  unread_announcements: number // 未读公告数
  unread_notifications: number // 未读站内通知数
}

// 获取当前用户导航栏角标汇总（侧边栏菜单项 + 顶栏铃铛）
export function fetchBadges() {
  return client.get('/badges') as Promise<Badges>
}
