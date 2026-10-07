import { useEffect, useId, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Link } from 'react-router-dom'
import { useResource } from '../hooks/useResource'
import { useReadingMenuPosition } from '../hooks/useReadingMenuPosition'
import { useI18n } from '../i18n'
import { loadChapters, novel } from '../lib/novel'
import UiIcon from './UiIcon'

/** NovelReadingMenu 只使用公开目录中的章节；挂在 body 下，避免页面过渡影响 fixed 定位。 */
export default function NovelReadingMenu({ chapterID }: { chapterID: number }) {
  const { t } = useI18n()
  const chapters = useResource(loadChapters)
  const [scrolled, setScrolled] = useState(false)
  const [open, setOpen] = useState(false)
  const floating = useReadingMenuPosition()
  const { shell } = floating
  const toggle = useRef<HTMLButtonElement>(null)
  const panel = useRef<HTMLDivElement>(null)
  const panelID = useId()
  const index = chapters.data?.findIndex((chapter) => chapter.id === chapterID) ?? -1
  const previous = index > 0 ? chapters.data?.[index - 1] : undefined
  const next = index >= 0 ? chapters.data?.[index + 1] : undefined
  const shown = scrolled || open

  useEffect(() => {
    const update = () => setScrolled(window.scrollY >= 320)
    update()
    window.addEventListener('scroll', update, { passive: true })
    return () => window.removeEventListener('scroll', update)
  }, [])

  useEffect(() => {
    if (!open) return
    const outside = (event: PointerEvent) => {
      if (event.target instanceof Node && !shell.current?.contains(event.target)) setOpen(false)
    }
    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        setOpen(false)
        toggle.current?.focus()
      }
    }
    document.addEventListener('pointerdown', outside)
    document.addEventListener('keydown', escape)
    return () => {
      document.removeEventListener('pointerdown', outside)
      document.removeEventListener('keydown', escape)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    // 聚焦当前章并仅滚动目录容器，不改变用户正在阅读的页面位置。
    const current = panel.current?.querySelector<HTMLAnchorElement>('[aria-current="page"]')
    const list = panel.current?.querySelector<HTMLOListElement>('ol')
    if (current && list) list.scrollTop = Math.max(0, current.offsetTop - list.offsetTop - list.clientHeight / 2)
    const focusTarget = current || panel.current
    focusTarget?.focus({ preventScroll: true })
  }, [open, chapters.data])

  function backToTop() {
    setOpen(false)
    document.getElementById('reading-title')?.focus({ preventScroll: true })
    window.scrollTo({ top: 0, behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth' })
  }

  return createPortal(<div ref={shell} className="reading-menu" style={floating.style} data-dragging={floating.dragging} data-visible={shown} inert={!shown} aria-hidden={!shown}
    onBlur={(event) => { if (event.relatedTarget instanceof Node && !event.currentTarget.contains(event.relatedTarget)) setOpen(false) }}>
    {open && <div ref={panel} id={panelID} className="reading-menu-panel" style={floating.panelStyle} role="region" aria-label={t('章节目录')} tabIndex={-1}>
      <div className="reading-menu-heading"><div><small>{t('章节目录')}</small><strong>{novel.title}</strong></div><button type="button" className="icon-button" aria-label={t('关闭目录')} onClick={() => { setOpen(false); toggle.current?.focus() }}><UiIcon name="close" size={17} /></button></div>
      {chapters.pending ? <p className="reading-menu-state" role="status">{t('正在加载章节…')}</p>
        : chapters.error ? <div className="reading-menu-state" role="alert"><p>{t('章节目录暂时无法加载，请稍后重试。')}</p><button type="button" className="text-link" onClick={chapters.reload}>{t('重新加载目录')}</button></div>
        : !chapters.data?.length ? <p className="reading-menu-state">{t('暂时没有可阅读的章节')}</p>
        : <ol>{chapters.data.map((chapter) => <li key={chapter.id}><Link to={`/articles/${chapter.id}`} aria-current={chapter.id === chapterID ? 'page' : undefined} onClick={() => setOpen(false)}><span>{String(chapter.number).padStart(2, '0')}</span><span>{chapter.title.replace('《未归之人》', '')}</span></Link></li>)}</ol>}
      <Link className="reading-menu-overview" to="/novels">{t('前往作品页')}<UiIcon name="arrow" size={14} /></Link>
    </div>}
    <nav className="reading-menu-bar" aria-label={t('阅读菜单')}>
      <button type="button" className="reading-menu-drag" aria-label={t('拖动阅读菜单')} title={t('拖动调整位置，也可用方向键移动，Home 键复位')} {...floating.handle}><UiIcon name="grip" size={18} /></button>
      {previous ? <Link to={`/articles/${previous.id}`} title={previous.title}><UiIcon name="back" size={17} /><span>{t('上一章')}</span></Link> : <button type="button" disabled title={chapters.pending ? t('正在加载章节…') : t('没有可用的上一章')}><UiIcon name="back" size={17} /><span>{t('上一章')}</span></button>}
      <button ref={toggle} type="button" aria-expanded={open} aria-controls={open ? panelID : undefined} onClick={() => setOpen((value) => !value)}><UiIcon name="book" size={17} /><span>{t('目录')}</span></button>
      {next ? <Link to={`/articles/${next.id}`} title={next.title}><span>{t('下一章')}</span><UiIcon name="arrow" size={17} /></Link> : <button type="button" disabled title={chapters.pending ? t('正在加载章节…') : t('没有可用的下一章')}><span>{t('下一章')}</span><UiIcon name="arrow" size={17} /></button>}
      <button type="button" onClick={backToTop}><UiIcon name="up" size={17} /><span>{t('回顶部')}</span></button>
    </nav>
  </div>, document.body)
}
