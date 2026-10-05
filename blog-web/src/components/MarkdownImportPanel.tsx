import { useEffect, useRef, useState } from 'react'
import { useI18n } from '../i18n'
import { readMarkdownFile } from '../lib/markdownImport'
import type { ImportedMarkdown } from '../lib/markdownImport'
import MarkdownBody from './MarkdownBody'

export default function MarkdownImportPanel({ disabled, hasContent, onImport }: {
  disabled: boolean
  hasContent: boolean
  onImport: (value: ImportedMarkdown) => void
}) {
  const { t } = useI18n()
  const input = useRef<HTMLInputElement>(null)
  const generation = useRef(0)
  const [reading, setReading] = useState(false)
  const [pending, setPending] = useState<ImportedMarkdown | null>(null)
  const [error, setError] = useState('')
  useEffect(() => () => { generation.current++ }, [])

  function cancel() {
    generation.current++
    setReading(false); setPending(null); setError('')
  }
  async function choose(files: File[]) {
    if (disabled || !files.length) return
    const request = ++generation.current
    setPending(null); setError('')
    if (files.length !== 1) { setReading(false); setError('请一次导入一个 Markdown 文件。'); return }
    setReading(true)
    try {
      const result = await readMarkdownFile(files[0]!)
      if (generation.current === request) setPending(result)
    } catch (cause) {
      if (generation.current === request) setError(cause instanceof Error ? cause.message : '无法读取文件，请重新选择。')
    } finally {
      if (generation.current === request) setReading(false)
    }
  }

  return <section className="markdown-import" aria-label={t('Markdown 导入')}>
    <div className="markdown-import-toolbar">
      <button type="button" className="button secondary small" disabled={disabled || reading} onClick={() => input.current?.click()}>{t(reading ? '正在读取文件…' : '导入 Markdown')}</button>
      <input ref={input} className="sr-only" tabIndex={-1} type="file" accept=".md,.markdown,text/markdown" aria-label={t('选择 Markdown 文件')} disabled={disabled || reading} onChange={(event) => { void choose(Array.from(event.currentTarget.files || [])); event.currentTarget.value = '' }} />
      <span className="small-help">{t('UTF-8 · 最大 1 MB · 先预览，再导入')}</span>
      {reading && <button type="button" className="link-button" onClick={cancel}>{t('取消导入')}</button>}
    </div>
    {error && <p role="alert" className="form-error">{t(error)}</p>}
    {pending && <div className="markdown-import-preview">
      <p className="small-help">{pending.filename} · {t('{count} 字符', { count: pending.content.length })}</p>
      <h2>{pending.title}</h2>
      <div className="markdown-import-body"><MarkdownBody source={pending.content.slice(0, 4000)} /></div>
      {pending.content.length > 4000 && <p className="small-help">{t('仅预览前 4000 字符，导入会保留完整正文。')}</p>}
      <p className="small-help">{t('首行一级标题作为文章标题；没有时使用文件名。分类、封面和发布设置保持不变。')}</p>
      {hasContent && <p className="inline-note">{t('确认导入会替换当前标题和正文，请先保留需要的内容。')}</p>}
      <p className="small-help">{t('本地图片不会一并上传，请在导入后补充图片。')}</p>
      <div className="markdown-import-actions">
        <button type="button" className="button primary small" disabled={disabled || reading} onClick={() => { if (disabled) return; onImport(pending); cancel() }}>{t(hasContent ? '替换标题和正文' : '使用这份内容')}</button>
        <button type="button" className="button secondary small" onClick={cancel}>{t('取消导入')}</button>
      </div>
    </div>}
  </section>
}
