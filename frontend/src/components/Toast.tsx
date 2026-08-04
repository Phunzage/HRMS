import { useToastStore } from '../store/toast'

// 类型 → daisyUI alert 颜色
const typeClass: Record<string, string> = {
  success: 'alert-success',
  error: 'alert-error',
  info: 'alert-info',
}

// 全局 toast 容器：渲染在主布局中，右上角浮层
export default function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts)
  const remove = useToastStore((s) => s.remove)

  return (
    <div className="toast toast-top toast-end z-50">
      {toasts.map((t) => (
        <div key={t.id} className={`alert ${typeClass[t.type]} py-2 text-sm shadow-lg`}>
          <span>{t.message}</span>
          <button type="button" className="btn btn-ghost btn-xs" onClick={() => remove(t.id)}>
            ✕
          </button>
        </div>
      ))}
    </div>
  )
}
