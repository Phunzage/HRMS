import { useEffect, useState } from 'react'
import {
  fetchJobs,
  fetchAllJobs,
  createJob,
  updateJob,
  deleteJob,
  fetchCandidates,
  createCandidate,
  updateCandidate,
  deleteCandidate,
  type JobOpening,
  type Candidate,
} from '../api/recruitment'
import { fetchDepartments, fetchAllPositions, type Position, type Department } from '../api/employee'
import type { PageResult } from '../api/client'
import { usePerm } from '../hooks/usePerm'
import { toast } from '../store/toast'
import { JOB_STATUS, CANDIDATE_STATUS } from '../utils/status'
import { formatDate } from '../utils/format'
import { flattenDepartments, deptOptionLabel } from '../utils/tree'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'
import StatusBadge from '../components/StatusBadge'

const PAGE_SIZE = 10

// 招聘职位表单
interface JobForm {
  title: string
  department_id: string
  position_id: string
  headcount: string
  requirement: string
  status: string
}

const emptyJobForm: JobForm = {
  title: '',
  department_id: '',
  position_id: '',
  headcount: '1',
  requirement: '',
  status: 'open',
}

// 候选人表单
interface CandidateForm {
  job_opening_id: string
  name: string
  phone: string
  email: string
  status: string
  remark: string
}

const emptyCandidateForm: CandidateForm = {
  job_opening_id: '',
  name: '',
  phone: '',
  email: '',
  status: 'applied',
  remark: '',
}

type TabKey = 'jobs' | 'candidates'

