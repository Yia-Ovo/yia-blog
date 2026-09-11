import { useCallback, useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { articleApi, categoryApi, commentApi } from '../api'
import { errorMessage } from '../api/http'
import type { ArticleDetail, Comment } from '../api/types'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { useSessions } from '../hooks/useSessions'
import { formatDate, positiveNumber } from '../lib/article'
import { safeImageURL } from '../lib/markdown'
import { notify } from '../stores/notice'
import MarkdownBody from '../components/MarkdownBody'
import Pagination from '../components/Pagination'
import StatePanel from '../components/StatePanel'
import UiIcon from '../components/UiIcon'
import NotFoundView from './NotFoundView'

export default function ArticleView() {
  const { id: parameter } = useParams()
  const id = positiveNumber(parameter, 0)
  if (!id || !/^\d+$/.test(parameter || '')) return <NotFoundView />
  return <ArticlePage key={id} id={id} />
}

function ArticlePage({ id }: { id: number }) {
  const location = useLocation()
  const sessions = useSessions()
  const article = useResource(useCallback((signal) => articleApi.detail(id, signal), [id]))
  const categories = useResource(categoryApi.all)
  const value = article.data
  const categoryName = categories.data?.find((item) => item.id === value?.category_id)?.name || '文章'
  useEffect(() => {
    if (!value) return
    document.title = `${value.title} · Yia`
    if (location.hash === '#comments') document.getElementById('comments')?.scrollIntoView()
  }, [value, location.hash])

  return <div className="container reading-page">
    <Link className="back-link" to="/"><UiIcon name="back" size={17} />所有文章</Link>
    {article.pending ? <StatePanel kind="loading" />
      : article.error ? <StatePanel kind="error" message={article.error} onRetry={article.reload} />
      : value && <>
        <article className="reading-article">
          <header className="reading-header">
            <div className="article-meta"><Link className="category-tag" to={value.category_id ? `/?category=${value.category_id}` : '/'}>{categoryName}</Link><span>{formatDate(value.publish_at || value.create_at)}</span></div>
            <h1>{value.title}</h1>{value.summary && <p className="reading-summary">{value.summary}</p>}
            <div className="author-line"><Avatar url={value.author?.avatar} name={value.author?.name} /><div><strong>{value.author?.name || '作者'}</strong><p>{value.author?.signature || '记录问题，也记录答案。'}</p></div></div>
          </header>
          {safeImageURL(value.cover) && <img className="reading-cover" src={safeImageURL(value.cover)} alt="文章封面" referrerPolicy="no-referrer" />}
          <MarkdownBody source={value.content || ''} />
          <div className="article-end"><span /><UiIcon name="book" size={18} /><span /></div>
        </article>
        <ArticleComments key={`${id}:${sessions.user?.token || ''}`} article={value} />
      </>}
  </div>
}

function Avatar({ url, name }: { url?: string; name?: string }) {
  return <span className="avatar">{safeImageURL(url) ? <img src={safeImageURL(url)} alt="" loading="lazy" referrerPolicy="no-referrer" /> : <span>{(name || 'Y').slice(0, 1)}</span>}</span>
}

function ArticleComments({ article }: { article: ArticleDetail }) {
  const sessions = useSessions()
  const location = useLocation()
  const [page, setPage] = useState(1)
  const comments = useResource(useCallback((signal) => commentApi.list(article.id, page, 10, signal), [article.id, page]))
  const [content, setContent] = useState('')
  const [reply, setReply] = useState<Pick<Comment, 'id' | 'nickname'> | null>(null)
  const [formError, setFormError] = useState('')
  const { busy: sending, run: send } = useMutation()
  const { busy: reacting, run: react } = useMutation()
  const pages = Math.ceil((comments.data?.total || 0) / 10)

  function submitComment() {
    if (sending || !content.trim()) return
    setFormError('')
    void send(() => commentApi.create(article.id, content.trim(), reply?.id || 0), () => {
      setContent(''); setReply(null)
      notify('评论已提交，审核通过后会出现在这里。')
      comments.reload()
    }, (cause) => setFormError(errorMessage(cause)))
  }

  function reactTo(commentID: number, cancel = false) {
    void react(() => commentApi.like(commentID, cancel), () => {
      notify(cancel ? '已撤回你的点赞。' : '已赞同这条评论。')
      comments.reload()
    }, (cause) => notify(errorMessage(cause), 'error'))
  }

  function replyTo(comment: Comment) {
    setReply({ id: comment.id, nickname: comment.nickname })
    document.getElementById('comment-content')?.focus()
  }

  return <section id="comments" className="comments-section" aria-labelledby="comments-title">
    <div className="section-heading"><div><p className="eyebrow">LET'S TALK</p><h2 id="comments-title">交流与讨论{comments.data && <span className="count-label">{comments.data.total} 条</span>}</h2></div><button className="link-button" disabled={comments.pending} onClick={comments.reload}>刷新评论</button></div>
    {article.comment_status !== 1 ? <div className="inline-note">这篇文章暂时关闭了评论。</div>
      : sessions.user ? <form className="comment-form" onSubmit={(event) => { event.preventDefault(); submitComment() }}>
        <div className="composer-heading"><span>以 <strong>{sessions.user.profile.nick_name}</strong> 的身份发表评论</span><small>{content.length} / 1000</small></div>
        {reply && <div className="reply-indicator">回复 {reply.nickname}<button type="button" className="link-button" disabled={sending} onClick={() => setReply(null)}>取消回复</button></div>}
        <label className="sr-only" htmlFor="comment-content">评论内容</label><textarea id="comment-content" value={content} onChange={(event) => setContent(event.target.value)} disabled={sending} rows={4} maxLength={1000} required placeholder="分享你的想法，或提出一个问题…" />
        {formError && <p className="form-error" role="alert">{formError}</p>}
        <div className="composer-footer"><small>评论审核通过后公开展示。</small><button className="button primary small" disabled={sending || !content.trim()}>{sending ? '提交中…' : '提交评论'}<UiIcon name="arrow" size={16} /></button></div>
      </form>
      : <div className="login-to-comment"><UiIcon name="comment" /><span>登录后，加入这次讨论。</span><Link className="button secondary small" to={`/login?next=${encodeURIComponent(location.pathname + location.search + '#comments')}`}>去登录</Link></div>}
    {comments.pending ? <StatePanel kind="loading" title="正在加载评论…" />
      : comments.error ? <StatePanel kind="error" message={comments.error} onRetry={comments.reload} />
      : !comments.data?.list.length ? <StatePanel kind="empty" title="还没有公开评论" message="有想法就留下来，交流从第一句话开始。" />
      : <div className="comment-list">{comments.data.list.map((comment) => <article key={comment.id} className="comment-item">
        <Avatar url={comment.avatar} name={comment.nickname} />
        <div className="comment-main">
          <div className="comment-byline"><strong>{comment.nickname}</strong><time>{formatDate(comment.create_at)}</time></div>
          {!!comment.parent_id && <small className="muted">回复了一条评论</small>}<p className="comment-content">{comment.content}</p>
          <div className="comment-actions">
            <button disabled={!sessions.user || reacting} onClick={() => reactTo(comment.id)}><UiIcon name="heart" size={15} />赞同 {comment.likes}</button>
            {sessions.user && <button disabled={reacting} onClick={() => reactTo(comment.id, true)}>撤回我的赞</button>}
            {sessions.user && article.comment_status === 1 && <button disabled={sending} onClick={() => replyTo(comment)}>回复</button>}
          </div>
        </div>
      </article>)}</div>}
    <Pagination page={page} pages={pages} busy={comments.pending || reacting} onChange={setPage} />
  </section>
}
