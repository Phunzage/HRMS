import { useEffect, useRef, useState } from 'react'
import { Plus, Trash2, Workflow } from 'lucide-react'
import {
  listApprovalFlows,
  updateApprovalFlow,
  NEW_APPROVER_TYPES,
  APPROVAL_PERMISSION_LABELS,
  APPROVER_TYPE_LABEL,
  type ApprovalFlowItem,
  type ApprovalPermission,
  type ApprovalStepItem,
} from '../api/approvalFlow'
import { fetchAllPositions, type Position } from '../api/employee'
import { listRoles, type RoleItem } from '../api/role'
import { usePerm } from '../hooks/usePerm'
import { toast } from '../store/toast'
import Modal from '../components/Modal'

// 审批人类型 → 展示 badge 配色（部门经理=info / 指定角色=secondary / 按权限=primary / HR终审=primary / 指定岗位=warning）
const TYPE_BADGE: Record<ApprovalStepItem['approver_type'], string> = {
  manager: 'badge-info',
  role: 'badge-secondary',
  permission: 'badge-primary',
  hr: 'badge-primary',
  position: 'badge-warning',
}

// 编辑 Modal 内的步骤行（key 用于增删时稳定 React key，nameEdited 标记名称是否被用户手动改过）
interface StepDraft extends ApprovalStepItem {
  key: number
  nameEdited: boolean
}

// 生成某类型的默认步骤名（类型选择后自动填充，用户可改；role 由所选角色决定，permission 由权限点决定，另行处理）
function defaultName(type: ApprovalStepItem['approver_type'], positionName = '') {
  switch (type) {
    case 'manager':
      return '部门经理审批'
    case 'hr':
      return 'HR 终审'
    case 'position':
      return positionName ? `${positionName}审批` : ''
    case 'role':
    case 'permission':
      return ''
  }
}

// 当前流程的建议审批权限点（请假流程→请假审批，补卡流程→补卡审批）
function defaultPermission(flowCode?: string) {
  return flowCode === 'attendance_correction' ? 'attendance_correction:approve' : 'leave:approve'
}

