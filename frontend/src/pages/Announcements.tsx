import { useEffect, useState } from 'react'
import {
  fetchAnnouncements,
  createAnnouncement,
  updateAnnouncement,
  deleteAnnouncement,
  markAllAnnouncementsRead,
  type Announcement,
} from '../api/announcement'
import type { PageResult } from '../api/client'
import { useAuthStore } from '../store/auth'
import { useBadgeStore } from '../store/badges'
import { toast } from '../store/toast'
import { Pin, Pencil, Trash2, Plus } from 'lucide-react'
import { formatDateTime } from '../utils/format'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'

const PAGE_SIZE = 10

// 公告表单
interface AnnouncementForm {
  title: string
  content: string
  is_pinned: boolean
}

const emptyForm: AnnouncementForm = {
  title: '',
  content: '',
  is_pinned: false,
}

export default function AnnouncementsPage() {
  const role = useAuthStore((s) => s.user?.role_code)
  // 公告增删改仅 hr（admin 只读监督）
  const canManage = role === 'hr'
  const setBadges = useBadgeStore((s) => s.setBadges)

  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [reloadKey, setReloadKey] = useState(0)

  const [data, setData] = useState<PageResult<Announcement> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // 新增/编辑弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Announcement | null>(null)
  const [form, setForm] = useState<AnnouncementForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchAnnouncements({
      keyword: keyword || undefined,
      page,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setData(res)
        // 阅读即已读：进入公告页若存在未读条目，标记全部已读并联动清零角标（本地同步置已读避免闪烁）
        if (res.list?.some((a) => !a.is_read)) {
          markAllAnnouncementsRead()
            .then(() => {
              setBadges({ unread_announcements: 0 })
              if (!cancelled) {
                setData((prev) => (prev ? { ...prev, list: prev.list.map((a) => ({ ...a, is_read: true })) } : prev))
              }
            })
            .catch(() => undefined)
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取公告失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [keyword, page, reloadKey])

  const reload = () => setReloadKey((k) => k + 1)

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setModalOpen(true)
  }

  const openEdit = (item: Announcement) => {
    setEditing(item)
    setForm({ title: item.title, content: item.content, is_pinned: item.is_pinned })
    setModalOpen(true)
  }

  const handleSave = async () => {
    if (!form.title.trim()) {
      toast.error('请输入公告标题')
      return
    }
    if (!form.content.trim()) {
      toast.error('请输入公告内容')
      return
    }
    setSaving(true)
    try {
      if (editing) {
        await updateAnnouncement(editing.id, {
          title: form.title.trim(),
          content: form.content.trim(),
          is_pinned: form.is_pinned,
        })
        toast.success('公告已更新')
      } else {
        await createAnnouncement({
          title: form.title.trim(),
          content: form.content.trim(),
          is_pinned: form.is_pinned,
        })
        toast.success('公告发布成功')
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (item: Announcement) => {
    if (!window.confirm(`确认删除公告「${item.title}」？`)) return
    deleteAnnouncement(item.id)
      .then(() => {
        toast.success('公告已删除')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '删除失败'))
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 工具栏 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body flex flex-row flex-wrap items-center gap-3 p-4">
          <div className="flex min-w-64 flex-1 gap-2">
            <input
              type="text"
              className="input input-bordered input-sm w-full"
              placeholder="搜索公告标题"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  setKeyword(search.trim())
                  setPage(1)
                }
              }}
            />
            <button
              type="button"
              className="btn btn-primary btn-sm"
              onClick={() => {
                setKeyword(search.trim())
                setPage(1)
              }}
            >
              搜索
            </button>
          </div>
          {canManage && (
            <div className="ml-auto">
              <button type="button" className="btn btn-primary btn-sm" onClick={openCreate}>
                <Plus className="h-4 w-4" />
                发布公告
              </button>
            </div>
          )}
        </div>
      </div>

      {/* 公告列表 */}
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
            <ul className="flex flex-col divide-y divide-base-200">
              {(data?.list ?? []).map((item) => (
                <li key={item.id} className="py-4 first:pt-0 last:pb-0">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        {item.is_pinned && (
                          <span className="badge badge-error badge-sm gap-0.5">
                            <Pin className="h-3 w-3" />
                            置顶
                          </span>
                        )}
                        {!item.is_read && <span className="badge badge-primary badge-xs">新</span>}
                        <h3 className={`text-base ${item.is_read ? 'font-medium' : 'font-semibold'}`}>{item.title}</h3>
                      </div>
                      <div className="mt-1 text-xs text-base-content/50">
                        {item.publisher_name || `#${item.publisher_id}`} · {formatDateTime(item.published_at)}
                      </div>
                      <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-base-content/80">
                        {item.content}
                      </p>
                    </div>
                    {canManage && (
                      <div className="flex shrink-0 gap-1">
                        <button type="button" className="btn btn-ghost btn-xs" onClick={() => openEdit(item)}>
                          <Pencil className="h-3.5 w-3.5" />
                          编辑
                        </button>
                        <button
                          type="button"
                          className="btn btn-ghost btn-xs text-error"
                          onClick={() => handleDelete(item)}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                          删除
                        </button>
                      </div>
                    )}
                  </div>
                </li>
              ))}
              {!data?.list.length && (
                <li className="py-10 text-center text-base-content/50">暂无公告</li>
              )}
            </ul>
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
        title={editing ? '编辑公告' : '发布公告'}
        onClose={() => setModalOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setModalOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={handleSave}>
              {saving && <span className="loading loading-spinner loading-sm" />}
              {editing ? '保存' : '发布'}
            </button>
          </>
        }
      >
        <div className="flex flex-col gap-3">
          <Field label="标题" required>
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              placeholder="请输入公告标题"
              required
            />
          </Field>
          <Field label="内容" required>
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={6}
              value={form.content}
              onChange={(e) => setForm({ ...form, content: e.target.value })}
              placeholder="请输入公告内容"
              required
            />
          </Field>
          <label className="flex cursor-pointer items-center gap-2 text-sm">
            <input
              type="checkbox"
              className="checkbox checkbox-primary checkbox-sm"
              checked={form.is_pinned}
              onChange={(e) => setForm({ ...form, is_pinned: e.target.checked })}
            />
            置顶显示
          </label>
        </div>
      </Modal>
    </div>
  )
}
