import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useI18n } from '../i18n'
import { useResource } from '../hooks/useResource'
import { loadChapters, novel, readProgress, readingKey } from '../lib/novel'
import StatePanel from '../components/StatePanel'
import UiIcon from '../components/UiIcon'

export default function NovelView() {
  const { t } = useI18n()
  const chapters = useResource(loadChapters)
  const [progress, setProgress] = useState(readProgress)
  useEffect(() => {
    const sync = (event: StorageEvent) => { if (event.key === readingKey || event.key === null) setProgress(readProgress()) }
    window.addEventListener('storage', sync)
    return () => window.removeEventListener('storage', sync)
  }, [])
  const first = chapters.data?.[0]
  const resume = chapters.data?.find((chapter) => chapter.id === progress)
  return <div className="container novel-page">
    <section className="novel-intro" aria-labelledby="novel-title">
      <div className="novel-jacket" aria-hidden="true"><small>YIA / FICTION</small><strong>未归之人</strong><span>IRIS</span></div>
      <div className="novel-copy"><p className="eyebrow">{t('原创连载')} · Iris</p><h1 id="novel-title">{novel.title}</h1>
        <p className="novel-description">{t('门外不是熟悉的楼道，而是一座即将崩塌的未来。有人被征召而来，有人从未离开。回家的路，从这里开始。')}</p>
        <p className="small-help">{t('第一部 · 连载中')}{chapters.data && <> · {t('已公开 {count} 章', { count: chapters.data.length })}</>}</p>
        <div className="novel-actions">{first && <Link className="button primary" to={`/articles/${(resume || first).id}`}>{t(resume ? '继续阅读' : '开始阅读')}<UiIcon name="arrow" size={17} /></Link>}
          {resume && first && resume.id !== first.id && <Link className="text-link" to={`/articles/${first.id}`}>{t('从头开始')}</Link>}
        </div>
        {resume && <p className="small-help">{t('上次打开：{title}', { title: resume.title.replace('《未归之人》', '') })}</p>}
        <p className="small-help">{t('阅读记录仅保存在当前浏览器，记录上次打开的章节。')}</p>
      </div>
    </section>
    <section aria-labelledby="chapter-list-title"><div className="section-heading"><h2 id="chapter-list-title">{t('章节目录')}</h2><span className="heading-rule" /></div>
      {chapters.pending ? <StatePanel kind="loading" /> : chapters.error ? <StatePanel kind="error" message={t(chapters.error)} onRetry={chapters.reload} />
        : !chapters.data?.length ? <StatePanel kind="empty" title={t('暂时没有可阅读的章节')} />
        : <ol className="novel-chapters">{chapters.data.map((chapter) => <li key={chapter.id}><Link to={`/articles/${chapter.id}`}><span className="chapter-number">{String(chapter.number).padStart(2, '0')}</span><span>{chapter.title.replace('《未归之人》', '')}{resume?.id === chapter.id && <small>{t('上次阅读')}</small>}</span><UiIcon name="arrow" size={16} /></Link></li>)}</ol>}
    </section>
  </div>
}
