import { useEffect, useMemo, useState } from 'react'
import {
  fetchEmployees,
  fetchDepartments,
  fetchAllPositions,
  createEmployee,
  updateEmployee,
  deleteEmployee,
  type Employee,
  type Position,
  type Department,
} from '../api/employee'
import type { PageResult } from '../api/client'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import { EMPLOYEE_STATUS, GENDER } from '../utils/status'
import { formatDate } from '../utils/format'
import { flattenDepartments, deptOptionLabel, type DeptOption } from '../utils/tree'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'

const PAGE_SIZE = 10

// 新增/编辑表单数据
interface EmployeeForm {
  name: string
  gender: number
  birth_date: string
  id_card: string
  phone: string
  email: string
  department_id: string
  position_id: string
  hire_date: string
  status: number
  address: string
  remark: string
  create_user: boolean
  username: string
}

const emptyForm: EmployeeForm = {
  name: '',
  gender: 1,
  birth_date: '',
  id_card: '',
  phone: '',
  email: '',
  department_id: '',
  position_id: '',
  hire_date: '',
  status: 1,
  address: '',
  remark: '',
  create_user: false,
  username: '',
}

export default function EmployeeList() {
  const role = useAuthStore((s) => s.user?.role_code)
  const canManage = role === 'admin' || role === 'hr'

  // 列表查询条件
  const [keyword, setKeyword] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [departmentId, setDepartmentId] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [reloadKey, setReloadKey] = useState(0)

  const [data, setData] = useState<PageResult<Employee> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // 下拉数据
  const [deptOptions, setDeptOptions] = useState<DeptOption[]>([])
  const [positions, setPositions] = useState<Position[]>([])

  // Modal 状态
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Employee | null>(null)
  const [form, setForm] = useState<EmployeeForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  // 加载列表
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchEmployees({
      keyword: keyword || undefined,
      department_id: departmentId ? Number(departmentId) : undefined,
      status: status ? Number(status) : undefined,
      page,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取员工列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [keyword, departmentId, status, page, reloadKey])

  // 加载下拉数据（部门树 + 岗位列表）；非关键下拉加载失败时静默降级为空数组，避免影响主列表
  useEffect(() => {
    fetchDepartments()
      .then((tree: Department[]) => setDeptOptions(flattenDepartments(tree)))
      .catch(() => setDeptOptions([]))
    fetchAllPositions()
      .then((res) => setPositions(res.list))
      .catch(() => setPositions([]))
  }, [])

  // 当前部门可选的岗位
  const positionOptions = useMemo(() => {
    if (!form.department_id) return positions
    return positions.filter((p) => p.department_id === Number(form.department_id))
  }, [positions, form.department_id])

  const reload = () => setReloadKey((k) => k + 1)

  const handleSearch = () => {
    setKeyword(searchInput.trim())
    setPage(1)
  }

  // 打开新增弹窗
  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setModalOpen(true)
  }

  // 打开编辑弹窗
  const openEdit = (emp: Employee) => {
    setEditing(emp)
    setForm({
      name: emp.name,
      gender: emp.gender,
      birth_date: emp.birth_date ? emp.birth_date.slice(0, 10) : '',
      id_card: emp.id_card,
      phone: emp.phone,
      email: emp.email,
      department_id: String(emp.department_id),
      position_id: String(emp.position_id),
      hire_date: emp.hire_date ? emp.hire_date.slice(0, 10) : '',
      status: emp.status,
      address: emp.address,
      remark: emp.remark,
      create_user: false,
      username: '',
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    const payload = {
      name: form.name,
      gender: Number(form.gender),
      birth_date: form.birth_date || null,
      id_card: form.id_card,
      phone: form.phone,
      email: form.email,
      department_id: Number(form.department_id),
      position_id: Number(form.position_id),
      hire_date: form.hire_date || null,
      status: Number(form.status),
      address: form.address,
      remark: form.remark,
    }
    setSaving(true)
    try {
      if (editing) {
        await updateEmployee(editing.id, payload)
        toast.success('员工信息已更新')
      } else {
        const res = await createEmployee({
          ...payload,
          create_user: form.create_user,
          username: form.username || undefined,
        })
        if (form.create_user && res.username) {
          toast.success(`员工创建成功，账号：${res.username}，默认密码：${res.default_password}`)
        } else {
          toast.success('员工创建成功')
        }
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (emp: Employee) => {
    if (!window.confirm(`确认删除员工「${emp.name}」？删除后其账号将一并停用。`)) return
    deleteEmployee(emp.id)
      .then(() => {
        toast.success('员工已删除')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '删除失败'))
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 工具栏 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body flex flex-row flex-wrap items-end gap-3 p-4">
          <div className="flex min-w-64 flex-1 gap-2">
            <input
              type="text"
              className="input input-bordered input-sm w-full"
              placeholder="搜索姓名 / 工号 / 手机号"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            />
            <button type="button" className="btn btn-primary btn-sm" onClick={handleSearch}>
              搜索
            </button>
          </div>
          <select
            className="select select-bordered select-sm w-40"
            value={departmentId}
            onChange={(e) => {
              setDepartmentId(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部部门</option>
            {deptOptions.map((d) => (
              <option key={d.id} value={d.id}>
                {deptOptionLabel(d)}
              </option>
            ))}
          </select>
          <select
            className="select select-bordered select-sm w-32"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部状态</option>
            <option value="1">在职</option>
            <option value="2">离职</option>
            <option value="3">试用</option>
          </select>
          {canManage && (
            <button type="button" className="btn btn-primary btn-sm" onClick={openCreate}>
              ＋ 新增员工
            </button>
          )}
        </div>
      </div>

      {/* 列表 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body p-4">
          {error ? (
            <div className="alert alert-error">
              <span>{error}</span>
            </div>
          ) : loading ? (
            <div className="flex items-center justify-center py-16">
              <span className="loading loading-spinner loading-lg text-primary" />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th>工号</th>
                    <th>姓名</th>
                    <th>性别</th>
                    <th>部门</th>
                    <th>岗位</th>
                    <th>手机号</th>
                    <th>入职日期</th>
                    <th>状态</th>
                    {canManage && <th className="text-right">操作</th>}
                  </tr>
                </thead>
                <tbody>
                  {(data?.list ?? []).map((emp) => (
                    <tr key={emp.id}>
                      <td className="font-mono text-xs">{emp.emp_no}</td>
                      <td className="font-medium">{emp.name}</td>
                      <td>{GENDER[emp.gender] ?? emp.gender}</td>
                      <td>{emp.department_name || '-'}</td>
                      <td>{emp.position_name || '-'}</td>
                      <td>{emp.phone || '-'}</td>
                      <td>{formatDate(emp.hire_date)}</td>
                      <td>
                        <span className={`badge badge-sm ${EMPLOYEE_STATUS[emp.status]?.badge ?? 'badge-ghost'}`}>
                          {EMPLOYEE_STATUS[emp.status]?.label ?? emp.status}
                        </span>
                      </td>
                      {canManage && (
                        <td className="text-right">
                          <div className="flex justify-end gap-1">
                            <button type="button" className="btn btn-ghost btn-xs" onClick={() => openEdit(emp)}>
                              编辑
                            </button>
                            <button type="button" className="btn btn-ghost btn-xs text-error" onClick={() => handleDelete(emp)}>
                              删除
                            </button>
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                  {!data?.list.length && (
                    <tr>
                      <td colSpan={canManage ? 9 : 8} className="py-8 text-center text-base-content/50">
                        暂无员工数据
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
          {data && (
            <div className="mt-2">
              <Pagination page={page} pageSize={PAGE_SIZE} total={data.total} onChange={setPage} />
            </div>
          )}
        </div>
      </div>

      {/* 新增/编辑 Modal */}
      <Modal
        open={modalOpen}
        title={editing ? `编辑员工 - ${editing.name}` : '新增员工'}
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
        <div className="grid grid-cols-2 gap-3">
          <Field label="姓名" required>
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </Field>
          <Field label="性别">
            <select
              className="select select-bordered select-sm"
              value={form.gender}
              onChange={(e) => setForm({ ...form, gender: Number(e.target.value) })}
            >
              <option value={1}>男</option>
              <option value={2}>女</option>
              <option value={3}>其他</option>
            </select>
          </Field>
          <Field label="手机号">
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.phone}
              onChange={(e) => setForm({ ...form, phone: e.target.value })}
            />
          </Field>
          <Field label="邮箱">
            <input
              type="email"
              className="input input-bordered input-sm"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </Field>
          <Field label="身份证号">
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.id_card}
              onChange={(e) => setForm({ ...form, id_card: e.target.value })}
            />
          </Field>
          <Field label="出生日期">
            <input
              type="date"
              className="input input-bordered input-sm"
              value={form.birth_date}
              onChange={(e) => setForm({ ...form, birth_date: e.target.value })}
            />
          </Field>
          <Field label="所属部门" required>
            <select
              className="select select-bordered select-sm"
              value={form.department_id}
              onChange={(e) => setForm({ ...form, department_id: e.target.value, position_id: '' })}
              required
            >
              <option value="">请选择部门</option>
              {deptOptions.map((d) => (
                <option key={d.id} value={d.id}>
                  {deptOptionLabel(d)}
                </option>
              ))}
            </select>
          </Field>
          <Field label="所属岗位" required>
            <select
              className="select select-bordered select-sm"
              value={form.position_id}
              onChange={(e) => setForm({ ...form, position_id: e.target.value })}
              required
            >
              <option value="">请选择岗位</option>
              {positionOptions.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="入职日期">
            <input
              type="date"
              className="input input-bordered input-sm"
              value={form.hire_date}
              onChange={(e) => setForm({ ...form, hire_date: e.target.value })}
            />
          </Field>
          <Field label="员工状态">
            <select
              className="select select-bordered select-sm"
              value={form.status}
              onChange={(e) => setForm({ ...form, status: Number(e.target.value) })}
            >
              <option value={1}>在职</option>
              <option value={3}>试用</option>
              <option value={2}>离职</option>
            </select>
          </Field>
          <Field label="家庭住址" className="col-span-2">
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.address}
              onChange={(e) => setForm({ ...form, address: e.target.value })}
            />
          </Field>
          <Field label="备注" className="col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={2}
              value={form.remark}
              onChange={(e) => setForm({ ...form, remark: e.target.value })}
            />
          </Field>

          {!editing && (
            <>
              <label className="label cursor-pointer gap-2 col-span-2 justify-start">
                <input
                  type="checkbox"
                  className="checkbox checkbox-sm"
                  checked={form.create_user}
                  onChange={(e) => setForm({ ...form, create_user: e.target.checked })}
                />
                <span className="label-text">同时创建系统账号（默认密码 123456）</span>
              </label>
              {form.create_user && (
                <Field label="账号用户名（留空自动按姓名拼音生成）" className="col-span-2">
                  <input
                    type="text"
                    className="input input-bordered input-sm"
                    value={form.username}
                    onChange={(e) => setForm({ ...form, username: e.target.value })}
                    placeholder="如 zhangsan"
                  />
                </Field>
              )}
            </>
          )}
        </div>
      </Modal>
    </div>
  )
}
