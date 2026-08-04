interface PaginationProps {
  page: number
  pageSize: number
  total: number
  onChange: (page: number) => void
}

// 通用分页条：上一页/下一页 + 当前页 + 总数
export default function Pagination({ page, pageSize, total, onChange }: PaginationProps) {
  const totalPages = Math.max(Math.ceil(total / pageSize), 1)
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-base-content/60">共 {total} 条</span>
      <div className="join">
        <button
          type="button"
          className="btn btn-sm join-item"
          disabled={page <= 1}
          onClick={() => onChange(page - 1)}
        >
          上一页
        </button>
        <button type="button" className="btn btn-sm join-item" disabled>
          {page} / {totalPages}
        </button>
        <button
          type="button"
          className="btn btn-sm join-item"
          disabled={page >= totalPages}
          onClick={() => onChange(page + 1)}
        >
          下一页
        </button>
      </div>
    </div>
  )
}
