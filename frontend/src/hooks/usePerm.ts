import { useAuthStore } from '../store/auth'

// 权限点缺失（如旧 localStorage 无 permissions 字段）时的稳定空引用：
// 避免 zustand useSyncExternalStore 因每次返回新数组引用（[] !== []）误判 store 变更而无限重渲染
const EMPTY_PERMS: string[] = []

// usePerm 权限点判断 Hook：读取当前登录用户全部权限点 code，供页面 UI 权限联动。
// 有权限点的操作（写按钮）一律用 has(code)；无权限点概念的浏览性功能保留 roles 过滤。
export function usePerm() {
  const perms = useAuthStore((s) => s.user?.permissions ?? EMPTY_PERMS)
  return {
    /** 是否拥有指定权限点 */
    has: (code: string) => perms.includes(code),
    /** 是否拥有任意一个权限点 */
    hasAny: (codes: string[]) => codes.some((c) => perms.includes(c)),
  }
}
