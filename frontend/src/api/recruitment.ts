import client from './client'
import type { PageResult } from './client'

// 招聘职位（列表附带部门、岗位名称）
export interface JobOpening {
  id: number
  title: string
  department_id: number
  position_id: number
  headcount: number
  requirement: string
  status: string
  department_name?: string
  position_name?: string
  created_at: string
  updated_at: string
}

// 候选人（列表附带应聘职位名称）
export interface Candidate {
  id: number
  job_opening_id: number
  name: string
  phone: string
  email: string
  status: string
  interview_date: string | null
  remark: string
  job_title?: string
  created_at: string
  updated_at: string
}

// 招聘职位分页列表
export function fetchJobs(params: { status?: string; keyword?: string; page?: number; page_size?: number } = {}) {
  return client.get('/recruitment/jobs', { params }) as Promise<PageResult<JobOpening>>
}

// 创建招聘职位
export function createJob(data: Record<string, unknown>) {
  return client.post('/recruitment/jobs', data) as Promise<JobOpening>
}

// 更新招聘职位
export function updateJob(id: number, data: Record<string, unknown>) {
  return client.put(`/recruitment/jobs/${id}`, data) as Promise<null>
}

// 删除招聘职位
export function deleteJob(id: number) {
  return client.delete(`/recruitment/jobs/${id}`) as Promise<null>
}

// 候选人分页列表（后端读取 job_id 过滤应聘职位）
export function fetchCandidates(params: { status?: string; job_id?: number; keyword?: string; page?: number; page_size?: number } = {}) {
  return client.get('/recruitment/candidates', { params }) as Promise<PageResult<Candidate>>
}

// 全部招聘职位（供候选人表单下拉，拉取一页大分页）
export function fetchAllJobs() {
  return fetchJobs({ page: 1, page_size: 1000 })
}

// 创建候选人
export function createCandidate(data: Record<string, unknown>) {
  return client.post('/recruitment/candidates', data) as Promise<Candidate>
}

// 更新候选人
export function updateCandidate(id: number, data: Record<string, unknown>) {
  return client.put(`/recruitment/candidates/${id}`, data) as Promise<null>
}

// 删除候选人
export function deleteCandidate(id: number) {
  return client.delete(`/recruitment/candidates/${id}`) as Promise<null>
}
