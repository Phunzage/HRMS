import { type ReactNode } from 'react'
import { X } from 'lucide-react'

interface ModalProps {
  open: boolean
  title: string
  children: ReactNode
  /** 底部操作区（通常是按钮组） */
  footer?: ReactNode
  onClose: () => void
}

// 通用 Modal：基于 daisyUI modal，open 控制显示，关闭按钮 + 点击遮罩关闭
export default function Modal({ open, title, children, footer, onClose }: ModalProps) {
  if (!open) return null
  return (
    <dialog className="modal modal-open" onClose={onClose}>
      <div className="modal-box max-h-[85vh] w-full max-w-2xl">
        <div className="flex items-center justify-between border-b border-base-300 pb-3">
          <h3 className="text-lg font-bold">{title}</h3>
          <button type="button" className="btn btn-sm btn-circle btn-ghost" onClick={onClose}>
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="max-h-[60vh] overflow-y-auto py-4">{children}</div>
        {footer && <div className="modal-action">{footer}</div>}
      </div>
      {/* 点击遮罩关闭 */}
      <form method="dialog" className="modal-backdrop">
        <button type="button" onClick={onClose}>
          关闭
        </button>
      </form>
    </dialog>
  )
}
