import { useCallback, useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { articleApi } from '../api'
import { errorMessage } from '../api/http'
import type { Identity, ManagedArticle } from '../api/types'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { articleStatuses, formatDate, positiveNumber } from '../lib/article'
import { notify } from '../stores/notice'
import WorkspaceNav from '../components/WorkspaceNav'
import StatePanel from '../components/StatePanel'
import Pagination from '../components/Pagination'
import UiIcon from '../components/UiIcon'

export default function ArticlesView({ identity }: { identity: Identity }) {
  const [params] = useSearchParams()
  const location = useLocation()
  const navigate = useNavigate()
  const title = params.get('q') || ''
  const [search, setSearch] = useState(title)
  const status = params.get('status')
  const selected = ['0', '1', '2'].includes(status || '') ? Number(status) : undefined
  const page = positiveNumber(params.get('page'))
  const base = identity === 'admin' ? '/admin/articles' : '/me/articles'
  const newPath = identity === 'admin' ? '/admin/articles/new' : '/write/new'
  const editPath = (id: number) => identity === 'admin' ? `/admin/articles/${id}/edit` : `/write/${id}`
  useEffect(() => { setSearch(title) }, [title])
  const articles = useResource(useCallback((signal) => articleApi.managedList(identity, { page_num: page, page_size: 10, status: selected, title }, signal), [identity, page, selected, title]))
  const { busy, run } = useMutation(location.key)

  function go(nextPage = 1, nextStatus: number | null | undefined = selected) {
    const query = new URLSearchParams()
    if (nextPage > 1) query.set('page', String(nextPage))
    if (nextStatus !== null && nextStatus !== undefined) query.set('status', String(nextStatus))
    if (search.trim()) query.set('q', search.trim())
    void navigate({ pathname: base, search: query.toString() })
  }

  function remove(article: ManagedArticle) {
    if (busy || !window.confirm(`确认删除《${article.title}》？删除后将不再展示。`)) return
    void run(() => articleApi.remove(identity, article.id), () => {
      notify('文章已删除。')
      if (articles.data?.list.length === 1 && page > 1) go(page - 1)
      else articles.reload()
    }, (cause) => notify(errorMessage(cause), 'error'))
  }

  return <div className="container workspace-page"><WorkspaceNav identity={identity} />
    <section className="workspace-panel">
      <div className="panel-heading"><div><h2>{identity === 'admin' ? '所有文章' : '我的文章'}{articles.data && <span className="count-label">{articles.data.total} 篇</span>}</h2><p>在这里继续草稿，或管理已经发布的记录。</p></div><Link className="button primary small" to={newPath}><UiIcon name="plus" size={18} />新建文章</Link></div>
      <div className="table-toolbar">
        <div className="filter-row"><button disabled={busy} className={selected === undefined ? 'selected' : ''} aria-pressed={selected === undefined} onClick={() => go(1, null)}>全部</button>{articleStatuses.map((label, value) => <button key={value} disabled={busy} className={selected === value ? 'selected' : ''} aria-pressed={selected === value} onClick={() => go(1, value)}>{label}</button>)}</div>
        <form className="search-box compact" onSubmit={(event) => { event.preventDefault(); if (!busy) go() }}><UiIcon name="search" size={17} /><label className="sr-only" htmlFor="managed-search">搜索管理文章标题</label><input id="managed-search" value={search} onChange={(event) => setSearch(event.target.value)} disabled={busy} type="search" placeholder="搜索标题…" maxLength={128} /><button disabled={busy}>搜索</button></form>
      </div>
      {articles.pending ? <StatePanel kind="loading" />
        : articles.error ? <StatePanel kind="error" message={articles.error} onRetry={articles.reload} />
        : !articles.data?.list.length ? <StatePanel kind="empty" title="这里还没有文章" message="可以换个筛选条件，也可以从一篇新草稿开始。"><Link className="button secondary small" to={newPath}>写一篇文章</Link></StatePanel>
        : <div className="table-wrap"><table className="data-table"><thead><tr><th>文章</th><th>状态</th><th className="date-column">最后更新</th><th className="align-right">操作</th></tr></thead><tbody>
          {articles.data.list.map((article) => <tr key={article.id}>
            <td><Link className="table-title" to={editPath(article.id)}>{article.title}</Link><p className="table-excerpt">{article.summary || '暂未填写摘要'}</p></td>
            <td><span className={`status-pill status-${article.status}`}>{articleStatuses[article.status]}</span></td><td className="date-column muted">{formatDate(article.update_at)}</td>
            <td><div className="row-actions"><Link to={editPath(article.id)} className="link-button">编辑</Link>{article.status === 1 && <Link to={`/articles/${article.id}`} className="icon-button" aria-label={`阅读文章：${article.title}`}><UiIcon name="eye" size={17} /></Link>}<button className="icon-button danger-text" disabled={busy} aria-label={`删除文章：${article.title}`} onClick={() => remove(article)}><UiIcon name="trash" size={17} /></button></div></td>
          </tr>)}
        </tbody></table></div>}
      {articles.data && <Pagination page={articles.data.pageNum} pages={articles.data.pages} busy={articles.pending || busy} onChange={(next) => go(next)} />}
    </section>
  </div>
}
