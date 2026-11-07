import { useEffect, useRef, useState } from 'react'
import { aiApi } from '../api/ai'
import type { AICodeJob, AIContextMode, AIWorkspace } from '../api/ai'
import { errorMessage } from '../api/http'
import UiIcon from '../components/UiIcon'
import { useI18n } from '../i18n'

const phases: Record<string, string> = { queued: '等待执行', running: '正在修改和测试', review: '等待审核', publish_queued: '等待发布分支', publishing: '正在发布分支', completed: '分支已发布', failed: '执行失败' }

export default function AICodePanel({ contextMode }: { contextMode: AIContextMode }) {
  const { t } = useI18n()
  const [workspace, setWorkspace] = useState<AIWorkspace | null>(null)
  const [prompt, setPrompt] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [reviewed, setReviewed] = useState('')
  const [selected, setSelected] = useState('')
  const submission = useRef<{ id: string; prompt: string; mode: AIContextMode } | null>(null)
  const alive = useRef(true)
  const job = workspace?.jobs.find(value => value.id === selected)

  useEffect(() => {
    alive.current = true
    const abort = new AbortController(); let timer: ReturnType<typeof setTimeout>
    async function poll() {
      try { const value = await aiApi.workspace(abort.signal); if (!abort.signal.aborted) { setWorkspace(value); setError('') } }
      catch (cause) { if (!abort.signal.aborted) setError(errorMessage(cause)) }
      if (!abort.signal.aborted) timer = setTimeout(() => { void poll() }, 5000)
    }
    void poll()
    return () => { alive.current = false; abort.abort(); clearTimeout(timer) }
  }, [])

  async function create() {
    if (!prompt.trim() || busy || !workspace?.enabled) return
    const text = prompt.trim()
    if (!submission.current || submission.current.prompt !== text || submission.current.mode !== contextMode) submission.current = { id: crypto.randomUUID(), prompt: text, mode: contextMode }
    setBusy(true); setError('')
    try {
      const value = await aiApi.createJob(submission.current.id, text, contextMode)
      if (!alive.current) return
      submission.current = null; setPrompt(''); setSelected(value.id); setReviewed('')
      setWorkspace(current => current ? { ...current, jobs: [value, ...current.jobs.filter(item => item.id !== value.id)] } : current)
    } catch (cause) { if (alive.current) setError(errorMessage(cause)) }
    finally { if (alive.current) setBusy(false) }
  }

  async function publish(value: AICodeJob) {
    if (busy || !value.review_digest || reviewed !== value.review_digest) return
    setBusy(true); setError('')
    try { await aiApi.publishJob(value.id, value.review_digest); const next = await aiApi.workspace(); if (alive.current) { setWorkspace(next); setReviewed('') } }
    catch (cause) { if (alive.current) setError(errorMessage(cause)) }
    finally { if (alive.current) setBusy(false) }
  }

  return <div className="ai-code-panel">
    <p className="small-help">{t('服务器开发工作区 · 修改后先审核差异。')}</p>
    {!workspace?.enabled && <p role="status">{t('服务器执行服务尚未启用。')}</p>}
    <label htmlFor="ai-code-prompt">{t('描述你想修改的项目功能')}</label>
    <textarea id="ai-code-prompt" rows={3} maxLength={4000} value={prompt} disabled={busy} onChange={event => setPrompt(event.target.value)} />
    <p className="small-help">{t(contextMode === 'project' ? '这次任务会关联项目资料。' : '这次任务仅使用项目源码和你的需求。')}</p>
    <button type="button" className="button primary small" disabled={busy || !workspace?.enabled || !prompt.trim()} onClick={() => { void create() }}>{t('创建代码任务')}</button>
    <p className="small-help">{t('每天最多 {count} 个任务。', { count: workspace?.daily_limit || 2 })}</p>
    {!!workspace?.jobs.length && <nav className="ai-job-list" aria-label={t('代码任务记录')}>{workspace.jobs.map(value => <button type="button" key={value.id} aria-current={selected === value.id} onClick={() => { setSelected(value.id); setReviewed('') }}><strong>{value.prompt}</strong><span>{t(phases[value.status] || value.status)}</span></button>)}</nav>}
    {job && <section className="ai-job-detail"><h3>{t(phases[job.status] || job.status)}</h3><code>{job.branch}</code><p>{job.summary}</p>
      {job.diff && <details open><summary>{t('修改差异')}</summary><pre>{job.diff}</pre></details>}
      {job.checks && <details><summary>{t('实际检查结果')}</summary><pre>{job.checks}</pre></details>}
      {job.status === 'review' && <><p>{t(job.tests_passed ? '检查通过，审核后可发布 GitHub 分支。' : '检查未通过，暂不能发布分支。')}</p><label className="ai-review-check"><input type="checkbox" checked={reviewed === job.review_digest && Boolean(job.review_digest)} disabled={!job.tests_passed} onChange={event => setReviewed(event.target.checked ? job.review_digest || '' : '')} />{t('我已查看差异和检查结果')}</label><button type="button" className="button primary small" disabled={busy || !job.tests_passed || reviewed !== job.review_digest} onClick={() => { void publish(job) }}>{t('发布 GitHub 分支')}</button><p className="small-help">{t('发布分支后，在 GitHub 审核合并；不会自动上线。')}</p></>}
      {job.url && <a href={job.url} target="_blank" rel="noreferrer">{t('查看 GitHub 差异')} <UiIcon name="arrow" size={14} /></a>}
    </section>}
    {error && <p className="form-error" role="alert">{error}</p>}
  </div>
}
