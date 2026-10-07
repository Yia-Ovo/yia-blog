import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import DailyQuoteCard from '../src/components/DailyQuoteCard'
import { locales, setLocale } from '../src/i18n'
import { dailyQuoteMessages } from '../src/i18n/dailyQuote'
import { dailyQuoteIndex, dailyQuotes, localDateKey, nextLocalMidnightDelay } from '../src/lib/dailyQuote'

let root: Root
let host: HTMLDivElement
const today = () => new Date(2026, 9, 7, 12, 0, 0)
const thought = () => host.querySelector('blockquote p')!.textContent
async function open() { await act(async () => { root.render(<DailyQuoteCard />) }) }
async function next() { await act(async () => { host.querySelector('button')!.click() }) }

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(today())
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  setLocale('zh-CN')
  host = document.createElement('div')
  document.body.append(host)
  root = createRoot(host)
})
afterEach(async () => {
  await act(async () => { root.unmount() })
  host.remove()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.useRealTimers()
  setLocale('zh-CN')
})

describe('daily thought', () => {
  it('uses the local calendar date and changes only at a day boundary', () => {
    const morning = new Date(2026, 9, 7, 0, 0, 0)
    const night = new Date(2026, 9, 7, 23, 59, 59)
    expect(localDateKey(night)).toBe('2026-10-07')
    expect(dailyQuoteIndex(morning)).toBe(dailyQuoteIndex(night))
    expect(dailyQuoteIndex(new Date(2026, 9, 8))).toBe((dailyQuoteIndex(night) + 1) % dailyQuotes.length)
    expect(nextLocalMidnightDelay(night)).toBe(1000)
    expect(nextLocalMidnightDelay(new Date(2026, 11, 31, 23, 59, 59, 500))).toBe(500)
  })

  it('cycles without repeating before all thoughts are shown, then remounts with the daily default', async () => {
    await open()
    const original = thought()
    const seen = new Set([original])
    for (let index = 1; index < dailyQuotes.length; index++) { await next(); seen.add(thought()) }
    expect(seen.size).toBe(dailyQuotes.length)
    await next()
    expect(thought()).toBe(original)
    await next()
    await act(async () => { root.render(null) })
    await open()
    expect(thought()).toBe(original)
  })

  it('resets a manually changed thought after midnight and updates the visible date', async () => {
    vi.setSystemTime(new Date(2026, 9, 7, 23, 59, 59))
    await open()
    await next(); await next()
    await act(async () => { vi.advanceTimersByTime(1100) })
    expect(host.querySelector('time')!.dateTime).toBe('2026-10-08')
    expect(thought()).toBe(dailyQuotes[dailyQuoteIndex(new Date(2026, 9, 8))]!['zh-CN'])
    expect(vi.getTimerCount()).toBe(1)
  })

  it.each(['focus', 'pageshow', 'visibilitychange'])('catches up after sleep on %s and releases listeners on unmount', async (eventName) => {
    await open(); await next()
    vi.setSystemTime(new Date(2026, 9, 9, 9))
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    const target = eventName === 'visibilitychange' ? document : window
    await act(async () => { target.dispatchEvent(new Event(eventName)) })
    expect(host.querySelector('time')!.dateTime).toBe('2026-10-09')
    expect(thought()).toBe(dailyQuotes[dailyQuoteIndex(new Date(2026, 9, 9))]!['zh-CN'])
    await act(async () => { root.render(null) })
    expect(vi.getTimerCount()).toBe(0)
    await act(async () => { target.dispatchEvent(new Event(eventName)) })
    expect(vi.getTimerCount()).toBe(0)
  })

  it('has original content and translated controls in all four languages without resetting selection', async () => {
    expect(dailyQuotes.length).toBeGreaterThanOrEqual(12)
    for (const entry of dailyQuotes) for (const locale of locales) expect(entry[locale].trim().length).toBeGreaterThan(10)
    await open(); await next()
    for (const locale of locales) {
      await act(async () => { setLocale(locale) })
      expect(thought()).toBe(dailyQuotes[dailyQuoteIndex(today(), 1)]![locale])
      expect(host.querySelector('h3')!.textContent).toBe(locale === 'zh-CN' ? '每日一言' : dailyQuoteMessages['每日一言'][locale])
      expect(host.querySelector('figcaption')!.textContent).toContain(locale === 'zh-CN' ? '本站原创' : dailyQuoteMessages['本站原创'][locale])
      expect(host.querySelector('button')!.textContent).toBe(locale === 'zh-CN' ? '换一句' : dailyQuoteMessages['换一句'][locale])
    }
  })

  it('works with browser storage disabled', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('blocked') })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('blocked') })
    await open()
    const original = thought()
    await next()
    expect(thought()).not.toBe(original)
    expect(localStorage.getItem).not.toHaveBeenCalled()
    expect(localStorage.setItem).not.toHaveBeenCalled()
  })
})
