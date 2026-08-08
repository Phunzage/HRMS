// 状态映射统一管理：标签 + daisyUI badge 颜色类
// 各业务状态枚举集中在此，供表格/筛选下拉复用，避免散落各处

export interface BadgeItem {
  label: string
  badge: string
}

// 员工状态：1在职 / 2离职 / 3试用
export const EMPLOYEE_STATUS: Record<number, BadgeItem> = {
  1: { label: '在职', badge: 'badge-success' },
  2: { label: '离职', badge: 'badge-error' },
  3: { label: '试用', badge: 'badge-warning' },
}

// 性别：1男 / 2女 / 3其他
export const GENDER: Record<number, string> = {
  1: '男',
  2: '女',
  3: '其他',
}

// 考勤状态
export const ATTENDANCE_STATUS: Record<string, BadgeItem> = {
  normal: { label: '正常', badge: 'badge-success' },
  late: { label: '迟到', badge: 'badge-warning' },
  early_leave: { label: '早退', badge: 'badge-warning' },
  absent: { label: '缺勤', badge: 'badge-error' },
  leave: { label: '请假', badge: 'badge-info' },
}

// 补卡申请状态
export const CORRECTION_STATUS: Record<string, BadgeItem> = {
  pending: { label: '待审批', badge: 'badge-warning' },
  approved: { label: '已通过', badge: 'badge-success' },
  rejected: { label: '已驳回', badge: 'badge-error' },
  cancelled: { label: '已取消', badge: 'badge-ghost' },
}

// 补卡类型
export const CORRECTION_TYPE: Record<string, string> = {
  check_in: '上班打卡',
  check_out: '下班打卡',
}

// 绩效状态
export const PERFORMANCE_STATUS: Record<string, BadgeItem> = {
  draft: { label: '草稿', badge: 'badge-ghost' },
  submitted: { label: '待审批', badge: 'badge-warning' },
  approved: { label: '已批准', badge: 'badge-success' },
}

// 绩效评级（S绿/A蓝/B黄/C橙/D红，对应 task 要求；B/C 用 Tailwind 精确色值区分黄/橙）
export const PERFORMANCE_RATING: Record<string, BadgeItem> = {
  S: { label: 'S', badge: 'badge-success' },
  A: { label: 'A', badge: 'badge-info' },
  B: { label: 'B', badge: 'bg-yellow-500 border-yellow-500 text-white' },
  C: { label: 'C', badge: 'bg-orange-500 border-orange-500 text-white' },
  D: { label: 'D', badge: 'badge-error' },
}

// 公告状态
export const ANNOUNCEMENT_STATUS: Record<string, BadgeItem> = {
  published: { label: '已发布', badge: 'badge-success' },
  draft: { label: '草稿', badge: 'badge-ghost' },
}

// 请假申请状态
export const LEAVE_STATUS: Record<string, BadgeItem> = {
  pending: { label: '待审批', badge: 'badge-warning' },
  approved: { label: '已通过', badge: 'badge-success' },
  rejected: { label: '已驳回', badge: 'badge-error' },
  cancelled: { label: '已取消', badge: 'badge-ghost' },
}

// 请假类型
export const LEAVE_TYPE: Record<string, string> = {
  annual: '年假',
  sick: '病假',
  personal: '事假',
  marriage: '婚假',
  maternity: '产假',
  other: '其他',
}

// 工资单状态
export const SALARY_STATUS: Record<string, BadgeItem> = {
  draft: { label: '草稿', badge: 'badge-ghost' },
  confirmed: { label: '已确认', badge: 'badge-info' },
  paid: { label: '已发放', badge: 'badge-success' },
}

// 招聘职位状态
export const JOB_STATUS: Record<string, BadgeItem> = {
  open: { label: '招聘中', badge: 'badge-success' },
  closed: { label: '已关闭', badge: 'badge-ghost' },
}

// 候选人状态
export const CANDIDATE_STATUS: Record<string, BadgeItem> = {
  applied: { label: '已投递', badge: 'badge-ghost' },
  interview: { label: '面试中', badge: 'badge-warning' },
  offer: { label: '已发Offer', badge: 'badge-info' },
  hired: { label: '已录用', badge: 'badge-success' },
  rejected: { label: '已拒绝', badge: 'badge-error' },
}
