import { useAuthStore } from '../store/auth'

// usePerm 权限点判断 Hook：读取当前登录用户全部权限点 code，供页面 UI 权限联动。
// 有权限点的操作（写按钮）一律用 has(code)；无权限点概念的浏览性功能保留 roles 过滤。
export function usePerm() {
  const perms = useAuthStore((s) => s.user?.permissions ?? [])
  return {
    /** 是否拥有指定权限点 */
    has: (code: string) => perms.includes(code),
    /** 是否拥有任意一个权限点 */
    hasAny: (codes: string[]) => codes.some((c) => perms.includes(c)),
  }
}
