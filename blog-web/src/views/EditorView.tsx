import { useCallback, useContext, useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useBlocker, useNavigate, useParams } from 'react-router-dom'
import type { BlockerFunction } from 'react-router-dom'
import { articleApi, categoryApi } from '../api'
import { errorMessage } from '../api/http'
import type { ArticleInput, ArticleStatus, Identity } from '../api/types'
import { useMutation } from '../hooks/useMutation'
import { useResource } from '../hooks/useResource'
import { useSessions } from '../hooks/useSessions'
import { useEditorDraft } from '../hooks/useEditorDraft'
import { useArticleImageUpload } from '../hooks/useArticleImageUpload'
import { getSessions } from '../stores/session'
import { insertArticleImage } from '../lib/articleImages'
import DraftRecoveryPanel from '../components/DraftRecoveryPanel'
import { articleInput, emptyArticle, positiveNumber, validateArticle } from '../lib/article'
import { safeImageURL } from '../lib/markdown'
import { notify } from '../stores/notice'
import { LeaveGuardContext } from '../components/LeaveGuardContext'
import MarkdownBody from '../components/MarkdownBody'
import StatePanel from '../components/StatePanel'
import UiIcon from '../components/UiIcon'
import NotFoundView from './NotFoundView'

export default function EditorView({ identity }: { identity: Identity }) {
  const { id: parameter } = useParams()
  const accountId = useSessions()[identity]?.profile.id
  const openedAccount = useRef(accountId)
  if (accountId) openedAccount.current = accountId
  const ownerId = accountId || openedAccount.current
  const id = positiveNumber(parameter, 0)
  if (parameter !== undefined && (!id || !/^\d+$/.test(parameter))) return <NotFoundView />
  if (!ownerId) return <StatePanel kind="error" title="请先登录后再编辑文章" />
  // 登录过期时仍归属于原账号；切换到其他账号则重建编辑器，避免串稿。
  return <EditorLoader key={`${identity}:${ownerId}:${id}`} identity={identity} accountId={ownerId} id={id} />
}

function EditorLoader({ identity, accountId, id }: { identity: Identity; accountId: number; id: number }) {
  const article = useResource(useCallback(async (signal) => id ? articleInput(await articleApi.managedDetail(identity, id, signal)) : emptyArticle(), [identity, id]))
  if (article.data) return <ArticleEditor identity={identity} accountId={accountId} id={id} initial={article.data} />
  return <div className="container editor-page">
    <Link className="back-link" to={identity === 'admin' ? '/admin/articles' : '/me/articles'}><UiIcon name="back" size={17} />返回文章列表</Link>
    {article.pending ? <StatePanel kind="loading" title="正在打开编辑器…" /> : <StatePanel kind="error" message={article.error} onRetry={article.reload} />}
  </div>
}

