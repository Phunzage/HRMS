import { useEffect, useRef, useState } from 'react'
import { Plus, Trash2, Workflow } from 'lucide-react'
import {
  listApprovalFlows,
  updateApprovalFlow,
  APPROVER_TYPE_LABEL,
  type ApprovalFlowItem,
  type ApprovalStepItem,
} from '../api/approvalFlow'
import { fetchAllPositions, type Position } from '../api/employee'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import Modal from '../components/Modal'

// 审批人类型 → 展示 badge 配色（部门经理=info / HR终审=primary / 指定岗位=warning）
const TYPE_BADGE: Record<ApprovalStepItem['approver_type'], string> = {
  manager: 'badge-info',
  hr: 'badge-primary',
  position: 'badge-warning',
}

// 编辑 Modal 内的步骤行（key 用于增删时稳定 React key，nameEdited 标记名称是否被用户手动改过）
interface StepDraft extends ApprovalStepItem {
  key: number
  nameEdited: boolean
}

// 生成某类型的默认步骤名（类型选择后自动填充，用户可改）
function defaultName(type: ApprovalStepItem['approver_type'], positionName = '') {
  switch (type) {
    case 'manager':
      return '部门经理审批'
    case 'hr':
      return 'HR 终审'
    case 'position':
      return positionName ? `${positionName}审批` : ''
  }
}

export default function ApprovalFlowsPage() {
  const role = useAuthStore((s) => s.user?.role_code)
  const isAdmin = role === 'admin'

  const [flows, setFlows] = useState<ApprovalFlowItem[]>([])
  const [positions, setPositions] = useState<Position[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  // 编辑弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editingFlow, setEditingFlow] = useState<ApprovalFlowItem | null>(null)
  const [draftSteps, setDraftSteps] = useState<StepDraft[]>([])
  const [saving, setSaving] = useState(false)
  const keyRef = useRef(0)

  // 岗位列表（编辑 Modal 岗位下拉用，仅拉一次）
  useEffect(() => {
    let cancelled = false
    fetchAllPositions()
      .then((res) => {
        if (!cancelled) setPositions(res.list ?? [])
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [])

  // 审批流程列表
  useEffect(() => {
    if (!isAdmin) return
    let cancelled = false
    setLoading(true)
    setError('')
    listApprovalFlows()
      .then((res) => {
        if (!cancelled) setFlows(res ?? [])
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
  }, [reloadKey, isAdmin])

  // 岗位 ID → 岗位名（卡片展示指定岗位步骤时用）
  const positionName = (ref: number) => positions.find((p) => p.id === ref)?.name

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
      { key: ++keyRef.current, step_order: steps.length + 1, approver_type: 'manager', approver_ref: 0, name: '部门经理审批', nameEdited: false },
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

  // 切换审批人类型：自动填充默认步骤名；非指定岗位清空岗位引用
  const handleTypeChange = (key: number, type: StepDraft['approver_type']) => {
    setDraftSteps((steps) =>
      steps.map((s) => {
        if (s.key !== key) return s
        const position = positions.find((p) => p.id === s.approver_ref)
        return {
          ...s,
          approver_type: type,
          approver_ref: type === 'position' ? s.approver_ref : 0,
          name: defaultName(type, position?.name),
          nameEdited: false,
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
    // 前端校验：步骤名非空；指定岗位必须已选岗位
    for (const s of draftSteps) {
      if (!s.name.trim()) {
        toast.error(`第 ${s.step_order} 级步骤名称不能为空`)
        return
      }
      if (s.approver_type === 'position' && !s.approver_ref) {
        toast.error(`第 ${s.step_order} 级（指定岗位）请选择岗位`)
        return
      }
    }
    setSaving(true)
    try {
      await updateApprovalFlow(editingFlow.code, {
        name: editingFlow.name,
        steps: draftSteps.map(({ step_order, approver_type, approver_ref, name }) => ({
          step_order,
          approver_type,
          approver_ref,
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

  // 非 admin 直接访问时显示无权限提示
  if (!isAdmin) {
    return (
      <div className="flex justify-center py-24">
        <div className="alert alert-error max-w-lg">
          <span>无权限访问该页面，仅系统管理员可配置审批流程。</span>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-xl font-bold">审批流程配置</h2>
        <div className="alert alert-info shadow-sm">
          <span>配置请假/补卡的审批级别与每级审批人，修改仅影响新申请，已进行中的审批不受影响。</span>
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
                        className="flex items-center gap-2 rounded-box border border-base-200 px-3 py-2"
                      >
                        <span className="badge badge-ghost badge-sm">{s.step_order}</span>
                        <span className="text-sm font-medium">{s.name}</span>
                        <span className={`badge badge-sm ${TYPE_BADGE[s.approver_type]}`}>
                          {APPROVER_TYPE_LABEL[s.approver_type]}
                        </span>
                        {s.approver_type === 'position' && positionName(s.approver_ref) && (
                          <span className="text-xs text-base-content/50">
                            （{positionName(s.approver_ref)}）
                          </span>
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
          <div className="grid grid-cols-[2.5rem_1fr_8rem_2.5rem] items-center gap-2 px-1 text-xs text-base-content/50">
            <span>序号</span>
            <span>步骤名称</span>
            <span>审批人</span>
            <span className="text-right">操作</span>
          </div>

          {draftSteps.map((s) => (
            <div key={s.key} className="rounded-box border border-base-300 p-2">
              <div className="grid grid-cols-[2.5rem_1fr_8rem_2.5rem] items-center gap-2">
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
                  {(Object.keys(APPROVER_TYPE_LABEL) as ApprovalStepItem['approver_type'][]).map((t) => (
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
