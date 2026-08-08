import type { Department } from '../api/employee'

// 扁平化部门树节点（用于下拉选择，带层级缩进）
export interface DeptOption {
  id: number
  name: string
  level: number
}

// 递归展开部门树，level 用于缩进展示
export function flattenDepartments(tree: Department[] = [], level = 0): DeptOption[] {
  // 兜底：后端空结果返回 null 时按空数组处理
  const list = tree ?? []
  const result: DeptOption[] = []
  for (const d of list) {
    result.push({ id: d.id, name: d.name, level })
    if (d.children && d.children.length > 0) {
      result.push(...flattenDepartments(d.children, level + 1))
    }
  }
  return result
}

// 下拉展示名：按层级加全角空格缩进
export function deptOptionLabel(opt: DeptOption): string {
  return '　'.repeat(opt.level) + opt.name
}
