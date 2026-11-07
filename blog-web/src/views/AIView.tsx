import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { aiApi } from '../api/ai'
import type { AIContextMode, AIConversation, AIImportedThread, AIOverview, AISyncImport } from '../api/ai'
import { errorMessage } from '../api/http'
import UiIcon from '../components/UiIcon'
import StatePanel from '../components/StatePanel'
import AICodePanel from './AICodePanel'
import { useSessions } from '../hooks/useSessions'
import { useI18n } from '../i18n'
import { renderMarkdown } from '../lib/markdown'
import './ai.css'

function renderAnswer(source: string) {
  const template = document.createElement('template')
  template.innerHTML = renderMarkdown(source)
  // 模型生成的图片链接不能自动向外部站点发送含私有上下文的请求。
  template.content.querySelectorAll('img').forEach(image => image.remove())
  return template.innerHTML
}

export default function AIView() {
  const { admin } = useSessions()
  return admin ? <AIWorkspace key={admin.token} /> : null
}

function AIWorkspace() {
  const { t } = useI18n()
  const [overview, setOverview] = useState<AIOverview | null>(null)
  const [conversation, setConversation] = useState<AIConversation | null>(null)
  const [selected, setSelected] = useState('')
  const [input, setInput] = useState('')
  const [contextMode, setContextMode] = useState<AIContextMode>('general')
  const [memory, setMemory] = useState('')
  const [memorySaved, setMemorySaved] = useState(false)
  const [memoryBusy, setMemoryBusy] = useState(false)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const [loadError, setLoadError] = useState('')
  const [loading, setLoading] = useState(true)
  const [imports, setImports] = useState<AIImportedThread[]>([])
  const [importSelected, setImportSelected] = useState('')
  const [importHistory, setImportHistory] = useState<AIImportedThread | null>(null)
  const [originImportID, setOriginImportID] = useState('')
  const [importBusy, setImportBusy] = useState(false)
  const [importError, setImportError] = useState('')
  const [showLatest, setShowLatest] = useState(false)
  const followLatest = useRef(true)
  const olderScroll = useRef<{ height: number; top: number } | null>(null)
  const importFile = useRef<HTMLInputElement>(null)
  const importRequest = useRef<AbortController | null>(null)
  const memoryDialog = useRef<HTMLDialogElement>(null)
  const codeDialog = useRef<HTMLDialogElement>(null)
  const [codeOpen, setCodeOpen] = useState(false)
  const historyDialog = useRef<HTMLDialogElement>(null)
  const alive = useRef(true)
  const activeRequest = useRef<AbortController | null>(null)
  const bottom = useRef<HTMLDivElement>(null)
  const messagesPane = useRef<HTMLDivElement>(null)
  const composing = useRef(false)
  const request = useRef<{ id: string; conversation: string; content: string; contextMode: AIContextMode; importID: string } | null>(null)
  const waiting = conversation?.messages?.some(message => message.status === 'pending') || false

  const load = useCallback(async (signal?: AbortSignal) => {
    setLoading(true); setLoadError('')
    try {
      const value = await aiApi.overview(signal)
      if (signal?.aborted || !alive.current) return
      setOverview(value); setMemory(value.memory)
    } catch (cause) { if (!signal?.aborted && alive.current) setLoadError(errorMessage(cause)) }
    finally { if (!signal?.aborted && alive.current) setLoading(false) }
  }, [])

  useEffect(() => {
    alive.current = true
    const abort = new AbortController(); void load(abort.signal)
    return () => { alive.current = false; abort.abort(); activeRequest.current?.abort(); importRequest.current?.abort() }
  }, [load])

  useEffect(() => {
    if (!overview?.private_knowledge_enabled) { setImports([]); setImportSelected(''); setImportHistory(null); return }
    const abort = new AbortController()
    void aiApi.imports(abort.signal).then(value => { if (!abort.signal.aborted) setImports(value) }).catch(cause => { if (!abort.signal.aborted) setImportError(errorMessage(cause)) })
    return () => abort.abort()
  }, [overview?.private_knowledge_enabled])

  async function readImport(id: string, before = 0) {
    if (sending || importBusy || !overview?.private_knowledge_enabled) return
    importRequest.current?.abort()
    const abort = new AbortController(); importRequest.current = abort
    setImportBusy(true); setImportError('')
    if (!before) { followLatest.current = true; setSelected(''); setConversation(null); setImportSelected(id); setImportHistory(null); setContextMode('project'); setMemory(overview?.project_memory || ''); request.current = null; historyDialog.current?.close() }
    try {
      const value = await aiApi.imported(id, before, abort.signal)
      if (!alive.current || abort.signal.aborted) return
      if (before && messagesPane.current) olderScroll.current = { height: messagesPane.current.scrollHeight, top: messagesPane.current.scrollTop }
      setImportHistory(current => before && current?.id === id ? { ...value, messages: [...(value.messages || []), ...(current.messages || [])] } : value)
    } catch (cause) { if (alive.current && !abort.signal.aborted) setImportError(errorMessage(cause)) }
    finally { if (alive.current && !abort.signal.aborted) setImportBusy(false) }
  }

  async function uploadImports(file?: File) {
    if (!file || importBusy || !overview?.private_knowledge_enabled) return
    setImportBusy(true); setImportError('')
    const abort = new AbortController(); importRequest.current = abort
    try {
      if (file.size > 50 * 1024 * 1024) throw new Error(t('同步文件过大，请分批导出。'))
      const data: unknown = JSON.parse(await file.text())
      const bundle = data as { version?: number; threads?: AISyncImport[] }
      if (!bundle || bundle.version !== 1 || !Array.isArray(bundle.threads) || !bundle.threads.length || bundle.threads.length > 200) throw new Error(t('请选择本地同步工具导出的 JSON 文件。'))
      for (const thread of bundle.threads) {
        if (!alive.current || abort.signal.aborted) return
        await aiApi.syncImport(thread, abort.signal)
      }
      const values = await aiApi.imports(abort.signal)
      if (alive.current && !abort.signal.aborted) setImports(values)
    } catch (cause) { if (alive.current && !abort.signal.aborted) setImportError(errorMessage(cause)) }
    finally { if (alive.current && !abort.signal.aborted) setImportBusy(false); if (importFile.current) importFile.current.value = '' }
  }

  useEffect(() => {
    if (!selected) { setConversation(null); return }
    const abort = new AbortController(); let timer: ReturnType<typeof setTimeout> | undefined
    async function poll() {
      try {
        const value = await aiApi.conversation(selected, abort.signal)
        if (abort.signal.aborted) return
        setConversation(value); setError('')
        if (value.messages?.some(message => message.status === 'pending')) timer = setTimeout(() => { void poll() }, 2000)
        else {
          const latest = await aiApi.overview(abort.signal)
          if (!abort.signal.aborted) setOverview(latest)
        }
      } catch (cause) { if (!abort.signal.aborted) { setError(errorMessage(cause)); timer = setTimeout(() => { void poll() }, 5000) } }
    }
    void poll()
    return () => { abort.abort(); if (timer) clearTimeout(timer) }
  }, [selected, sending])

  const visibleMessages = importSelected ? importHistory?.messages : conversation?.messages

  function updateScroll() {
    const pane = messagesPane.current
    if (!pane) return
    const away = pane.scrollHeight - pane.clientHeight - pane.scrollTop > 100
    followLatest.current = !away; setShowLatest(away)
  }

  function jumpToLatest() {
    followLatest.current = true
    const pane = messagesPane.current
    if (!pane) return
    if (pane.scrollTo) pane.scrollTo({ top: pane.scrollHeight, behavior: window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' })
    else pane.scrollTop = pane.scrollHeight
  }

  useLayoutEffect(() => {
    const pane = messagesPane.current
    if (!pane) return
    if (olderScroll.current) {
      pane.scrollTop = olderScroll.current.top + pane.scrollHeight - olderScroll.current.height
      olderScroll.current = null
    } else if (!visibleMessages?.length) pane.scrollTop = 0
    else if (followLatest.current) pane.scrollTop = pane.scrollHeight
    updateScroll()
  }, [visibleMessages, waiting])

  useEffect(() => {
    const pane = messagesPane.current
    if (!pane || typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(updateScroll)
    observer.observe(pane)
    return () => observer.disconnect()
  }, [loading])

  function choose(id: string) { if (sending || importBusy) return; followLatest.current = true; const item = overview?.conversations.find(value => value.id === id); const mode = item?.context_mode || contextMode; setImportSelected(''); setImportHistory(null); setOriginImportID(item?.origin_import_id || ''); setContextMode(mode); setMemory(mode === 'project' ? overview?.project_memory || '' : overview?.memory || ''); setSelected(id); setConversation(null); setError(''); historyDialog.current?.close(); request.current = null }

  function continueImport() {
    if (!importHistory || importBusy) return
    setOriginImportID(importHistory.id); setImportSelected(''); setImportHistory(null)
    setSelected(''); setConversation(null); request.current = null; setContextMode('project')
    setMemory(overview?.project_memory || ''); setError(''); document.getElementById('ai-question')?.focus()
  }

  function changeContext(mode: AIContextMode) {
    if (sending || waiting || importBusy) return
    // 每段对话使用固定模式，切换后开始新对话，不继承私人历史。
    setSelected(''); setConversation(null); request.current = null; setContextMode(mode)
    setImportSelected(''); setImportHistory(null); setOriginImportID('')
    setMemory(mode === 'project' ? overview?.project_memory || '' : overview?.memory || '')
    setMemorySaved(false); setError('')
  }

  async function send(event: FormEvent) {
    event.preventDefault()
    const content = input.trim()
    if (!content || sending || waiting || importBusy || (importSelected && !importHistory)) return
    if ([...content].length > 4000) { setError(t('问题最多 4000 字。')); return }
    setSending(true); setError('')
    const importID = importSelected || originImportID
    const mode = importID ? 'project' : contextMode
    const id = selected || (request.current?.content === content ? request.current.conversation : crypto.randomUUID())
    if (!request.current || request.current.content !== content || request.current.conversation !== id || request.current.contextMode !== mode || request.current.importID !== importID) request.current = { id: crypto.randomUUID(), conversation: id, content, contextMode: mode, importID }
    const abort = new AbortController(); activeRequest.current = abort
    try {
      const value = await aiApi.send(id, request.current.id, content, abort.signal, mode, importID || undefined)
      if (!alive.current) return
      followLatest.current = true; setInput(''); setImportSelected(''); setImportHistory(null); setOriginImportID(importID); setContextMode(mode); setConversation(value); setSelected(id); request.current = null
      const latest = await aiApi.overview(abort.signal)
      if (alive.current) setOverview(latest)
    } catch (cause) { if (alive.current && !abort.signal.aborted) setError(errorMessage(cause)) }
    finally { if (alive.current) setSending(false) }
  }

  async function saveMemory(value = memory) {
    if (memoryBusy || [...value].length > 4000) return
    setMemoryBusy(true); setMemorySaved(false); setError('')
    try { await aiApi.memory(value, undefined, contextMode); if (alive.current) { setMemory(value); setMemorySaved(true); setOverview(current => current ? { ...current, [contextMode === 'project' ? 'project_memory' : 'memory']: value } : current) } }
    catch (cause) { if (alive.current) setError(errorMessage(cause)) }
    finally { if (alive.current) setMemoryBusy(false) }
  }

  function rememberConclusion() {
    const last = [...(conversation?.messages || [])].reverse().find(message => message.role === 'assistant' && message.status === 'completed')
    if (!last) return
    const value = `${memory.trim()}\n\n${conversation?.title || ''}\n${last.content}`.trim()
    memoryDialog.current?.showModal()
    if ([...value].length > 4000) { setError(t('记忆空间不足，请先整理记忆，或让 AI 总结成更短的结论。')); return }
    void saveMemory(value)
  }

  async function remove() {
    if (!selected || waiting || sending) return
    if (!window.confirm(t('删除此对话？长期记忆会保留。'))) return
    try { await aiApi.remove(selected); if (alive.current) { choose(''); setOverview(await aiApi.overview()) } }
    catch (cause) { if (alive.current) setError(errorMessage(cause)) }
  }

  const lastAnswer = [...(conversation?.messages || [])].reverse().find(message => message.role === 'assistant' && message.status === 'completed')
  const privateKnowledge = overview?.private_knowledge_enabled === true && contextMode === 'project'
  const prompts = privateKnowledge ? [
    { icon: 'book', title: '练习面试', question: '按照资料里的学习进度，继续问我一道面试题。' },
    { icon: 'code', title: '理解项目', question: '结合我的项目，解释 JWT 为什么需要 token_version。' },
    { icon: 'activity', title: '排查问题', question: '帮我分析博客首次访问慢的原因，给出修改方案。' },
  ] as const : [
    { icon: 'book', title: '练习面试', question: '请问我一道 Go 后端面试题，等我回答后再点评。' },
    { icon: 'code', title: '理解代码', question: '解释 JWT 与 token_version 的用途，给出一个简单例子。' },
    { icon: 'activity', title: '排查问题', question: '我会提供遇到的问题，请帮我分析原因和排查步骤。' },
  ] as const
  const history = overview && <>
    <div className="ai-project-card"><span className="ai-project-mark">Y<span>.</span></span><div><strong>{privateKnowledge ? 'yia-blog' : t('通用聊天')}</strong><small>{t(privateKnowledge ? '项目与面试资料' : '独立对话与记忆')}</small></div></div>
    <button type="button" className="ai-new-thread" disabled={sending} onClick={() => choose('')}><UiIcon name="plus" size={17} />{t('新建对话')}</button>
    <p className="ai-rail-label">{t('历史对话')}</p>
    <nav className="ai-conversation-list" aria-label={t('历史对话')}>
      {overview.conversations.map(item => <button type="button" key={item.id} disabled={sending} aria-current={selected === item.id ? 'true' : undefined} onClick={() => choose(item.id)}><UiIcon name="comment" size={15} /><span><strong>{item.title}</strong><small>{new Date(item.update_at).toLocaleDateString()}</small></span></button>)}
      {!overview.conversations.length && <p className="ai-rail-empty">{t('你的对话会保存在这里。')}</p>}
    </nav>
    {overview.private_knowledge_enabled && <div className="ai-import-rail">
      <p className="ai-rail-label">{t('本地 Codex 历史')}</p>
      <button type="button" className="ai-tool-button" disabled={importBusy} onClick={() => importFile.current?.click()}><UiIcon name="folder" size={14} />{t(importBusy ? '正在同步…' : '同步本地历史')}</button>
      <nav className="ai-conversation-list" aria-label={t('本地 Codex 历史')}>{imports.map(item => <button type="button" key={item.id} disabled={sending || importBusy} aria-current={importSelected === item.id ? 'true' : undefined} onClick={() => { void readImport(item.id) }}><UiIcon name="comment" size={15} /><span><strong>{item.title}</strong><small>{new Date(item.updated_at).toLocaleDateString()}</small></span></button>)}</nav>
      <p className="small-help">{t('仅超管可同步和读取；只包含 yia-blog / yia-project。')}</p>
      {importError && <p className="form-error" role="alert">{importError}</p>}
    </div>}
    <div className="ai-rail-footer"><span><i className="ai-status-dot" />{privateKnowledge ? t('已关联 {count} 份资料', { count: overview.knowledge_documents }) : t('通用聊天 · 未关联站主私有资料')}</span><small>{t('对话自动保存，可以跨设备继续。')}</small></div>
  </>

  return <section className="ai-page" aria-labelledby="ai-title">
    <header className="ai-heading">
      <div className="ai-heading-title"><button type="button" className="icon-button ai-mobile-history" aria-label={t('历史对话')} onClick={() => historyDialog.current?.showModal()} disabled={!overview}><UiIcon name="clock" size={18} /></button><h1 id="ai-title">{t('AI 助手')}<span className="accent-dot">.</span></h1><span className="ai-model">{overview?.model || 'yia-blog'}</span></div>
      <div className="ai-heading-actions">{overview?.private_knowledge_enabled && <button type="button" className="ai-tool-button" aria-label={t('代码任务')} onClick={() => { setCodeOpen(true); codeDialog.current?.showModal() }}><UiIcon name="code" size={16} /><span>{t('代码任务')}</span></button>}<button type="button" className="ai-tool-button" aria-label={t('长期记忆')} disabled={!overview} onClick={() => memoryDialog.current?.showModal()}><UiIcon name="book" size={16} /><span>{t('长期记忆')}</span></button><Link className="ai-tool-button ai-return" to="/admin/articles" aria-label={t('返回管理')}><UiIcon name="back" size={16} /><span>{t('返回管理')}</span></Link></div>
    </header>
      {loading ? <StatePanel kind="loading" title={t('正在读取对话…')} /> : loadError ? <StatePanel kind="error" message={loadError} onRetry={() => { void load() }} /> : overview && <>
        <div className="ai-layout">
          {overview.private_knowledge_enabled && <input ref={importFile} type="file" accept=".json,application/json" hidden aria-label={t('同步本地历史')} onChange={event => { void uploadImports(event.target.files?.[0]) }} />}
          <aside className="ai-sidebar">{history}</aside>
          <div className="ai-chat">
            <div className="ai-context-switch" aria-label={t('对话资料模式')}><button type="button" aria-pressed={contextMode === 'general'} disabled={sending || waiting || importBusy} onClick={() => changeContext('general')}>{t('通用聊天')}</button>{overview.private_knowledge_enabled && <button type="button" aria-pressed={contextMode === 'project'} disabled={sending || waiting || importBusy} onClick={() => changeContext('project')}><UiIcon name="folder" size={14} />{t('项目资料')}</button>}<small>{t(privateKnowledge ? '已开启项目资料；切换模式会新建对话。' : '不发送项目资料与项目记忆。')}</small></div>
            <div className="ai-chat-toolbar"><h2>{importSelected ? importHistory?.title || t('本地 Codex 历史') : conversation?.title || t(originImportID ? '继续本地对话' : '新建对话')}</h2><span className="ai-quota">{t('今日 {used} / {limit}', { used: overview.used, limit: overview.daily_limit })}</span>{selected && <button type="button" className="icon-button" aria-label={t('删除对话')} title={t('删除对话')} disabled={waiting || sending} onClick={() => { void remove() }}><UiIcon name="trash" size={16} /></button>}</div>
            <div className="ai-message-area"><div ref={messagesPane} className="ai-messages" onScroll={updateScroll} aria-live="polite" aria-relevant="additions text">
              {!importSelected && !conversation?.messages?.length && <div className="ai-empty"><div className="ai-empty-mark" aria-hidden="true">Y<span>.</span></div><p className="eyebrow">YIA-BLOG / YOUR WORKSPACE</p><h2>{t('今天想聊什么？')}</h2><p>{t('练一道面试题，或把项目里的一个问题讲清楚。')}</p><div className="ai-prompts">{prompts.map(prompt => <button type="button" key={prompt.title} onClick={() => { setInput(t(prompt.question)); document.getElementById('ai-question')?.focus() }}><UiIcon name={prompt.icon} size={19} /><strong>{t(prompt.title)}</strong><span>{t(prompt.question)}</span><UiIcon name="arrow" size={15} /></button>)}</div></div>}
              {importSelected && <div className="ai-import-banner"><p>{t('本地历史副本 · 不会修改桌面会话')}</p>{importHistory && <><small>{t('已显示 {shown} / {total} 条消息', { shown: importHistory.messages?.length || 0, total: importHistory.total_messages })}</small>{importHistory.next_before > 0 && <button type="button" className="ai-tool-button" disabled={importBusy} onClick={() => { void readImport(importSelected, importHistory.next_before) }}>{t('加载更早消息')}</button>}<button type="button" className="ai-tool-button" disabled={importBusy} onClick={continueImport}>{t('继续本地对话')}</button></>}{importError && <p className="form-error" role="alert">{importError}</p>}</div>}
              {visibleMessages?.map(message => <article className={`ai-message ai-message-${message.role}`} key={message.id}>
                <span className="ai-message-label">{message.role === 'assistant' && <span className="ai-author-mark" aria-hidden="true">Y.</span>}{t(message.role === 'user' ? '你' : 'AI 助手')}</span>
                {message.status === 'pending' ? <p className="ai-waiting" role="status">{t(privateKnowledge ? '正在结合资料回答，离开页面后仍会继续…' : '正在回答，离开页面后仍会继续…')}</p> : message.role === 'user' || message.status === 'failed' ? <p className={message.status === 'failed' ? 'form-error' : ''}>{message.content}</p> : <div className="reading-content ai-answer" dangerouslySetInnerHTML={{ __html: renderAnswer(message.content) }} />}
                {!!message.sources?.length && <details className="ai-sources"><summary>{t('参考了 {count} 段资料', { count: message.sources.length })}</summary>{message.sources.map((source, index) => <details key={index}><summary>{source.path}:{source.line}<span>{t('资料更新于')} {new Date(source.updated_at).toLocaleString()}</span></summary><pre>{source.excerpt}</pre></details>)}</details>}
                {!importSelected && message.id === lastAnswer?.id && <button type="button" className="ai-save-conclusion" disabled={waiting || sending || memoryBusy} onClick={rememberConclusion}><UiIcon name="plus" size={14} />{t('保存当前结论')}</button>}
              </article>)}<div ref={bottom} />
            </div>{showLatest && <button type="button" className="ai-jump-latest" aria-label={t('跳到最新消息')} title={t('跳到最新消息')} onClick={jumpToLatest}><UiIcon name="up" size={19} /></button>}</div>
            <form className="ai-composer" onSubmit={event => { void send(event) }} aria-busy={sending || waiting}>
              <div className="ai-input-box"><label className="sr-only" htmlFor="ai-question">{t('向 AI 提问')}</label>
                <textarea id="ai-question" value={input} rows={2} maxLength={8000} placeholder={t(importSelected ? '接着这段对话提问…' : '输入问题…')} disabled={sending} onChange={event => setInput(event.target.value)} onCompositionStart={() => { composing.current = true }} onCompositionEnd={() => { composing.current = false }} onKeyDown={event => { if (event.key === 'Enter' && !event.shiftKey && !composing.current && !event.nativeEvent.isComposing && event.keyCode !== 229) { event.preventDefault(); event.currentTarget.form?.requestSubmit() } }} />
                <div className="ai-input-actions"><span>{importSelected || originImportID ? t('已关联本地对话 · 项目资料') : overview.model}</span><button type="submit" className="ai-send" aria-label={t(sending || waiting ? '正在回答…' : '发送')} title={t(sending || waiting ? '正在回答…' : '发送')} disabled={sending || waiting || importBusy || (Boolean(importSelected) && !importHistory) || !input.trim()}><UiIcon name="up" size={19} /></button></div>
              </div>
              <p className="ai-composer-hint">{t('回车发送，Shift + 回车换行。')}</p>
              {importSelected && <p className="ai-composer-hint">{t('发送会带入这段历史，新建网页项目对话。')}</p>}
              {error && <p className="form-error" role="alert">{error}</p>}
            </form>
          </div>
        </div>
        <dialog ref={historyDialog} className="ai-history-dialog" aria-label={t('历史对话')}><button type="button" className="icon-button ai-dialog-close" aria-label={t('关闭历史对话')} onClick={() => historyDialog.current?.close()}><UiIcon name="close" size={18} /></button>{history}</dialog>
        <dialog ref={memoryDialog} className="ai-memory" aria-labelledby="ai-memory-title">
          <header><div><p className="eyebrow">YIA-BLOG / MEMORY</p><h2 id="ai-memory-title">{t('长期记忆与学习进度')}</h2></div><button type="button" className="icon-button" aria-label={t('关闭记忆面板')} onClick={() => memoryDialog.current?.close()}><UiIcon name="close" size={18} /></button></header>
          <p className="small-help">{t('写下你希望持续记住的背景、薄弱点和当前进度。保存后，新对话和其他设备都会使用。')}</p><label className="sr-only" htmlFor="ai-memory">{t('长期记忆')}</label><textarea id="ai-memory" value={memory} rows={8} maxLength={8000} onChange={event => { setMemory(event.target.value); setMemorySaved(false) }} />
          <div className="ai-memory-actions"><span>{[...memory].length} / 4000</span>{memorySaved && <span className="ai-memory-saved" role="status">{t('记忆已保存。')}</span>}<button type="button" className="button primary small" disabled={memoryBusy || [...memory].length > 4000} onClick={() => { void saveMemory() }}>{t(memoryBusy ? '正在保存…' : '保存记忆')}</button></div>
          {error && <p className="form-error" role="alert">{error}</p>}<p className="ai-privacy small-help">{t(privateKnowledge ? '相关资料片段会发送给模型服务。' : '仅发送你的提问、对话和记忆给模型服务。')}</p>
        </dialog>
        {overview.private_knowledge_enabled && <dialog ref={codeDialog} className="ai-memory ai-code-dialog" aria-labelledby="ai-code-title" onClose={() => setCodeOpen(false)}><header><h2 id="ai-code-title">{t('代码任务')}</h2><button type="button" className="icon-button" aria-label={t('关闭代码任务')} onClick={() => codeDialog.current?.close()}><UiIcon name="close" size={18} /></button></header>{codeOpen && <AICodePanel contextMode={contextMode} />}</dialog>}
      </>}
  </section>
}
