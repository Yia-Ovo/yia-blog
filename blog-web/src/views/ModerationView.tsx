import { useCallback, useState } from 'react'
import { Link } from 'react-router-dom'
import { commentApi } from '../api'
import { ApiError, errorMessage } from '../api/http'
import type { ManagedComment } from '../api/types'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { commentStatuses, formatDate } from '../lib/article'
import { notify } from '../stores/notice'
import WorkspaceNav from '../components/WorkspaceNav'
import StatePanel from '../components/StatePanel'
import Pagination from '../components/Pagination'
import UiIcon from '../components/UiIcon'

export default function ModerationView() {
  const [status, setStatus] = useState('0')
  const [page, setPage] = useState(1)
  const [articleFilter, setArticleFilter] = useState('')
  const [articleID, setArticleID] = useState<number>()
  const [filterError, setFilterError] = useState('')
  const { busy, run } = useMutation(`${status}:${page}:${articleID || ''}`)
  const comments = useResource(useCallback((signal) => commentApi.managedList({ status: status === 'all' ? undefined : Number(status), page, size: 10, articleID, orderBy: 'create_at', orderDir: 'desc' }, signal), [status, page, articleID]))

  function filter() {
    if (busy) return
    setFilterError('')
    const value = articleFilter.trim()
    const id = Number(value)
    if (value && (!Number.isSafeInteger(id) || id <= 0)) { setFilterError('文章编号需要是正整数。'); return }
    setPage(1); setArticleID(value ? id : undefined)
  }

  function review(comment: ManagedComment, target: 1 | 2) {
    // 不修改快照、不自动重试；管理员必须重新阅读冲突后的正文。
    void run(() => commentApi.review({ id: comment.id, expected_content: comment.content, expected_status: comment.status, status: target }), () => {
      notify(target === 1 ? '评论已通过审核。' : '评论已拒绝，不再公开展示。')
      if (comments.data?.list.length === 1 && page > 1 && status !== 'all' && Number(status) !== target) setPage(page - 1)
      else comments.reload()
    }, (cause) => {
      if (cause instanceof ApiError && cause.code === 400) {
        notify('评论内容或状态已变化。已刷新列表，请重新查看后审核。', 'error')
        comments.reload()
      } else notify(errorMessage(cause), 'error')
    })
  }

  return <div className="container workspace-page"><WorkspaceNav identity="admin" /><section className="workspace-panel">
    <div className="panel-heading"><div><h2>评论审核{comments.data && <span className="count-label">{comments.data.total} 条</span>}</h2><p>仔细读过每一条留言，再决定是否公开。</p></div><button className="button secondary small" disabled={comments.pending || busy} onClick={comments.reload}>刷新列表</button></div>
    <div className="table-toolbar">
      <div className="filter-row">{[...commentStatuses, '全部'].map((label, index) => {
        const value = index === 3 ? 'all' : String(index)
        return <button key={value} disabled={busy} className={status === value ? 'selected' : ''} aria-pressed={status === value} onClick={() => { setPage(1); setStatus(value) }}>{label}</button>
      })}</div>
      <form className="search-box compact" onSubmit={(event) => { event.preventDefault(); filter() }}><UiIcon name="search" size={17} /><label className="sr-only" htmlFor="comment-article-filter">按文章编号筛选</label><input id="comment-article-filter" value={articleFilter} onChange={(event) => setArticleFilter(event.target.value)} disabled={busy} inputMode="numeric" placeholder="按文章编号筛选" /><button disabled={busy}>筛选</button></form>
    </div>
    {filterError && <p className="form-error" role="alert">{filterError}</p>}
    {comments.pending ? <StatePanel kind="loading" />
      : comments.error ? <StatePanel kind="error" message={comments.error} onRetry={comments.reload} />
      : !comments.data?.list.length ? <StatePanel kind="empty" title="当前筛选下没有评论" message="新的留言会出现在待审核列表中。" />
      : <div className="moderation-list">{comments.data.list.map((comment) => <article key={comment.id} className="moderation-card">
        <div className="moderation-top"><span className="profile-monogram small-monogram">{comment.nickname.slice(0, 1) || 'Y'}</span><div><strong>{comment.nickname}</strong><p>{formatDate(comment.create_at)}{!!comment.parent_id && <span> · 回复评论</span>}</p></div><span className={`status-pill status-${comment.status}`}>{commentStatuses[comment.status]}</span></div>
        <blockquote>{comment.content}</blockquote>
        <div className="moderation-bottom"><Link to={`/admin/articles/${comment.article_id}/edit`} className="subtle-link">查看所属文章 <UiIcon name="arrow" size={15} /></Link><div className="row-actions">
          {comment.status !== 2 && <button className="button secondary small" disabled={busy} onClick={() => review(comment, 2)}>{comment.status === 1 ? '撤下评论' : '拒绝'}</button>}
          {comment.status !== 1 && <button className="button primary small" disabled={busy} onClick={() => review(comment, 1)}><UiIcon name="check" size={16} />通过审核</button>}
        </div></div>
      </article>)}</div>}
    {comments.data && <Pagination page={comments.data.pageNum} pages={comments.data.pages} busy={comments.pending || busy} onChange={setPage} />}
  </section></div>
}
