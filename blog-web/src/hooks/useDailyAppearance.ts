import { useEffect, useLayoutEffect, useState } from 'react'
import { appearanceApi, artworkURL } from '../api/appearance'
import { appearanceRefreshDelay, artworkTheme, loadArtwork } from '../lib/artwork'
import type { AppearanceMode, LoadedArtwork } from '../lib/artwork'

type Appearance = LoadedArtwork & { date: string }

// 轮换仅影响 CSS，不重新挂载页面，编辑中的文章不会因为跨日而丢失。
export function useDailyAppearance(mode: AppearanceMode) {
  const [appearance, setAppearance] = useState<Appearance | null>(null)

  useEffect(() => {
    let active = true
    let busy = false
    let sourceURL = ''
    let dueAt = 0
    let timer: ReturnType<typeof setTimeout> | undefined
    let controller: AbortController | undefined

    async function refresh() {
      if (!active || busy) return
      busy = true
      clearTimeout(timer)
      controller = new AbortController()
      const signal = controller.signal
      const deadline = setTimeout(() => controller?.abort(), 20_000)
      const started = performance.now()
      let delay = 60_000
      try {
        const daily = await appearanceApi.today(signal)
        if (!/^\d{4}-\d{2}-\d{2}$/.test(daily.date)) throw new Error('Invalid appearance date')
        const url = artworkURL(daily.image_url)
        // 图片下载耗时也计入下一次轮换，避免每日累积延迟。
        appearanceRefreshDelay(daily.server_time, daily.next_change_at)
        if (url !== sourceURL) {
          const loaded = await loadArtwork(url, signal)
          if (!active || signal.aborted) { URL.revokeObjectURL(loaded.objectURL); return }
          sourceURL = url
          setAppearance({ ...loaded, date: daily.date })
        } else if (active) {
          setAppearance(current => current && current.date !== daily.date ? { ...current, date: daily.date } : current)
        }
        delay = appearanceRefreshDelay(daily.server_time, daily.next_change_at, performance.now() - started)
      } catch { /* 图片或 API 暂时不可用时保留当前外观，稍后重试。 */ }
      finally {
        clearTimeout(deadline)
        busy = false
        if (active) {
          dueAt = Date.now() + delay
          timer = setTimeout(() => { void refresh() }, delay)
        }
      }
    }
    const resume = () => { if (document.visibilityState === 'visible' && Date.now() >= dueAt) void refresh() }
    document.addEventListener('visibilitychange', resume)
    window.addEventListener('focus', resume)
    void refresh()
    return () => {
      active = false
      controller?.abort()
      clearTimeout(timer)
      document.removeEventListener('visibilitychange', resume)
      window.removeEventListener('focus', resume)
    }
  }, [])

  useLayoutEffect(() => {
    if (!appearance) return
    const values = { ...artworkTheme(appearance.color, mode), '--daily-artwork': `url("${appearance.objectURL}")` }
    for (const [name, value] of Object.entries(values)) document.documentElement.style.setProperty(name, value)
    return () => { for (const name of Object.keys(values)) document.documentElement.style.removeProperty(name) }
  }, [appearance, mode])

  useEffect(() => {
    const objectURL = appearance?.objectURL
    return () => { if (objectURL) URL.revokeObjectURL(objectURL) }
  }, [appearance?.objectURL])

  return appearance
}
