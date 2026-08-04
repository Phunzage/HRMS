import { create } from 'zustand'

// 轻量 toast 状态：全局单例，组件内用 toast.success() / toast.error() 触发
export interface ToastItem {
  id: number
  message: string
  type: 'success' | 'error' | 'info'
}

let nextId = 1

interface ToastState {
  toasts: ToastItem[]
  /** 追加一条 toast，3 秒后自动消失 */
  push: (message: string, type?: ToastItem['type']) => void
  remove: (id: number) => void
}

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],
  push: (message, type = 'success') => {
    const id = nextId++
    set((s) => ({ toasts: [...s.toasts, { id, message, type }] }))
    setTimeout(() => {
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }))
    }, 3000)
  },
  remove: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}))

// 便捷调用：toast.success('保存成功') / toast.error('请求失败')
export const toast = {
  success: (msg: string) => useToastStore.getState().push(msg, 'success'),
  error: (msg: string) => useToastStore.getState().push(msg, 'error'),
  info: (msg: string) => useToastStore.getState().push(msg, 'info'),
}
