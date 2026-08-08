// 纯 CSS 柱状图：div 高度百分比，用于仪表盘趋势展示

export interface BarItem {
  label: string
  value: number
}

interface BarChartProps {
  items: BarItem[]
  /** tailwind 背景色类，如 bg-primary */
  color: string
  /** 数值展示格式化 */
  formatValue?: (v: number) => string
  /** 柱状图高度（px） */
  height?: number
}

export default function BarChart({ items, color, formatValue, height = 28 }: BarChartProps) {
  // 兜底：items 为 null/undefined 时按空数组渲染
  const data = items ?? []
  const max = Math.max(...data.map((i) => i.value), 1)
  return (
    <div className="flex items-end gap-2">
      {data.map((it) => (
        <div key={it.label} className="flex min-w-0 flex-1 flex-col items-center gap-1">
          <div className="truncate text-xs text-base-content/70">
            {formatValue ? formatValue(it.value) : String(it.value)}
          </div>
          <div className="flex w-full items-end justify-center" style={{ height: `${height * 4}px` }}>
            <div
              className={`w-full max-w-10 rounded-t ${color}`}
              style={{ height: `${Math.max((it.value / max) * 100, 2)}%` }}
            />
          </div>
          <div className="truncate text-xs text-base-content/60">{it.label}</div>
        </div>
      ))}
    </div>
  )
}

// 金额紧凑格式化：≥1万显示「x.x万」
export function formatCompactAmount(v: number): string {
  if (v >= 10000) {
    return `${(v / 10000).toFixed(1)}万`
  }
  return String(Math.round(v))
}