export default function RecruitmentPage() {
  // 招聘增删改需 recruitment:write 权限点（种子仅 hr，可在角色管理动态调整）
  const { has } = usePerm()
  const canManage = has('recruitment:write')

  const [activeTab, setActiveTab] = useState<TabKey>('jobs')

  // 下拉数据
  const [deptOptions, setDeptOptions] = useState<{ id: number; name: string; level: number }[]>([])
  const [positions, setPositions] = useState<Position[]>([])
  const [jobOptions, setJobOptions] = useState<JobOpening[]>([])

  // 职位列表
  const [jobs, setJobs] = useState<PageResult<JobOpening> | null>(null)
  const [jobsLoading, setJobsLoading] = useState(true)
  const [jobsError, setJobsError] = useState('')
  const [jobsKeyword, setJobsKeyword] = useState('')
  const [jobsSearch, setJobsSearch] = useState('')
  const [jobsStatus, setJobsStatus] = useState('')
  const [jobsPage, setJobsPage] = useState(1)
  const [jobsReload, setJobsReload] = useState(0)

  // 候选人列表
  const [candidates, setCandidates] = useState<PageResult<Candidate> | null>(null)
  const [candLoading, setCandLoading] = useState(true)
  const [candError, setCandError] = useState('')
  const [candKeyword, setCandKeyword] = useState('')
  const [candSearch, setCandSearch] = useState('')
  const [candStatus, setCandStatus] = useState('')
  const [candJob, setCandJob] = useState('')
  const [candPage, setCandPage] = useState(1)
  const [candReload, setCandReload] = useState(0)

  // 弹窗状态
  const [jobModalOpen, setJobModalOpen] = useState(false)
  const [editingJob, setEditingJob] = useState<JobOpening | null>(null)
  const [jobForm, setJobForm] = useState<JobForm>(emptyJobForm)

  const [candModalOpen, setCandModalOpen] = useState(false)
  const [editingCand, setEditingCand] = useState<Candidate | null>(null)
  const [candForm, setCandForm] = useState<CandidateForm>(emptyCandidateForm)

  const [saving, setSaving] = useState(false)

  // 加载下拉数据；非关键下拉加载失败时静默降级为空数组
  useEffect(() => {
    fetchDepartments()
      .then((tree: Department[]) => setDeptOptions(flattenDepartments(tree)))
      .catch(() => setDeptOptions([]))
    fetchAllPositions()
      .then((res) => setPositions(res.list ?? []))
      .catch(() => setPositions([]))
    fetchAllJobs()
      .then((res) => setJobOptions(res.list ?? []))
      .catch(() => setJobOptions([]))
  }, [])

  // 职位列表
  useEffect(() => {
    if (activeTab !== 'jobs') return
    let cancelled = false
    setJobsLoading(true)
    setJobsError('')
    fetchJobs({
      keyword: jobsKeyword || undefined,
      status: jobsStatus || undefined,
      page: jobsPage,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setJobs(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setJobsError(err instanceof Error ? err.message : '获取职位列表失败')
      })
      .finally(() => {
        if (!cancelled) setJobsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [activeTab, jobsKeyword, jobsStatus, jobsPage, jobsReload])

  // 候选人列表
  useEffect(() => {
    if (activeTab !== 'candidates') return
    let cancelled = false
    setCandLoading(true)
    setCandError('')
    fetchCandidates({
      keyword: candKeyword || undefined,
      status: candStatus || undefined,
      job_id: candJob ? Number(candJob) : undefined,
      page: candPage,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setCandidates(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setCandError(err instanceof Error ? err.message : '获取候选人列表失败')
      })
      .finally(() => {
        if (!cancelled) setCandLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [activeTab, candKeyword, candStatus, candJob, candPage, candReload])

  const openJobCreate = () => {
    setEditingJob(null)
    setJobForm(emptyJobForm)
    setJobModalOpen(true)
  }

  const openJobEdit = (job: JobOpening) => {
    setEditingJob(job)
    setJobForm({
      title: job.title,
      department_id: String(job.department_id),
      position_id: String(job.position_id),
      headcount: String(job.headcount),
      requirement: job.requirement,
      status: job.status,
    })
    setJobModalOpen(true)
  }

  const handleJobSave = async () => {
    const payload = {
      title: jobForm.title,
      department_id: Number(jobForm.department_id),
      position_id: Number(jobForm.position_id),
      headcount: Number(jobForm.headcount),
      requirement: jobForm.requirement,
      status: jobForm.status,
    }
    setSaving(true)
    try {
      if (editingJob) {
        await updateJob(editingJob.id, payload)
        toast.success('招聘职位已更新')
      } else {
        await createJob(payload)
        toast.success('招聘职位创建成功')
      }
      setJobModalOpen(false)
      setJobsReload((k) => k + 1)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleJobDelete = (job: JobOpening) => {
    if (!window.confirm(`确认删除招聘职位「${job.title}」？`)) return
    deleteJob(job.id)
      .then(() => {
        toast.success('招聘职位已删除')
        setJobsReload((k) => k + 1)
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '删除失败'))
  }

  const openCandCreate = () => {
    setEditingCand(null)
    setCandForm(emptyCandidateForm)
    setCandModalOpen(true)
  }

  const openCandEdit = (c: Candidate) => {
    setEditingCand(c)
    setCandForm({
      job_opening_id: String(c.job_opening_id),
      name: c.name,
      phone: c.phone,
      email: c.email,
      status: c.status,
      remark: c.remark,
    })
    setCandModalOpen(true)
  }

  const handleCandSave = async () => {
    const payload = {
      job_opening_id: Number(candForm.job_opening_id),
      name: candForm.name,
      phone: candForm.phone,
      email: candForm.email,
      status: candForm.status,
      remark: candForm.remark,
    }
    setSaving(true)
    try {
      if (editingCand) {
        await updateCandidate(editingCand.id, payload)
        toast.success('候选人已更新')
      } else {
        await createCandidate(payload)
        toast.success('候选人添加成功')
      }
      setCandModalOpen(false)
      setCandReload((k) => k + 1)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleCandDelete = (c: Candidate) => {
    if (!window.confirm(`确认删除候选人「${c.name}」？`)) return
    deleteCandidate(c.id)
      .then(() => {
        toast.success('候选人已删除')
        setCandReload((k) => k + 1)
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '删除失败'))
  }

  // 候选人状态流转：行内下拉直接更新
  const handleCandStatusChange = (c: Candidate, status: string) => {
    if (status === c.status) return
    updateCandidate(c.id, { status })
      .then(() => {
        toast.success('候选人状态已更新')
        setCandReload((k) => k + 1)
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '状态更新失败'))
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Tab */}
      <div role="tablist" className="tabs tabs-boxed w-fit">
        <button
          type="button"
          role="tab"
          className={`tab ${activeTab === 'jobs' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('jobs')}
        >
          招聘职位
        </button>
        <button
          type="button"
          role="tab"
          className={`tab ${activeTab === 'candidates' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('candidates')}
        >
          候选人
        </button>
      </div>

      {activeTab === 'jobs' ? (
        <>
          {/* 职位工具栏 */}
          <div className="card bg-base-100 shadow-sm">
            <div className="card-body flex flex-row flex-wrap items-center gap-3 p-4">
              <div className="flex min-w-64 flex-1 gap-2">
                <input
                  type="text"
                  className="input input-bordered input-sm w-full"
                  placeholder="搜索职位标题"
                  value={jobsSearch}
                  onChange={(e) => setJobsSearch(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      setJobsKeyword(jobsSearch.trim())
                      setJobsPage(1)
                    }
                  }}
                />
                <button
                  type="button"
                  className="btn btn-primary btn-sm"
                  onClick={() => {
                    setJobsKeyword(jobsSearch.trim())
                    setJobsPage(1)
                  }}
                >
                  搜索
                </button>
              </div>
              <select
                className="select select-bordered select-sm w-32"
                value={jobsStatus}
                onChange={(e) => {
                  setJobsStatus(e.target.value)
                  setJobsPage(1)
                }}
              >
                <option value="">全部状态</option>
                <option value="open">招聘中</option>
                <option value="closed">已关闭</option>
              </select>
              {canManage && (
                <button type="button" className="btn btn-primary btn-sm" onClick={openJobCreate}>
                  ＋ 新增职位
                </button>
              )}
            </div>
          </div>

          {/* 职位表格 */}
          <div className="card bg-base-100 shadow-sm">
            <div className="card-body p-4">
              {jobsError ? (
                <div className="alert alert-error">
                  <span>{jobsError}</span>
                </div>
              ) : jobsLoading ? (
                <div className="flex items-center justify-center py-16">
                  <span className="loading loading-spinner loading-lg text-primary" />
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="table table-sm">
                    <thead>
                      <tr>
                        <th>职位标题</th>
                        <th>部门</th>
                        <th>岗位</th>
                        <th>编制</th>
                        <th>状态</th>
                        {canManage && <th className="text-right">操作</th>}
                      </tr>
                    </thead>
                    <tbody>
                      {(jobs?.list ?? []).map((job) => (
                        <tr key={job.id}>
                          <td className="font-medium">{job.title}</td>
                          <td>{job.department_name || '-'}</td>
                          <td>{job.position_name || '-'}</td>
                          <td>{job.headcount}</td>
                          <td>
                            <StatusBadge map={JOB_STATUS} value={job.status} />
                          </td>
                          {canManage && (
                            <td className="text-right">
                              <div className="flex justify-end gap-1">
                                <button type="button" className="btn btn-ghost btn-xs" onClick={() => openJobEdit(job)}>
                                  编辑
                                </button>
                                <button
                                  type="button"
                                  className="btn btn-ghost btn-xs text-error"
                                  onClick={() => handleJobDelete(job)}
                                >
                                  删除
                                </button>
                              </div>
                            </td>
                          )}
                        </tr>
                      ))}
                      {!jobs?.list.length && (
                        <tr>
                          <td colSpan={canManage ? 6 : 5} className="py-8 text-center text-base-content/50">
                            暂无招聘职位
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}
              {jobs && (
                <div className="mt-2">
                  <Pagination page={jobsPage} pageSize={PAGE_SIZE} total={jobs.total} onChange={setJobsPage} />
                </div>
              )}
            </div>
          </div>
        </>
      ) : (
        <>
          {/* 候选人工具栏 */}
          <div className="card bg-base-100 shadow-sm">
            <div className="card-body flex flex-row flex-wrap items-center gap-3 p-4">
              <div className="flex min-w-64 flex-1 gap-2">
                <input
                  type="text"
                  className="input input-bordered input-sm w-full"
                  placeholder="搜索候选人姓名 / 电话"
                  value={candSearch}
                  onChange={(e) => setCandSearch(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      setCandKeyword(candSearch.trim())
                      setCandPage(1)
                    }
                  }}
                />
                <button
                  type="button"
                  className="btn btn-primary btn-sm"
                  onClick={() => {
                    setCandKeyword(candSearch.trim())
                    setCandPage(1)
                  }}
                >
                  搜索
                </button>
              </div>
              <select
                className="select select-bordered select-sm w-44"
                value={candJob}
                onChange={(e) => {
                  setCandJob(e.target.value)
                  setCandPage(1)
                }}
              >
                <option value="">全部职位</option>
                {jobOptions.map((j) => (
                  <option key={j.id} value={j.id}>
                    {j.title}
                  </option>
                ))}
              </select>
              <select
                className="select select-bordered select-sm w-32"
                value={candStatus}
                onChange={(e) => {
                  setCandStatus(e.target.value)
                  setCandPage(1)
                }}
              >
                <option value="">全部状态</option>
                <option value="applied">已投递</option>
                <option value="interview">面试中</option>
                <option value="offer">已发Offer</option>
                <option value="hired">已录用</option>
                <option value="rejected">已拒绝</option>
              </select>
              {canManage && (
                <button type="button" className="btn btn-primary btn-sm" onClick={openCandCreate}>
                  ＋ 添加候选人
                </button>
              )}
            </div>
          </div>

          {/* 候选人表格 */}
          <div className="card bg-base-100 shadow-sm">
            <div className="card-body p-4">
              {candError ? (
                <div className="alert alert-error">
                  <span>{candError}</span>
                </div>
              ) : candLoading ? (
                <div className="flex items-center justify-center py-16">
                  <span className="loading loading-spinner loading-lg text-primary" />
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="table table-sm">
                    <thead>
                      <tr>
                        <th>姓名</th>
                        <th>电话</th>
                        <th>邮箱</th>
                        <th>应聘职位</th>
                        <th>面试日期</th>
                        <th>状态</th>
                        {canManage && <th className="text-right">操作</th>}
                      </tr>
                    </thead>
                    <tbody>
                      {(candidates?.list ?? []).map((c) => (
                        <tr key={c.id}>
                          <td className="font-medium">{c.name}</td>
                          <td>{c.phone || '-'}</td>
                          <td className="max-w-32 truncate">{c.email || '-'}</td>
                          <td>{c.job_title || '-'}</td>
                          <td>{formatDate(c.interview_date)}</td>
                          <td>
                            {canManage ? (
                              <select
                                className="select select-bordered select-xs w-28"
                                value={c.status}
                                onChange={(e) => handleCandStatusChange(c, e.target.value)}
                              >
                                {Object.entries(CANDIDATE_STATUS).map(([value, item]) => (
                                  <option key={value} value={value}>
                                    {item.label}
                                  </option>
                                ))}
                              </select>
                            ) : (
                              <StatusBadge map={CANDIDATE_STATUS} value={c.status} />
                            )}
                          </td>
                          {canManage && (
                            <td className="text-right">
                              <div className="flex justify-end gap-1">
                                <button type="button" className="btn btn-ghost btn-xs" onClick={() => openCandEdit(c)}>
                                  编辑
                                </button>
                                <button
                                  type="button"
                                  className="btn btn-ghost btn-xs text-error"
                                  onClick={() => handleCandDelete(c)}
                                >
                                  删除
                                </button>
                              </div>
                            </td>
                          )}
                        </tr>
                      ))}
                      {!candidates?.list.length && (
                        <tr>
                          <td colSpan={canManage ? 7 : 6} className="py-8 text-center text-base-content/50">
                            暂无候选人
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}
              {candidates && (
                <div className="mt-2">
                  <Pagination page={candPage} pageSize={PAGE_SIZE} total={candidates.total} onChange={setCandPage} />
                </div>
              )}
            </div>
          </div>
        </>
      )}

      {/* 职位 Modal */}
      <Modal
        open={jobModalOpen}
        title={editingJob ? `编辑职位 - ${editingJob.title}` : '新增招聘职位'}
        onClose={() => setJobModalOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setJobModalOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={handleJobSave}>
              {saving && <span className="loading loading-spinner loading-sm" />}
              保存
            </button>
          </>
        }
      >
        <div className="grid grid-cols-2 gap-3">
          <Field label="职位标题" required className="col-span-2">
            <input
              type="text"
              className="input input-bordered input-sm"
              value={jobForm.title}
              onChange={(e) => setJobForm({ ...jobForm, title: e.target.value })}
              placeholder="如 高级后端工程师"
              required
            />
          </Field>
          <Field label="需求部门" required>
            <select
              className="select select-bordered select-sm"
              value={jobForm.department_id}
              onChange={(e) => setJobForm({ ...jobForm, department_id: e.target.value, position_id: '' })}
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
          <Field label="对应岗位" required>
            <select
              className="select select-bordered select-sm"
              value={jobForm.position_id}
              onChange={(e) => setJobForm({ ...jobForm, position_id: e.target.value })}
              required
            >
              <option value="">请选择岗位</option>
              {positions
                .filter((p) => !jobForm.department_id || p.department_id === Number(jobForm.department_id))
                .map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
            </select>
          </Field>
          <Field label="招聘人数">
            <input
              type="number"
              min="1"
              className="input input-bordered input-sm"
              value={jobForm.headcount}
              onChange={(e) => setJobForm({ ...jobForm, headcount: e.target.value })}
            />
          </Field>
          <Field label="状态">
            <select
              className="select select-bordered select-sm"
              value={jobForm.status}
              onChange={(e) => setJobForm({ ...jobForm, status: e.target.value })}
            >
              <option value="open">招聘中</option>
              <option value="closed">已关闭</option>
            </select>
          </Field>
          <Field label="任职要求" className="col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={3}
              value={jobForm.requirement}
              onChange={(e) => setJobForm({ ...jobForm, requirement: e.target.value })}
            />
          </Field>
        </div>
      </Modal>

      {/* 候选人 Modal */}
      <Modal
        open={candModalOpen}
        title={editingCand ? `编辑候选人 - ${editingCand.name}` : '添加候选人'}
        onClose={() => setCandModalOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setCandModalOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={handleCandSave}>
              {saving && <span className="loading loading-spinner loading-sm" />}
              保存
            </button>
          </>
        }
      >
        <div className="grid grid-cols-2 gap-3">
          <Field label="应聘职位" required className="col-span-2">
            <select
              className="select select-bordered select-sm"
              value={candForm.job_opening_id}
              onChange={(e) => setCandForm({ ...candForm, job_opening_id: e.target.value })}
              required
            >
              <option value="">请选择职位</option>
              {jobOptions.map((j) => (
                <option key={j.id} value={j.id}>
                  {j.title}
                </option>
              ))}
            </select>
          </Field>
          <Field label="姓名" required>
            <input
              type="text"
              className="input input-bordered input-sm"
              value={candForm.name}
              onChange={(e) => setCandForm({ ...candForm, name: e.target.value })}
              required
            />
          </Field>
          <Field label="联系电话">
            <input
              type="text"
              className="input input-bordered input-sm"
              value={candForm.phone}
              onChange={(e) => setCandForm({ ...candForm, phone: e.target.value })}
            />
          </Field>
          <Field label="邮箱">
            <input
              type="email"
              className="input input-bordered input-sm"
              value={candForm.email}
              onChange={(e) => setCandForm({ ...candForm, email: e.target.value })}
            />
          </Field>
          {editingCand && (
            <Field label="状态">
              <select
                className="select select-bordered select-sm"
                value={candForm.status}
                onChange={(e) => setCandForm({ ...candForm, status: e.target.value })}
              >
                {Object.entries(CANDIDATE_STATUS).map(([value, item]) => (
                  <option key={value} value={value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </Field>
          )}
          <Field label="备注" className="col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={2}
              value={candForm.remark}
              onChange={(e) => setCandForm({ ...candForm, remark: e.target.value })}
            />
          </Field>
        </div>
      </Modal>
    </div>
  )
}