function ArticleEditor({ identity, accountId, id, initial }: { identity: Identity; accountId: number; id: number; initial: ArticleInput }) {
  const navigate = useNavigate()
  const sessions = useSessions()
  const listPath = identity === 'admin' ? '/admin/articles' : '/me/articles'
  const [form, setForm] = useState(initial)
  const [original, setOriginal] = useState(() => JSON.stringify(initial))
  const [formError, setFormError] = useState('')
  const [preview, setPreview] = useState(false)
  const [draftOwner] = useState(() => ({ identity, accountId, articleId: id }))
  const draft = useEditorDraft({ owner: draftOwner, initial, value: form, onRestore: setForm })
  const draftPending = draft.canDiscard
  const images = useArticleImageUpload(identity)
  const contentInput = useRef<HTMLTextAreaElement>(null)
  const bodyImageInput = useRef<HTMLInputElement>(null)
  const coverImageInput = useRef<HTMLInputElement>(null)
  const { busy: saving, run } = useMutation()
  const saveInFlight = useRef(false)
  const savedForExit = useRef(false)
  const dirty = original !== JSON.stringify(articleInput(form))
  const categories = useResource(categoryApi.all)
  const missingCategory = form.category_id > 0 && !categories.data?.some((item) => item.id === form.category_id)
  const registerLeaveGuard = useContext(LeaveGuardContext)

  const canLeave = useCallback(() => {
    if (savedForExit.current) return true
    if (saveInFlight.current || images.isUploading()) return false
    return !dirty || window.confirm('文章还有未保存的修改，确认离开编辑器？')
  }, [dirty, images.uploading])
  const blocker = useBlocker(useCallback<BlockerFunction>(({ currentLocation, nextLocation }) =>
    !savedForExit.current && (saveInFlight.current || images.isUploading() || dirty) &&
    (currentLocation.pathname !== nextLocation.pathname || currentLocation.search !== nextLocation.search), [dirty, images.uploading]))

  useEffect(() => {
    if (blocker.state !== 'blocked') return
    if (canLeave()) blocker.proceed()
    else blocker.reset()
  }, [blocker, canLeave])
  useEffect(() => registerLeaveGuard(canLeave), [registerLeaveGuard, canLeave])
  useEffect(() => {
    const beforeUnload = (event: BeforeUnloadEvent) => {
      if (!savedForExit.current && (dirty || saveInFlight.current || images.isUploading())) { event.preventDefault(); event.returnValue = '' }
    }
    window.addEventListener('beforeunload', beforeUnload)
    return () => window.removeEventListener('beforeunload', beforeUnload)
  }, [dirty, images.uploading])

  function setField<K extends keyof ArticleInput>(key: K, value: ArticleInput[K]) {
    setForm((current) => ({ ...current, [key]: value }))
  }

  function uploadBody(files: File[]) {
    if (saveInFlight.current || saving || draftPending) return
    const selection = { content: form.content, position: contentInput.current?.selectionEnd ?? form.content.length }
    void images.upload(files, (image, file) => {
      setForm((current) => ({ ...current, content: insertArticleImage(current.content, image, file.name, selection) }))
      setPreview(false)
    })
  }

  function uploadCover(files: File[]) {
    if (saveInFlight.current || saving || draftPending) return
    void images.upload(files, (image) => setForm((current) => ({ ...current, cover: image.url })))
  }

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saveInFlight.current || images.isUploading()) return
    if (draftPending) { setFormError('请先恢复或放弃已有的本机暂存。'); return }
    if (!sessions[identity]) { setFormError('请保留当前内容后重新登录，再继续保存。'); return }
    const error = validateArticle(form)
    if (error) { setFormError(error); return }
    if (form.cover && !safeImageURL(form.cover)) { setFormError('封面需要使用有效的 HTTP 或 HTTPS 图片地址。'); return }
    setFormError('')
    const payload = articleInput(form)
    payload.title = payload.title.trim()
    if (payload.status === 1 && !payload.publish_at) payload.publish_at = new Date().toISOString()
    const saveToken = sessions[identity]?.token
    draft.flush()
    saveInFlight.current = true
    void run(() => articleApi.save(identity, payload, id || undefined), () => {
      if (getSessions()[identity]?.token !== saveToken) {
        saveInFlight.current = false
        setFormError('登录状态已变化，请核对文章后重新保存。')
        return
      }
      const cleared = draft.clearAfterSave()
      setForm(payload); setOriginal(JSON.stringify(payload))
      // 先标记成功，避免同一轮 React 更新中的路由检查读到保存前的 dirty 状态。
      savedForExit.current = true; saveInFlight.current = false
      notify((payload.status === 0 ? '草稿已保存。' : payload.status === 1 ? '文章已保存并发布。' : '文章已保存为下架状态。') + (cleared ? '' : '本机暂存尚未清理，下次打开请核对。'))
      void navigate(listPath)
    }, (cause) => { saveInFlight.current = false; setFormError(errorMessage(cause)) })
  }

  return <div className="container editor-page">
    <div className="editor-topbar"><Link className="back-link" to={listPath}><UiIcon name="back" size={17} />返回文章列表</Link><span className="editor-save-state"><span className={dirty ? 'unsaved-dot' : 'tiny-dot'} />{saving ? '正在保存…' : dirty ? '有未保存的修改' : id ? '已加载文章' : '一篇新的草稿'}</span></div>
    <form className="editor-form" onSubmit={save}><fieldset disabled={saving}>
      <div className="editor-heading"><div><p className="eyebrow">MAKE ROOM FOR AN IDEA</p><h1>{id ? '继续这篇记录' : '写下新的想法'}<span className="accent-dot">.</span></h1></div><button className="button primary" type="submit" disabled={!sessions[identity] || images.uploading || draftPending}>{saving ? '保存中…' : form.status === 1 ? '保存并发布' : form.status === 2 ? '保存并下架' : '保存草稿'}<UiIcon name="check" size={18} /></button></div>
      {formError && <p className="form-error" role="alert">{formError}</p>}
      <DraftRecoveryPanel draft={draft} />
      <div className="editor-layout">
        <section className="editor-paper">
          <label className="sr-only" htmlFor="article-title">文章标题</label><input id="article-title" value={form.title} onChange={(event) => setField('title', event.target.value)} className="title-input" required maxLength={128} placeholder="给这篇记录起个名字…" />
          <label className="summary-label" htmlFor="article-summary">摘要 <span>· 可选，帮助读者快速了解内容</span></label><textarea id="article-summary" value={form.summary} onChange={(event) => setField('summary', event.target.value)} className="summary-input" rows={2} maxLength={512} placeholder="用一两句话介绍这篇文章。" />
          <div className="editor-mode-tabs"><button type="button" className={!preview ? 'active' : ''} aria-pressed={!preview} onClick={() => setPreview(false)}><UiIcon name="edit" size={16} />编辑</button><button type="button" className={preview ? 'active' : ''} aria-pressed={preview} onClick={() => setPreview(true)}><UiIcon name="eye" size={17} />预览</button><span>Markdown</span></div>
          <div className="article-image-toolbar">
            <button type="button" className="text-link link-button" disabled={images.uploading || draftPending || !sessions[identity]} onClick={() => bodyImageInput.current?.click()}><UiIcon name="plus" size={16} />插入图片</button>
            <input ref={bodyImageInput} className="sr-only" tabIndex={-1} aria-label="选择正文图片" type="file" accept="image/jpeg,image/png,.jpg,.jpeg,.png" disabled={images.uploading || draftPending || !sessions[identity]} onChange={(event) => { uploadBody(Array.from(event.currentTarget.files || [])); event.currentTarget.value = '' }} />
            <span>{images.uploading ? '图片正在上传…' : '可粘贴或拖入 JPG / PNG，最大 5 MB'}</span>
          </div>
          {images.error && <p className="form-error image-upload-error" role="alert">{images.error}</p>}
          {preview ? <div className="editor-preview">{form.content.trim() ? <MarkdownBody source={form.content} /> : <p className="muted">写下正文后，就可以在这里查看排版。</p>}</div>
            : <><label className="sr-only" htmlFor="article-content">文章正文（Markdown）</label><textarea ref={contentInput} id="article-content" onPaste={(event) => {
              const files = Array.from(event.clipboardData.files)
              if (files.length) { event.preventDefault(); uploadBody(files) }
            }} onDragOver={(event) => { if (event.dataTransfer.types.includes('Files')) event.preventDefault() }}
            onDrop={(event) => {
              const files = Array.from(event.dataTransfer.files)
              if (files.length) { event.preventDefault(); uploadBody(files) }
            }} value={form.content} onChange={(event) => setField('content', event.target.value)} className="content-input" rows={23} placeholder={'从一个问题、一段经历，或一行代码开始。\n\n支持 Markdown，例如：\n## 一个小标题\n\n写下你的正文…'} spellCheck={false} /></>}
          <div className="editor-paper-footer"><span>内容由你保存后写入博客。</span><span>{form.content.length} 字符</span></div>
        </section>
        <aside className="editor-settings">
          <section className="settings-card"><h2>文章设置</h2>
            <label>发布状态<select value={form.status} onChange={(event) => setField('status', Number(event.target.value) as ArticleStatus)}><option value={0}>草稿 · 仅作者和管理员可见</option><option value={1}>发布 · 所有人可见</option><option value={2}>下架 · 暂不公开</option></select></label>
            <label>分类<select value={form.category_id} onChange={(event) => setField('category_id', Number(event.target.value))}><option value={0}>未分类</option>{missingCategory && <option value={form.category_id}>保留原分类（未启用或已移除）</option>}{categories.data?.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}</select></label>
            {categories.error && <p className="inline-note">分类加载失败。<button type="button" className="link-button" onClick={categories.reload}>重试</button></p>}
            <label className="checkbox-label"><input checked={form.comment_status === 1} onChange={(event) => setField('comment_status', event.target.checked ? 1 : 0)} type="checkbox" /><span>允许读者发表评论</span></label>
            <label className="checkbox-label"><input checked={form.is_top === 1} onChange={(event) => setField('is_top', event.target.checked ? 1 : 0)} type="checkbox" /><span>置顶这篇文章</span></label>
            <label>排序<input value={Number.isNaN(form.sort) ? '' : form.sort} onChange={(event) => setField('sort', event.target.valueAsNumber)} type="number" min={0} step={1} required /></label>
          </section>
          <section className="settings-card"><h2>封面图片</h2><button type="button" className="button secondary small cover-upload-button" disabled={images.uploading || draftPending || !sessions[identity]} onClick={() => coverImageInput.current?.click()}><UiIcon name="plus" size={16} />上传封面</button><input ref={coverImageInput} className="sr-only" tabIndex={-1} type="file" aria-label="选择封面图片" accept="image/jpeg,image/png,.jpg,.jpeg,.png" disabled={images.uploading || draftPending || !sessions[identity]} onChange={(event) => { uploadCover(Array.from(event.currentTarget.files || [])); event.currentTarget.value = '' }} /><label className="sr-only" htmlFor="article-cover">封面图片地址</label><input id="article-cover" disabled={images.uploading} value={form.cover} onChange={(event) => setField('cover', event.target.value)} placeholder="粘贴图片地址（可选）" />{safeImageURL(form.cover) && <img className="cover-preview" src={safeImageURL(form.cover)} alt="封面预览" referrerPolicy="no-referrer" />}<p className="small-help">JPG / PNG，最大 5 MB、400 万像素。也可粘贴图片地址；留空使用默认封面。上传图片可通过链接访问。</p></section>
          <div className="editor-tip"><UiIcon name="book" size={20} /><p>不必一次写完。<br />先保存草稿，想清楚了再发布。</p></div>
        </aside>
      </div>
    </fieldset></form>
  </div>
}
