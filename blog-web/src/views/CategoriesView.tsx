import { useCallback, useState } from 'react'
import { categoryApi } from '../api'
import { errorMessage } from '../api/http'
import type { Category } from '../api/types'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { notify } from '../stores/notice'
import WorkspaceNav from '../components/WorkspaceNav'
import StatePanel from '../components/StatePanel'
import Pagination from '../components/Pagination'
import UiIcon from '../components/UiIcon'

type CategoryForm = Omit<Category, 'id'> & { id?: number }
const empty = (): CategoryForm => ({ name: '', sort: 0, status: 1 })

export default function CategoriesView() {
  const [page, setPage] = useState(1)
  const categories = useResource(useCallback((signal) => categoryApi.list({ pageNum: page, pageSize: 10 }, signal), [page]))
  const [form, setForm] = useState<CategoryForm>(empty)
  const [formError, setFormError] = useState('')
  const { busy, run } = useMutation(page)
  function reset() { setForm(empty()); setFormError('') }
  function edit(category: Category) { setForm({ ...category }); setFormError(''); document.getElementById('category-name')?.focus() }

  function save() {
    if (busy) return
    setFormError('')
    if (!form.name.trim()) { setFormError('请填写分类名称。'); return }
    if (!Number.isSafeInteger(form.sort) || form.sort < 0) { setFormError('排序值需要是非负整数。'); return }
    void run(() => categoryApi.save({ name: form.name.trim(), sort: form.sort, status: form.status }, form.id), () => {
      notify(form.id ? '分类已更新。' : '分类已创建。'); reset(); categories.reload()
    }, (cause) => setFormError(errorMessage(cause)))
  }

  function remove(category: Category) {
    if (busy || !window.confirm(`确认删除分类「${category.name}」？已有文章的分类需要另行检查。`)) return
    void run(() => categoryApi.remove(category.id), () => {
      if (form.id === category.id) reset()
      notify('分类已删除。')
      if (categories.data?.list.length === 1 && page > 1) setPage(page - 1)
      else categories.reload()
    }, (cause) => notify(errorMessage(cause), 'error'))
  }

  return <div className="container workspace-page"><WorkspaceNav identity="admin" /><div className="category-layout">
    <section className="workspace-panel">
      <div className="panel-heading"><div><h2>分类{categories.data && <span className="count-label">{categories.data.total} 个</span>}</h2><p>给不同方向的记录，安排一个位置。</p></div><UiIcon name="folder" size={24} /></div>
      {categories.pending ? <StatePanel kind="loading" />
        : categories.error ? <StatePanel kind="error" message={categories.error} onRetry={categories.reload} />
        : !categories.data?.list.length ? <StatePanel kind="empty" title="还没有分类" message="在分类表单中创建第一个分类。" />
        : <div className="table-wrap"><table className="data-table"><thead><tr><th>名称</th><th>排序</th><th>状态</th><th className="align-right">操作</th></tr></thead><tbody>
          {categories.data.list.map((category) => <tr key={category.id}>
            <td><strong>{category.name}</strong></td><td className="muted">{category.sort}</td><td><span className={`status-pill status-${category.status === 1 ? 1 : 2}`}>{category.status === 1 ? '启用' : '禁用'}</span></td>
            <td><div className="row-actions"><button className="link-button" disabled={busy} onClick={() => edit(category)}>编辑</button><button className="icon-button danger-text" disabled={busy} aria-label={`删除分类：${category.name}`} onClick={() => remove(category)}><UiIcon name="trash" size={17} /></button></div></td>
          </tr>)}
        </tbody></table></div>}
      {categories.data && <Pagination page={categories.data.pageNum} pages={categories.data.pages} busy={categories.pending || busy} onChange={setPage} />}
    </section>
    <section className="category-form-card"><p className="eyebrow">ORGANIZE YOUR NOTES</p><h2>{form.id ? '编辑分类' : '新建分类'}</h2>
      <form className="stack-form" onSubmit={(event) => { event.preventDefault(); save() }}>
        <label>分类名称<input id="category-name" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} disabled={busy} required maxLength={64} placeholder="例如：Go 学习笔记" /></label>
        <div className="form-columns"><label>排序<input value={Number.isNaN(form.sort) ? '' : form.sort} onChange={(event) => setForm({ ...form, sort: event.target.valueAsNumber })} disabled={busy} type="number" min={0} step={1} required /></label><label>状态<select value={form.status} onChange={(event) => setForm({ ...form, status: Number(event.target.value) as 0 | 1 })} disabled={busy}><option value={1}>启用</option><option value={0}>禁用</option></select></label></div>
        {formError && <p className="form-error" role="alert">{formError}</p>}
        <button className="button primary full" disabled={busy}>{busy ? '保存中…' : form.id ? '保存修改' : '创建分类'}</button>
        {form.id && <button className="button secondary full" disabled={busy} type="button" onClick={reset}>取消编辑</button>}
      </form><p className="small-help">启用的分类会出现在博客筛选和文章编辑器中。</p>
    </section>
  </div></div>
}
