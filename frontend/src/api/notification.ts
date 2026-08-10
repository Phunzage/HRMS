import client from './client'
import type { PageResult } from './client'

// 站内通知
export interface NotificationItem {
  id: number
  user_id: number
  type: string
  title: string
  content: string
  link: string
  is_read: boolean
  created_at: string
}

// 通知分页列表
export function listNotifications(params: { is_read?: boolean; page?: number; page_size?: number } = {}) {
  return client.get('/notifications', { params }) as Promise<PageResult<NotificationItem>>
}

// 未读通知数量（顶栏铃铛）
export function fetchUnreadCount() {
  return client.get('/notifications/unread-count') as Promise<{ count: number }>
}

// 最近 N 条未读通知（按 id 倒序），供增量检测新通知（对比最新一条 id 是否变化）
export function fetchRecentNotifications(limit = 5) {
  return client.get('/notifications/recent', { params: { limit } }) as Promise<NotificationItem[]>
}

// 标记单条通知已读
export function markRead(id: number) {
  return client.put(`/notifications/${id}/read`) as Promise<null>
}

// 全部标记已读
export function markAllRead() {
  return client.put('/notifications/read-all') as Promise<null>
}
