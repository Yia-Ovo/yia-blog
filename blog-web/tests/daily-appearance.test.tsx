import { act, StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import type { Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { appearanceApi } from '../src/api/appearance'
import { loadArtwork } from '../src/lib/artwork'
import { useDailyAppearance } from '../src/hooks/useDailyAppearance'
import type { AppearanceMode } from '../src/lib/artwork'

vi.mock('../src/api/appearance', async importOriginal => ({
  ...await importOriginal<typeof import('../src/api/appearance')>(),
  appearanceApi: { today: vi.fn() },
}))
vi.mock('../src/lib/artwork', async importOriginal => ({
  ...await importOriginal<typeof import('../src/lib/artwork')>(), loadArtwork: vi.fn(),
}))

let root: Root
let container: HTMLDivElement
const revoke = vi.fn()
const metadata = (day: number) => ({
  date: `2026-09-${day}`, image_url: `/api/blog/v1/appearance/images/image-${day}.jpg`,
  server_time: `2026-09-${day}T23:59:58+08:00`,
  next_change_at: day === 30 ? '2026-10-01T00:00:00+08:00' : '2026-09-30T00:00:00+08:00',
})
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((done, fail) => { resolve = done; reject = fail })
  return { promise, resolve, reject }
}
function Probe({ mode = 'dark' }: { mode?: AppearanceMode }) {
  const appearance = useDailyAppearance(mode)
  return <><span>{appearance?.date || 'fallback'}</span><input defaultValue="未保存的正文" /></>
}
beforeEach(() => {
  vi.useFakeTimers()
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  const OriginalURL = URL
  vi.stubGlobal('URL', class extends OriginalURL { static revokeObjectURL = revoke })
  revoke.mockReset()
  vi.mocked(appearanceApi.today).mockReset().mockResolvedValue(metadata(29))
  vi.mocked(loadArtwork).mockReset().mockResolvedValue({ color: [40, 100, 180], objectURL: 'blob:first' })
  container = document.createElement('div'); document.body.append(container)
  root = createRoot(container)
})
afterEach(async () => {
  await act(async () => root.unmount())
  expect(document.documentElement.dataset.artworkReady).toBeUndefined()
  container.remove()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})
describe('daily appearance lifecycle', () => {
  it('reveals artwork only after both metadata and the image are ready', async () => {
    const daily = deferred<Awaited<ReturnType<typeof appearanceApi.today>>>()
    const image = deferred<Awaited<ReturnType<typeof loadArtwork>>>()
    vi.mocked(appearanceApi.today).mockReturnValueOnce(daily.promise)
    vi.mocked(loadArtwork).mockReturnValueOnce(image.promise)
    await act(async () => root.render(<Probe />))
    expect(document.documentElement.dataset.artworkReady).toBeUndefined()
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toBe('')
    expect(loadArtwork).not.toHaveBeenCalled()

    await act(async () => daily.resolve(metadata(29)))
    expect(loadArtwork).toHaveBeenCalledTimes(1)
    expect(document.documentElement.dataset.artworkReady).toBeUndefined()
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toBe('')

    const attributes: string[] = []
    const observer = new MutationObserver(records => {
      attributes.push(...records.map(record => record.attributeName!))
    })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['style', 'data-artwork-ready'] })
    await act(async () => image.resolve({ color: [40, 100, 180], objectURL: 'blob:decoded' }))
    observer.disconnect()
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toContain('blob:decoded')
    expect(attributes.indexOf('data-artwork-ready')).toBeGreaterThan(attributes.lastIndexOf('style'))
    expect(attributes.lastIndexOf('style')).toBeGreaterThanOrEqual(0)
  })

  it.each(['resolved', 'rejected'] as const)('ignores a %s API response from the discarded StrictMode effect', async result => {
    const discardedRequest = deferred<Awaited<ReturnType<typeof appearanceApi.today>>>()
    const currentImage = deferred<Awaited<ReturnType<typeof loadArtwork>>>()
    const discardedImage = deferred<Awaited<ReturnType<typeof loadArtwork>>>()
    vi.mocked(appearanceApi.today).mockReturnValueOnce(discardedRequest.promise).mockResolvedValueOnce(metadata(29))
    vi.mocked(loadArtwork).mockReturnValueOnce(currentImage.promise).mockReturnValueOnce(discardedImage.promise)

    await act(async () => root.render(<StrictMode><Probe /></StrictMode>))
    expect(appearanceApi.today).toHaveBeenCalledTimes(2)
    expect(loadArtwork).toHaveBeenCalledTimes(1)
    expect(document.documentElement.dataset.artworkReady).toBeUndefined()

    if (result === 'resolved') {
      await act(async () => discardedRequest.resolve(metadata(30)))
      expect(loadArtwork).toHaveBeenCalledTimes(2)
      expect(vi.mocked(loadArtwork).mock.calls[1]?.[1]?.aborted).toBe(true)
      expect(document.documentElement.dataset.artworkReady).toBeUndefined()
      await act(async () => discardedImage.resolve({ color: [180, 80, 40], objectURL: 'blob:discarded-round' }))
      expect(revoke).toHaveBeenCalledWith('blob:discarded-round')
    } else {
      await act(async () => discardedRequest.reject(new Error('late API failure')))
      expect(loadArtwork).toHaveBeenCalledTimes(1)
    }
    expect(document.documentElement.dataset.artworkReady).toBeUndefined()
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toBe('')

    await act(async () => currentImage.resolve({ color: [40, 100, 180], objectURL: 'blob:current-round' }))
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toContain('blob:current-round')
    expect(container.textContent).toBe('2026-09-29')
    expect(revoke).not.toHaveBeenCalledWith('blob:current-round')
  })

  it.each(['api', 'image'] as const)('reveals the fallback after initial %s failure and retries', async failure => {
    if (failure === 'api') vi.mocked(appearanceApi.today).mockRejectedValueOnce(new Error('offline'))
    else vi.mocked(loadArtwork).mockRejectedValueOnce(new Error('image unavailable'))

    await act(async () => root.render(<Probe />))
    expect(container.textContent).toBe('fallback')
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toBe('')
    expect(appearanceApi.today).toHaveBeenCalledTimes(1)

    await act(async () => { await vi.advanceTimersByTimeAsync(60_000) })
    expect(appearanceApi.today).toHaveBeenCalledTimes(2)
    expect(container.textContent).toBe('2026-09-29')
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toContain('blob:first')
  })

  it('changes image at the server midnight without replacing the open editor', async () => {
    vi.mocked(appearanceApi.today).mockResolvedValueOnce(metadata(29)).mockResolvedValue(metadata(30))
    vi.mocked(loadArtwork).mockResolvedValueOnce({ color: [40, 100, 180], objectURL: 'blob:first' }).mockResolvedValue({ color: [180, 80, 40], objectURL: 'blob:second' })
    await act(async () => root.render(<Probe />))
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    const input = container.querySelector('input')!
    input.value = '这段输入必须保留'
    await act(async () => { await vi.advanceTimersByTimeAsync(2001) })
    expect(container.textContent).toBe('2026-09-30')
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    expect(container.querySelector('input')).toBe(input)
    expect(input.value).toBe('这段输入必须保留')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toContain('blob:second')
    expect(revoke).toHaveBeenCalledWith('blob:first')
  })

  it('recolors light mode without reloading or revoking the current image', async () => {
    await act(async () => root.render(<Probe />))
    const dark = document.documentElement.style.getPropertyValue('--bg')
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    const readyChanges: MutationRecord[] = []
    const observer = new MutationObserver(records => { readyChanges.push(...records) })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-artwork-ready'] })
    await act(async () => root.render(<Probe mode="light" />))
    observer.disconnect()
    expect(document.documentElement.style.getPropertyValue('--bg')).not.toBe(dark)
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toContain('blob:first')
    expect(readyChanges).toHaveLength(0)
    expect(loadArtwork).toHaveBeenCalledTimes(1)
    expect(appearanceApi.today).toHaveBeenCalledTimes(1)
    expect(revoke).not.toHaveBeenCalled()
  })

  it('retains the previous appearance on failure and retries after one minute', async () => {
    vi.mocked(appearanceApi.today).mockResolvedValueOnce(metadata(29)).mockRejectedValueOnce(new Error('offline')).mockResolvedValue(metadata(30))
    vi.mocked(loadArtwork).mockResolvedValueOnce({ color: [40, 100, 180], objectURL: 'blob:first' }).mockResolvedValue({ color: [180, 80, 40], objectURL: 'blob:second' })
    await act(async () => root.render(<Probe />))
    await act(async () => { await vi.advanceTimersByTimeAsync(2001) })
    expect(container.textContent).toBe('2026-09-29')
    expect(document.documentElement.dataset.artworkReady).toBe('true')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toContain('blob:first')
    await act(async () => { await vi.advanceTimersByTimeAsync(60_000) })
    expect(container.textContent).toBe('2026-09-30')
  })

  it('discards a late image after unmounting and cancels future refreshes', async () => {
    let resolve!: (value: Awaited<ReturnType<typeof loadArtwork>>) => void
    vi.mocked(loadArtwork).mockReturnValue(new Promise(done => { resolve = done }))
    await act(async () => root.render(<Probe />))
    expect(document.documentElement.dataset.artworkReady).toBeUndefined()
    await act(async () => root.render(null))
    expect(document.documentElement.dataset.artworkReady).toBeUndefined()
    await act(async () => resolve({ color: [40, 100, 180], objectURL: 'blob:late' }))
    expect(revoke).toHaveBeenCalledWith('blob:late')
    expect(document.documentElement.dataset.artworkReady).toBeUndefined()
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toBe('')
    await act(async () => { await vi.advanceTimersByTimeAsync(86_400_000) })
    expect(appearanceApi.today).toHaveBeenCalledTimes(1)
  })
})
