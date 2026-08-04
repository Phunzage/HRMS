import type { BadgeItem } from '../utils/status'

interface StatusBadgeProps {
  map: Record<string, BadgeItem>
  value: string
}

// 通用状态徽章：按状态映射渲染 daisyUI badge，未知状态回退 ghost
export default function StatusBadge({ map, value }: StatusBadgeProps) {
  const item = map[value]
  return <span className={`badge badge-sm ${item?.badge ?? 'badge-ghost'}`}>{item?.label ?? value}</span>
}