export default function ApprovalFlowsPage() {
  // 审批流程配置为写操作页面，须拥有 approval_flow:write（admin 种子已含该权限点）
  const { has } = usePerm()
  const canConfigure = has('approval_flow:write')

  const [flows, setFlows] = useState<ApprovalFlowItem[]>([])
  const [approvalPerms, setApprovalPerms] = useState<ApprovalPermission[]>([])
  const [positions, setPositions] = useState<Position[]>([])
  const [roles, setRoles] = useState<RoleItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  // 编辑弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editingFlow, setEditingFlow] = useState<ApprovalFlowItem | null>(null)
  const [draftSteps, setDraftSteps] = useState<StepDraft[]>([])
  const [saving, setSaving] = useState(false)
  const keyRef = useRef(0)

  // 权限点 code → 名称（优先后端 approval_permissions，缺省走本地常量）
  const permissionName = (code: string) =>
    approvalPerms.find((p) => p.code === code)?.name ?? APPROVAL_PERMISSION_LABELS[code] ?? code

  // 权限点 code → 拥有该权限的角色列表
  const permissionRoles = (code: string) => approvalPerms.find((p) => p.code === code)?.roles ?? []

  // 岗位列表 + 角色列表（编辑 Modal 岗位/角色下拉用，仅拉一次）
  useEffect(() => {
    let cancelled = false
    fetchAllPositions()
      .then((res) => {
        if (!cancelled) setPositions(res.list ?? [])
      })
      .catch(() => undefined)
    listRoles()
      .then((res) => {
        if (!cancelled) setRoles(Array.isArray(res) ? res : [])
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [])

  // 审批流程列表 + 审批权限点
  useEffect(() => {
    if (!canConfigure) return
    let cancelled = false
    setLoading(true)
    setError('')
    listApprovalFlows()
      .then((res) => {
        if (!cancelled) {
          setFlows(res.list ?? [])
          setApprovalPerms(res.approval_permissions ?? [])
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取审批流程失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [reloadKey, canConfigure])

  // 岗位 ID → 岗位名（卡片展示指定岗位步骤时用）
  const positionName = (ref: number) => positions.find((p) => p.id === ref)?.name

  // 角色 ID → 角色名（卡片展示指定角色步骤、编辑 Modal 自动命名时用）
  const roleName = (id: number) => roles.find((r) => r.id === id)?.name

  const openEdit = (flow: ApprovalFlowItem) => {
    setEditingFlow(flow)
    setDraftSteps(
      flow.steps.map((s) => ({ ...s, key: ++keyRef.current, nameEdited: false })),
    )
    setModalOpen(true)
  }

  const addStep = () => {
    if (draftSteps.length >= 5) return // 后端限制 1~5 级
    setDraftSteps((steps) => [
      ...steps,
      { key: ++keyRef.current, step_order: steps.length + 1, approver_type: 'manager', approver_ref: 0, approver_permission: '', approver_role_id: 0, name: '部门经理审批', nameEdited: false },
    ])
  }

  const removeStep = (key: number) => {
    setDraftSteps((steps) =>
      steps
        .filter((s) => s.key !== key)
        .map((s, i) => ({ ...s, step_order: i + 1 })),
    )
  }

  const updateStep = (key: number, patch: Partial<StepDraft>) => {
    setDraftSteps((steps) => steps.map((s) => (s.key === key ? { ...s, ...patch } : s)))
  }

  // 切换审批人类型：permission 自动建议当前流程对应的权限点并填充步骤名；
  // role 清空角色引用（选角色后自动填充步骤名）；非 permission/role 清空对应引用
  const handleTypeChange = (key: number, type: StepDraft['approver_type']) => {
    setDraftSteps((steps) =>
      steps.map((s) => {
        if (s.key !== key) return s
        const position = positions.find((p) => p.id === s.approver_ref)
        if (type === 'permission') {
          const perm = s.approver_permission || defaultPermission(editingFlow?.code)
          return {
            ...s,
            approver_type: type,
            approver_ref: 0,
            approver_role_id: 0,
            approver_permission: perm,
            name: s.nameEdited ? s.name : permissionName(perm) || '权限审批',
            nameEdited: false,
          }
        }
        if (type === 'role') {
          const currentRole = roles.find((r) => r.id === s.approver_role_id)
          return {
            ...s,
            approver_type: type,
            approver_ref: 0,
            approver_role_id: s.approver_role_id || 0,
            approver_permission: '',
            // 已选过角色时按「{角色名}审批」自动填充，否则留空待选角色后填充
            name: s.nameEdited ? s.name : currentRole ? `${currentRole.name}审批` : '',
            nameEdited: false,
          }
        }
        return {
          ...s,
          approver_type: type,
          approver_ref: type === 'position' ? s.approver_ref : 0,
          approver_role_id: 0,
          approver_permission: '',
          name: defaultName(type, position?.name),
          nameEdited: false,
        }
      }),
    )
  }

  // 选择审批角色：未手动改过名称时自动填充「{角色名}审批」
  const handleRoleChange = (key: number, roleId: number) => {
    setDraftSteps((steps) =>
      steps.map((s) => {
        if (s.key !== key) return s
        const role = roles.find((r) => r.id === roleId)
        const name = role && !s.nameEdited ? `${role.name}审批` : s.name
        return { ...s, approver_role_id: roleId, name }
      }),
    )
  }

  // 选择审批权限点：未手动改过名称时自动填充「{权限名}」
  const handlePermissionChange = (key: number, code: string) => {
    setDraftSteps((steps) =>
      steps.map((s) => {
        if (s.key !== key) return s
        return {
          ...s,
          approver_permission: code,
          name: s.nameEdited ? s.name : permissionName(code) || '权限审批',
        }
      }),
    )
  }

  // 选择岗位：未手动改过名称时自动填充「{岗位名}审批」
  const handlePositionChange = (key: number, positionId: number) => {
    setDraftSteps((steps) =>
      steps.map((s) => {
        if (s.key !== key) return s
        const position = positions.find((p) => p.id === positionId)
        const name = position && !s.nameEdited ? `${position.name}审批` : s.name
        return { ...s, approver_ref: positionId, name }
      }),
    )
  }

  const handleSave = async () => {
    if (!editingFlow) return
    // 前端校验：步骤名非空；指定角色必须已选角色；指定岗位必须已选岗位；按权限审批必须已选权限点
    for (const s of draftSteps) {
      if (!s.name.trim()) {
        toast.error(`第 ${s.step_order} 级步骤名称不能为空`)
        return
      }
      if (s.approver_type === 'role' && !s.approver_role_id) {
        toast.error(`第 ${s.step_order} 级（指定角色）请选择角色`)
        return
      }
      if (s.approver_type === 'position' && !s.approver_ref) {
        toast.error(`第 ${s.step_order} 级（指定岗位）请选择岗位`)
        return
      }
      if (s.approver_type === 'permission' && !s.approver_permission) {
        toast.error(`第 ${s.step_order} 级（按权限审批）请选择审批权限`)
        return
      }
    }
    setSaving(true)
    try {
      await updateApprovalFlow(editingFlow.code, {
        name: editingFlow.name,
        steps: draftSteps.map(({ step_order, approver_type, approver_ref, approver_permission, approver_role_id, name }) => ({
          step_order,
          approver_type,
          approver_ref,
          approver_permission,
          approver_role_id,
          name,
        })),
      })
      toast.success('审批流程已保存')
      setModalOpen(false)
      setReloadKey((k) => k + 1)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  // 无审批流程配置权限时显示无权限提示
  if (!canConfigure) {
    return (
      <div className="flex justify-center py-24">
        <div className="alert alert-error max-w-lg">
          <span>无权限访问该页面，需要「审批流程配置」权限。</span>
        </div>
      </div>
    )
  }

  // 步骤卡片：审批人类型 badge 文案（role 显示「指定角色·角色名」、permission 显示「按权限·权限名」）
  const stepTypeLabel = (s: ApprovalStepItem) =>
    s.approver_type === 'role'
      ? `指定角色·${s.approver_role_name || roleName(s.approver_role_id) || '未命名'}`
      : s.approver_type === 'permission'
        ? `按权限·${permissionName(s.approver_permission)}`
        : APPROVER_TYPE_LABEL[s.approver_type]

  // permission 步骤可审批角色文案（如「人事、经理 可审批」）；role 步骤直接展示角色名
  const stepRolesText = (s: ApprovalStepItem) => {
    if (s.approver_type === 'permission') {
      return permissionRoles(s.approver_permission)
        .map((r) => r.name)
        .join('、')
    }
    if (s.approver_type === 'role') {
      return s.approver_role_name || roleName(s.approver_role_id) || ''
    }
    return ''
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-xl font-bold">审批流程配置</h2>
        <div className="alert alert-info shadow-sm">
          <span>配置请假/补卡的审批级别与每级审批人，修改仅影响新申请，已进行中的审批不受影响。审批优先级=步骤顺序：第 1 级先审，通过后进入第 2 级…</span>
        </div>
      </div>

      {error ? (
        <div className="alert alert-error">
          <span>{error}</span>
        </div>
      ) : loading ? (
        <div className="flex items-center justify-center py-24">
          <span className="loading loading-spinner loading-lg text-primary" />
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {flows.map((flow) => (
            <div key={flow.code} className="card bg-base-100 shadow-sm">
              <div className="card-body p-4">
                <div className="flex items-center gap-2">
                  <h3 className="text-base font-semibold">{flow.name}</h3>
                  {flow.status === 'active' ? (
                    <span className="badge badge-success badge-sm">启用</span>
                  ) : (
                    <span className="badge badge-ghost badge-sm">停用</span>
                  )}
                  <button
                    type="button"
                    className="btn btn-ghost btn-sm ml-auto"
                    onClick={() => openEdit(flow)}
                  >
                    <Workflow className="h-4 w-4" />
                    编辑配置
                  </button>
                </div>

                {flow.steps.length > 0 ? (
                  <ul className="flex flex-col gap-2">
                    {flow.steps.map((s) => (
                      <li
                        key={s.id ?? `${flow.code}-${s.step_order}`}
                        className="rounded-box border border-base-200 px-3 py-2"
                      >
                        <div className="flex items-center gap-2">
                          <span className="badge badge-ghost badge-sm">{s.step_order}</span>
                          <span className="text-sm font-medium">{s.name}</span>
                          <span className={`badge badge-sm ${TYPE_BADGE[s.approver_type]}`}>
                            {stepTypeLabel(s)}
                          </span>
                          {s.approver_type === 'position' && positionName(s.approver_ref) && (
                            <span className="text-xs text-base-content/50">
                              （{positionName(s.approver_ref)}）
                            </span>
                          )}
                        </div>
                        {s.approver_type === 'role' && (
                          <div className="mt-1 text-xs text-base-content/50">
                            {stepRolesText(s) ? `${stepRolesText(s)} 可审批` : '未选择角色'}
                          </div>
                        )}
                        {s.approver_type === 'permission' && (
                          <div className="mt-1 text-xs text-base-content/50">
                            {stepRolesText(s) ? `${stepRolesText(s)} 可审批` : '暂无角色拥有该权限'}
                          </div>
                        )}
                      </li>
                    ))}
                  </ul>
                ) : (
                  <div className="py-4 text-center text-sm text-base-content/50">未配置审批步骤</div>
                )}

                <div className="text-xs text-base-content/50">共 {flow.steps.length} 级审批</div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 编辑流程 Modal */}
      <Modal
        open={modalOpen}
        title={editingFlow ? `编辑流程 - ${editingFlow.name}` : '编辑流程'}
        onClose={() => setModalOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setModalOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={handleSave}>
              {saving && <span className="loading loading-spinner loading-sm" />}
              保存
            </button>
          </>
        }
      >
        <div className="flex flex-col gap-3">
          {/* 列头说明 */}
          <div className="grid grid-cols-[2.5rem_1fr_9rem_2.5rem] items-center gap-2 px-1 text-xs text-base-content/50">
            <span>序号</span>
            <span>步骤名称</span>
            <span>审批人</span>
            <span className="text-right">操作</span>
          </div>

          {draftSteps.map((s) => (
            <div key={s.key} className="rounded-box border border-base-300 p-2">
              <div className="grid grid-cols-[2.5rem_1fr_9rem_2.5rem] items-center gap-2">
                <span className="badge badge-ghost justify-center">{s.step_order}</span>
                <input
                  type="text"
                  className="input input-bordered input-sm w-full"
                  placeholder="步骤名称"
                  value={s.name}
                  onChange={(e) => updateStep(s.key, { name: e.target.value, nameEdited: true })}
                />
                <select
                  className="select select-bordered select-sm w-full"
                  value={s.approver_type}
                  onChange={(e) => handleTypeChange(s.key, e.target.value as StepDraft['approver_type'])}
                >
                  {/* 新配置仅 manager/role；旧 hr/position/permission 步骤保留当前选项以便展示 */}
                  {(s.approver_type === 'hr' || s.approver_type === 'position' || s.approver_type === 'permission'
                    ? [...NEW_APPROVER_TYPES, s.approver_type]
                    : NEW_APPROVER_TYPES
                  ).map((t) => (
                    <option key={t} value={t}>
                      {APPROVER_TYPE_LABEL[t]}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  className="btn btn-ghost btn-sm btn-circle justify-self-end text-error"
                  disabled={draftSteps.length <= 1}
                  onClick={() => removeStep(s.key)}
                  title="删除本级"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>

              {/* 按权限审批：权限点下拉 + 可审批角色 badge */}
              {s.approver_type === 'permission' && (
                <div className="mt-2 flex flex-col gap-1.5">
                  <select
                    className="select select-bordered select-sm w-full"
                    value={s.approver_permission}
                    onChange={(e) => handlePermissionChange(s.key, e.target.value)}
                  >
                    <option value="">请选择审批权限</option>
                    {approvalPerms.map((p) => (
                      <option key={p.code} value={p.code}>
                        {p.name}
                      </option>
                    ))}
                  </select>
                  <div className="flex flex-wrap items-center gap-1 text-xs text-base-content/60">
                    <span>可审批角色：</span>
                    {permissionRoles(s.approver_permission).length > 0 ? (
                      permissionRoles(s.approver_permission).map((r) => (
                        <span key={r.id} className="badge badge-sm badge-ghost">
                          {r.name}
                        </span>
                      ))
                    ) : (
                      <span className="text-base-content/40">暂无角色拥有该权限</span>
                    )}
                  </div>
                </div>
              )}

              {/* 指定角色：角色下拉（全部角色，含部门经理/人事专员/副总经理等） */}
              {s.approver_type === 'role' && (
                <select
                  className="select select-bordered select-sm mt-2 w-full"
                  value={s.approver_role_id}
                  onChange={(e) => handleRoleChange(s.key, Number(e.target.value))}
                >
                  <option value={0}>请选择角色</option>
                  {roles.map((r) => (
                    <option key={r.id} value={r.id}>
                      {r.name}
                    </option>
                  ))}
                </select>
              )}

              {/* 仅指定岗位时显示岗位下拉 */}
              {s.approver_type === 'position' && (
                <select
                  className="select select-bordered select-sm mt-2 w-full"
                  value={s.approver_ref}
                  onChange={(e) => handlePositionChange(s.key, Number(e.target.value))}
                >
                  <option value={0}>请选择岗位</option>
                  {positions.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
              )}

              {/* 旧类型兼容提示 */}
              {(s.approver_type === 'hr' || s.approver_type === 'position' || s.approver_type === 'permission') && (
                <div className="mt-1 text-xs text-warning">旧类型，建议迁移为「指定角色」或「按权限审批」</div>
              )}
            </div>
          ))}

          <button
            type="button"
            className="btn btn-outline btn-sm"
            disabled={draftSteps.length >= 5}
            onClick={addStep}
          >
            <Plus className="h-4 w-4" />
            添加一级
          </button>
        </div>
      </Modal>
    </div>
  )
}
