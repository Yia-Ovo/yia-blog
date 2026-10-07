import { useEffect, useId, useState } from 'react'
import { useI18n } from '../i18n'
import { dailyQuoteIndex, dailyQuotes, localDateKey, nextLocalMidnightDelay } from '../lib/dailyQuote'
import './daily-quote.css'

export default function DailyQuoteCard() {
  const { locale, t } = useI18n()
  const titleID = useId()
  const [selection, setSelection] = useState(() => ({ date: new Date(), offset: 0 }))

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>
    function refreshDay() {
      const now = new Date()
      setSelection((previous) => localDateKey(previous.date) === localDateKey(now) ? previous : { date: now, offset: 0 })
      clearTimeout(timer)
      // 用下一个本地午夜而非固定 24 小时；稍作缓冲，避免在日界线前触发。
      timer = setTimeout(refreshDay, nextLocalMidnightDelay(now) + 50)
    }
    function onVisible() { if (document.visibilityState === 'visible') refreshDay() }
    refreshDay()
    document.addEventListener('visibilitychange', onVisible)
    window.addEventListener('focus', refreshDay)
    window.addEventListener('pageshow', refreshDay)
    return () => {
      clearTimeout(timer)
      document.removeEventListener('visibilitychange', onVisible)
      window.removeEventListener('focus', refreshDay)
      window.removeEventListener('pageshow', refreshDay)
    }
  }, [])

  function nextThought() {
    const now = new Date()
    setSelection((previous) => ({ date: now, offset: localDateKey(previous.date) === localDateKey(now) ? previous.offset + 1 : 1 }))
  }

  const quote = (dailyQuotes[dailyQuoteIndex(selection.date, selection.offset)] ?? dailyQuotes[0])[locale]
  return <section className="daily-quote-card" aria-labelledby={titleID}>
    <header className="daily-quote-heading">
      <h3 id={titleID}>{t('每日一言')}</h3>
      <time dateTime={localDateKey(selection.date)}>{new Intl.DateTimeFormat(locale, { month: 'short', day: 'numeric' }).format(selection.date)}</time>
    </header>
    <figure className="daily-quote-thought">
      <span className="daily-quote-mark" aria-hidden="true">“</span>
      <blockquote aria-live="polite" aria-atomic="true"><p>{quote}</p></blockquote>
      <figcaption><span aria-hidden="true">— </span>{t('本站原创')}</figcaption>
    </figure>
    <footer className="daily-quote-footer">
      <span className="daily-quote-rule" aria-hidden="true" />
      <button type="button" onClick={nextThought}>
        <svg aria-hidden="true" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"><path d="M20 7v5h-5M4 17v-5h5" /><path d="M6.1 6.1a8 8 0 0 1 13.5 4M4.4 13.9a8 8 0 0 0 13.5 4" /></svg>
        {t('换一句')}
      </button>
    </footer>
  </section>
}
