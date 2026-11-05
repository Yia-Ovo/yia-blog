import NovelReadingMenu from '../components/NovelReadingMenu'
import { findChapter, rememberChapter } from '../lib/novel'
import { useCallback, useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { articleApi, categoryApi, commentApi } from '../api'
import { errorMessage } from '../api/http'
import type { ArticleDetail, Comment } from '../api/types'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { useSessions } from '../hooks/useSessions'
import { useI18n } from '../i18n'
import { formatDate, positiveNumber } from '../lib/article'
import { safeImageURL } from '../lib/imageURL'
import { notify } from '../stores/notice'
import MarkdownBody from '../components/MarkdownBody'
import Pagination from '../components/Pagination'
import StatePanel from '../components/StatePanel'
import UiIcon from '../components/UiIcon'
import Avatar from '../components/Avatar'
import NotFoundView from './NotFoundView'

export default function ArticleView() {
  const { id: parameter } = useParams()
  const id = positiveNumber(parameter, 0)
  if (!id || !/^\d+$/.test(parameter || '')) return <NotFoundView />
  return <ArticlePage key={id} id={id} />
}

function ArticlePage({ id }: { id: number }) {
  const { locale, t } = useI18n()
  const location = useLocation()
  const sessions = useSessions()
  const article = useResource(useCallback((signal) => articleApi.detail(id, signal), [id]))
  const categories = useResource(categoryApi.all)
  const value = article.data
  const chapter = value && findChapter(value)
  useEffect(() => { if (value) rememberChapter(value) }, [value])
  const categoryName = categories.data?.find((item) => item.id === value?.category_id)?.name || t('文章')
  useEffect(() => {
    if (value) document.title = `${value.title} · Yia`
  }, [value, locale])
  useEffect(() => {
    if (value && location.hash === '#comments') document.getElementById('comments')?.scrollIntoView()
  }, [value, location.hash])

  return <div className={`container reading-page${chapter ? ' reading-page--novel' : ''}`}>
    <Link className="back-link" to={chapter ? '/novels' : '/'}><UiIcon name="back" size={17} />{t(chapter ? '章节目录' : '所有文章')}</Link>
    {article.pending ? <StatePanel kind="loading" />
      : article.error ? <StatePanel kind="error" message={t(article.error)} onRetry={article.reload} />
      : value && <>
        <article className="reading-article">
          <header className="reading-header">
            <div className="article-meta"><Link className="category-tag" to={value.category_id ? `/?category=${value.category_id}` : '/'}>{categoryName}</Link><span>{formatDate(value.publish_at || value.create_at)}</span></div>
            <h1 id="reading-title" tabIndex={-1}>{value.title}</h1>{value.summary && <p className="reading-summary">{value.summary}</p>}
            <div className="author-line"><Avatar url={value.author?.avatar} name={value.author?.name} /><div><strong>{value.author?.name || t('作者')}</strong><p>{value.author?.signature || t('记录问题，也记录答案。')}</p></div></div>
          </header>
          {safeImageURL(value.cover) && <img className="reading-cover" src={safeImageURL(value.cover)} alt={t('文章封面')} referrerPolicy="no-referrer" />}
          <MarkdownBody source={value.content || ''} />
          <div className="article-end"><span /><UiIcon name="book" size={18} /><span /></div>
        </article>
        {chapter && <><Link className="button secondary" to="/novels">{t('返回章节目录')}</Link><NovelReadingMenu key={id} chapterID={id} /></>}
        <ArticleComments key={`${id}:${sessions.user?.token || ''}`} article={value} />
      </>}
  </div>
}

function ArticleComments({ article }: { article: ArticleDetail }) {
  const { t } = useI18n()
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
    <div className="section-heading"><div><p className="eyebrow">LET'S TALK</p><h2 id="comments-title">{t('交流与讨论')}{comments.data && <span className="count-label">{t('{count} 条', { count: comments.data.total })}</span>}</h2></div><button className="link-button" disabled={comments.pending} onClick={comments.reload}>{t('刷新评论')}</button></div>
    {article.comment_status !== 1 ? <div className="inline-note">{t('这篇文章暂时关闭了评论。')}</div>
      : sessions.user ? <form className="comment-form" onSubmit={(event) => { event.preventDefault(); submitComment() }}>
        <div className="composer-heading"><span>{t('以 {name} 的身份发表评论', { name: sessions.user.profile.nick_name || t('用户') })}</span><small>{content.length} / 1000</small></div>
        {reply && <div className="reply-indicator">{t('回复 {name}', { name: reply.nickname })}<button type="button" className="link-button" disabled={sending} onClick={() => setReply(null)}>{t('取消回复')}</button></div>}
        <label className="sr-only" htmlFor="comment-content">{t('评论内容')}</label><textarea id="comment-content" value={content} onChange={(event) => setContent(event.target.value)} disabled={sending} rows={4} maxLength={1000} required placeholder={t('分享你的想法，或提出一个问题…')} />
        {formError && <p className="form-error" role="alert">{t(formError)}</p>}
        <div className="composer-footer"><small>{t('评论审核通过后公开展示。')}</small><button className="button primary small" disabled={sending || !content.trim()}>{t(sending ? '提交中…' : '提交评论')}<UiIcon name="arrow" size={16} /></button></div>
      </form>
      : <div className="login-to-comment"><UiIcon name="comment" /><span>{t('登录后，加入这次讨论。')}</span><Link className="button secondary small" to={`/login?next=${encodeURIComponent(location.pathname + location.search + '#comments')}`}>{t('去登录')}</Link></div>}
    {comments.pending ? <StatePanel kind="loading" title={t('正在加载评论…')} />
      : comments.error ? <StatePanel kind="error" message={t(comments.error)} onRetry={comments.reload} />
      : !comments.data?.list.length ? <StatePanel kind="empty" title={t('还没有公开评论')} message={t('有想法就留下来，交流从第一句话开始。')} />
      : <div className="comment-list">{comments.data.list.map((comment) => <article key={comment.id} className="comment-item">
        <Avatar url={comment.avatar} name={comment.nickname} />
        <div className="comment-main">
          <div className="comment-byline"><strong>{comment.nickname}</strong><time>{formatDate(comment.create_at)}</time></div>
          {!!comment.parent_id && <small className="muted">{t('回复了一条评论')}</small>}<p className="comment-content">{comment.content}</p>
          <div className="comment-actions">
            <button disabled={!sessions.user || reacting} onClick={() => reactTo(comment.id)}><UiIcon name="heart" size={15} />{t('赞同 {count}', { count: comment.likes })}</button>
            {sessions.user && <button disabled={reacting} onClick={() => reactTo(comment.id, true)}>{t('撤回我的赞')}</button>}
            {sessions.user && article.comment_status === 1 && <button disabled={sending} onClick={() => replyTo(comment)}>{t('回复')}</button>}
          </div>
        </div>
      </article>)}</div>}
    <Pagination page={page} pages={pages} busy={comments.pending || reacting} onChange={setPage} />
  </section>
}
