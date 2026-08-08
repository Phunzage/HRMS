import client from './client'
import type { PageResult } from './client'

// 公告
export interface Announcement {
  id: number
  title: string
  content: string
  publisher_id: number
  publisher_name: string
  is_pinned: boolean
  status: string // published / draft
  published_at: string | null
  created_at: string
  updated_at: string
}

// 公告分页列表
export function fetchAnnouncements(
  params: {
    keyword?: string
    status?: string
    page?: number
    page_size?: number
  } = {},
) {
  return client.get('/announcements', { params }) as Promise<PageResult<Announcement>>
}

// 发布公告
export function createAnnouncement(data: { title: string; content: string; is_pinned?: boolean }) {
  return client.post('/announcements', data) as Promise<Announcement>
}

// 编辑公告
export function updateAnnouncement(id: number, data: Partial<Announcement>) {
  return client.put(`/announcements/${id}`, data) as Promise<Announcement>
}

// 删除公告（软删除）
export function deleteAnnouncement(id: number) {
  return client.delete(`/announcements/${id}`) as Promise<null>
}
