// 通用格式化工具

// 日期 → YYYY-MM-DD（后端 date 字段可能带时分秒，取前 10 位）
export function formatDate(v?: string | null): string {
  if (!v) return '-'
  const d = v.slice(0, 10)
  return d || '-'
}

// 日期时间 → YYYY-MM-DD HH:mm（本地时区显示）
export function formatDateTime(v?: string | null): string {
  if (!v) return '-'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return v
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 金额千分位（保留两位小数）
export function formatMoney(v: number | string): string {
  const n = typeof v === 'string' ? Number(v) : v
  if (Number.isNaN(n)) return '-'
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

// 天数：整数不带小数，非整数保留一位
export function formatDays(v: number): string {
  if (Number.isNaN(v)) return '-'
  if (v === Math.floor(v)) return String(v)
  return v.toFixed(1)
}

// datetime-local 输入值（YYYY-MM-DDTHH:mm）→ 后端格式（YYYY-MM-DD HH:mm）
export function toBackendDateTime(v?: string): string | undefined {
  if (!v) return undefined
  return v.replace('T', ' ')
}

// 后端时间字符串 → datetime-local 输入值（YYYY-MM-DDTHH:mm）
export function toDatetimeLocal(v?: string | null): string {
  if (!v) return ''
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 当前月份 YYYY-MM（本地时区）
export function currentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}
