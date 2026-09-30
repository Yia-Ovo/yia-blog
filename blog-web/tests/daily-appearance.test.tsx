import { act } from 'react'
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
  container.remove()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})
describe('daily appearance lifecycle', () => {
  it('changes image at the server midnight without replacing the open editor', async () => {
    vi.mocked(appearanceApi.today).mockResolvedValueOnce(metadata(29)).mockResolvedValue(metadata(30))
    vi.mocked(loadArtwork).mockResolvedValueOnce({ color: [40, 100, 180], objectURL: 'blob:first' }).mockResolvedValue({ color: [180, 80, 40], objectURL: 'blob:second' })
    await act(async () => root.render(<Probe />))
    const input = container.querySelector('input')!
    input.value = '这段输入必须保留'
    await act(async () => { await vi.advanceTimersByTimeAsync(2001) })
    expect(container.textContent).toBe('2026-09-30')
    expect(container.querySelector('input')).toBe(input)
    expect(input.value).toBe('这段输入必须保留')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toContain('blob:second')
    expect(revoke).toHaveBeenCalledWith('blob:first')
  })

  it('recolors light mode without reloading or revoking the current image', async () => {
    await act(async () => root.render(<Probe />))
    const dark = document.documentElement.style.getPropertyValue('--bg')
    await act(async () => root.render(<Probe mode="light" />))
    expect(document.documentElement.style.getPropertyValue('--bg')).not.toBe(dark)
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
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toContain('blob:first')
    await act(async () => { await vi.advanceTimersByTimeAsync(60_000) })
    expect(container.textContent).toBe('2026-09-30')
  })

  it('discards a late image after unmounting and cancels future refreshes', async () => {
    let resolve!: (value: Awaited<ReturnType<typeof loadArtwork>>) => void
    vi.mocked(loadArtwork).mockReturnValue(new Promise(done => { resolve = done }))
    await act(async () => root.render(<Probe />))
    await act(async () => root.render(null))
    await act(async () => resolve({ color: [40, 100, 180], objectURL: 'blob:late' }))
    expect(revoke).toHaveBeenCalledWith('blob:late')
    expect(document.documentElement.style.getPropertyValue('--daily-artwork')).toBe('')
    await act(async () => { await vi.advanceTimersByTimeAsync(86_400_000) })
    expect(appearanceApi.today).toHaveBeenCalledTimes(1)
  })
})
