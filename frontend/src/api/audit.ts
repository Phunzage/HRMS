import client from './client'
import type { PageResult } from './client'

// 审计日志项
export interface AuditLog {
  id: number
  user_id: number
  username: string
  module: string
  action: string
  resource_id: string
  detail: string
  ip: string
  created_at: string
}

// 审计日志分页列表，支持 module / action / username / 日期范围过滤
export function listAuditLogs(
  params: {
    module?: string
    action?: string
    username?: string
    start_date?: string
    end_date?: string
    page?: number
    page_size?: number
  } = {},
) {
  return client.get('/audit-logs', { params }) as Promise<PageResult<AuditLog>>
}
