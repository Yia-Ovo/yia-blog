import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { WeatherResult } from '../src/api/weather'
import WeatherCard from '../src/components/WeatherCard'
import { setLocale } from '../src/i18n'

let container: HTMLDivElement
let root: Root
const cityKey = 'yia.weather.city.v1'
const reply = (data: WeatherResult) => new Response(JSON.stringify({ code: 200, msg: '', data }))
const ready = (city = '南宁', temperature = 27, source: 'ip' | 'manual' = 'ip'): WeatherResult => ({
  status: 'ready',
  location: { city, region: '', country: '', source },
  weather: { temperature, code: 0, is_day: true, humidity: 68, wind_speed: 12, minimum: 22, maximum: 29, updated_at: '2026-10-07T12:00:00Z' },
  stale: false,
})
function deferred() {
  let resolve!: (value: Response) => void
  const promise = new Promise<Response>((done) => { resolve = done })
  return { promise, resolve }
}
const picker = () => container.querySelector<HTMLSelectElement>('select')!
const button = (name: string) => Array.from(container.querySelectorAll('button')).find((node) => node.textContent?.trim() === name)!
const requestURL = (index: number) => new URL(String(vi.mocked(fetch).mock.calls[index]![0]), 'https://blog.example')
async function render() { await act(async () => { root.render(<WeatherCard />) }) }
async function selectCity(value: string) {
  await act(async () => { picker().value = value; picker().dispatchEvent(new Event('change', { bubbles: true })) })
}
async function click(node: HTMLButtonElement) { await act(async () => { node.click() }) }

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.stubGlobal('fetch', vi.fn(async () => reply(ready())))
  localStorage.clear(); setLocale('zh-CN')
  container = document.createElement('div'); document.body.append(container)
  root = createRoot(container)
})
afterEach(async () => {
  await act(async () => { root.unmount() })
  container.remove()
  vi.restoreAllMocks(); vi.unstubAllGlobals(); vi.useRealTimers()
  localStorage.clear(); setLocale('zh-CN')
})

describe('home weather card', () => {
  it('requests automatic IP weather without a city parameter and displays the returned location', async () => {
    const response = deferred()
    vi.mocked(fetch).mockReturnValueOnce(response.promise)
    await render()
    expect(container.querySelector('[role="status"]')?.textContent).toContain('正在识别大致城市…')
    expect(container.querySelector('.weather-temperature')).toBeNull()
    expect(requestURL(0).pathname).toBe('/api/blog/v1/weather/current')
    expect(requestURL(0).search).toBe('')
    expect(vi.mocked(fetch).mock.calls[0]![1]?.credentials).toBe('omit')
    await act(async () => { response.resolve(reply(ready('苏州', 18))) })
    expect(picker().value).toBe('')
    expect(container.querySelector('.weather-location')?.textContent).toContain('苏州IP 估算')
    expect(container.querySelector('.weather-temperature')?.textContent).toBe('18°C')
    expect(container.textContent).toContain('按网络出口估算，城市不对可手动切换。')
    expect(container.querySelector('a[href="https://ipwho.is/"]')).not.toBeNull()
  })

  it('loads the saved manual city, persists changes and can return to automatic detection', async () => {
    localStorage.setItem(cityKey, 'london')
    vi.mocked(fetch).mockResolvedValueOnce(reply(ready('London', 13, 'manual')))
    await render()
    expect(picker().value).toBe('london')
    expect(requestURL(0).searchParams.get('city')).toBe('london')
    expect(container.querySelector('.weather-location')?.textContent).toBe('伦敦手动选择')
    expect(container.querySelector('a[href="https://ipwho.is/"]')).toBeNull()
    await selectCity('shanghai')
    expect(requestURL(1).searchParams.get('city')).toBe('shanghai')
    expect(localStorage.getItem(cityKey)).toBe('shanghai')
    expect(container.querySelector('.weather-location strong')?.textContent).toBe('上海')
    await selectCity('')
    expect(requestURL(2).search).toBe('')
    expect(localStorage.getItem(cityKey)).toBe('')
    expect(container.querySelector('.weather-location')?.textContent).toContain('IP 估算')
  })

  it('falls back for an invalid preference and still changes city when storage is unavailable', async () => {
    localStorage.setItem(cityKey, 'unknown-city')
    await render()
    expect(picker().value).toBe('')
    expect(requestURL(0).search).toBe('')
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new DOMException('blocked', 'SecurityError') })
    await selectCity('tokyo')
    expect(requestURL(1).searchParams.get('city')).toBe('tokyo')
    expect(picker().value).toBe('tokyo')
    expect(container.querySelector('.weather-location strong')?.textContent).toBe('东京')
  })

  it('hides the previous city while loading, aborts superseded requests and ignores their late responses', async () => {
    await render()
    expect(container.querySelector('.weather-temperature')?.textContent).toBe('27°C')
    const beijing = deferred()
    const tokyo = deferred()
    vi.mocked(fetch).mockReturnValueOnce(beijing.promise).mockReturnValueOnce(tokyo.promise)
    await selectCity('beijing')
    const oldSignal = vi.mocked(fetch).mock.calls[1]![1]?.signal
    expect(oldSignal?.aborted).toBe(false)
    expect(container.querySelector('.weather-temperature')).toBeNull()
    expect(container.querySelector('[role="status"]')?.textContent).toContain('正在获取天气…')
    await selectCity('tokyo')
    expect(oldSignal?.aborted).toBe(true)
    await act(async () => { tokyo.resolve(reply(ready('Tokyo', 21, 'manual'))) })
    expect(container.querySelector('.weather-location strong')?.textContent).toBe('东京')
    expect(container.querySelector('.weather-temperature')?.textContent).toBe('21°C')
    // 模拟上游忽略取消信号，旧城市响应仍然晚到。
    await act(async () => { beijing.resolve(reply(ready('Beijing', 8, 'manual'))) })
    expect(picker().value).toBe('tokyo')
    expect(container.querySelector('.weather-location strong')?.textContent).toBe('东京')
    expect(container.querySelector('.weather-temperature')?.textContent).toBe('21°C')
  })

  it.each([
    ['location_unavailable', '暂时无法识别城市，请手动选择。'],
    ['weather_unavailable', '天气暂时不可用，可以换个城市或稍后重试。'],
    ['busy', '天气暂时不可用，可以换个城市或稍后重试。'],
  ] as const)('shows no invented temperature for %s and recovers on retry', async (reason, message) => {
    vi.mocked(fetch).mockResolvedValueOnce(reply({ status: 'unavailable', location: null, weather: null, stale: false, reason }))
    await render()
    expect(container.textContent).toContain(message)
    expect(container.querySelector('.weather-temperature')).toBeNull()
    expect(container.querySelector('.weather-metrics')).toBeNull()
    expect(container.textContent).not.toContain('°C')
    await click(button('重试'))
    expect(fetch).toHaveBeenCalledTimes(2)
    expect(requestURL(1).search).toBe('')
    expect(container.querySelector('.weather-temperature')?.textContent).toBe('27°C')
  })

  it('recovers from a failed weather refresh without retaining a stale temperature or exposing network details', async () => {
    localStorage.setItem(cityKey, 'beijing')
    await render()
    vi.mocked(fetch).mockRejectedValueOnce(new Error('internal-upstream-secret'))
    await click(button('刷新'))
    expect(container.querySelector('.weather-temperature')).toBeNull()
    expect(container.textContent).toContain('天气暂时不可用，可以换个城市或稍后重试。')
    expect(container.textContent).not.toContain('internal-upstream-secret')
    await click(button('重试'))
    expect(requestURL(2).searchParams.get('city')).toBe('beijing')
    expect(container.querySelector('.weather-location strong')?.textContent).toBe('北京')
    expect(container.querySelector('.weather-temperature')?.textContent).toBe('27°C')
  })

  it('translates the accessible heading, city picker and refresh control in all four languages without refetching', async () => {
    localStorage.setItem(cityKey, 'tokyo')
    await render()
    const select = picker()
    const translations = [
      ['zh-CN', '此刻天气', '切换天气城市', '自动识别城市', '东京', '刷新天气', '晴朗'],
      ['en', 'Weather now', 'Weather city', 'Automatic city', 'Tokyo', 'Refresh weather', 'Clear'],
      ['ja', 'いまの天気', '天気の都市を選択', '都市を自動判定', '東京', '天気を更新', '晴れ'],
      ['vi', 'Thời tiết lúc này', 'Chọn thành phố thời tiết', 'Tự nhận diện thành phố', 'Tokyo', 'Làm mới thời tiết', 'Trời quang'],
    ] as const
    for (const [locale, heading, label, automatic, city, refresh, condition] of translations) {
      await act(async () => { setLocale(locale) })
      const section = container.querySelector('section')!
      expect(container.querySelector(`#${section.getAttribute('aria-labelledby')}`)?.textContent).toBe(heading)
      expect(select.labels?.[0]?.querySelector('.sr-only')?.textContent).toBe(label)
      expect(select.querySelector('option[value=""]')?.textContent).toBe(automatic)
      expect(select.querySelector('option[value="tokyo"]')?.textContent).toBe(city)
      expect(container.querySelector('.weather-location strong')?.textContent).toBe(city)
      expect(container.querySelector('button[aria-label]')?.getAttribute('aria-label')).toBe(refresh)
      expect(container.querySelector('.weather-current p')?.textContent).toContain(condition)
      expect(picker()).toBe(select)
      expect(select.value).toBe('tokyo')
    }
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('does not present stale cached weather as current and recovers with fresh data', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(reply({ ...ready('南宁', 41), stale: true }))
    await render()
    expect(container.querySelector('.weather-temperature')).toBeNull()
    expect(container.querySelector('.weather-metrics')).toBeNull()
    expect(container.querySelector('.weather-credit')?.textContent).not.toContain('更新')
    expect(container.textContent).not.toContain('41°C')
    expect(container.querySelector('[aria-live="polite"]')?.textContent).toContain('天气暂时不可用')
    await click(button('重试'))
    expect(container.querySelector('.weather-temperature')?.textContent).toBe('27°C')
  })

  it.each([0, 1])('includes a date only when weather was updated on another local day (%s days ago)', async (daysAgo) => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 9, 7, 12))
    const timestamp = new Date(2026, 9, 7 - daysAgo, 10, 30)
    const data = ready()
    data.weather!.updated_at = timestamp.toISOString()
    vi.mocked(fetch).mockResolvedValueOnce(reply(data))
    await render()
    const options: Intl.DateTimeFormatOptions = { hour: '2-digit', minute: '2-digit' }
    if (daysAgo) { options.month = 'short'; options.day = 'numeric' }
    const expected = new Intl.DateTimeFormat('zh-CN', options).format(timestamp)
    expect(container.querySelector('.weather-credit > span:last-child')?.textContent).toBe(`${expected} 更新`)
  })

  it('keeps keyboard focus and the same live region across refresh, failure and retry without duplicate pending requests', async () => {
    await render()
    const action = container.querySelector<HTMLButtonElement>('.weather-actions button')!
    const announcement = container.querySelector('[aria-live="polite"]')!
    expect(announcement.getAttribute('aria-atomic')).toBe('true')
    action.focus()
    const refresh = deferred()
    vi.mocked(fetch).mockReturnValueOnce(refresh.promise)
    await click(action)
    expect(container.querySelector('.weather-actions button')).toBe(action)
    expect(document.activeElement).toBe(action)
    expect(action.getAttribute('aria-disabled')).toBe('true')
    expect(container.querySelector('[aria-live="polite"]')).toBe(announcement)
    expect(announcement.textContent).toBe('')
    await click(action)
    expect(fetch).toHaveBeenCalledTimes(2)
    await act(async () => { refresh.resolve(reply({ status: 'unavailable', location: null, weather: null, stale: false, reason: 'weather_unavailable' })) })
    expect(container.querySelector('.weather-actions button')).toBe(action)
    expect(document.activeElement).toBe(action)
    expect(action.textContent).toBe('重试')
    expect(action.getAttribute('aria-disabled')).toBe('false')
    expect(announcement.textContent).toContain('天气暂时不可用')
    const retry = deferred()
    vi.mocked(fetch).mockReturnValueOnce(retry.promise)
    await click(action)
    expect(document.activeElement).toBe(action)
    await act(async () => { retry.resolve(reply(ready())) })
    expect(container.querySelector('.weather-actions button')).toBe(action)
    expect(document.activeElement).toBe(action)
    expect(action.textContent).toBe('刷新')
    expect(action.getAttribute('aria-disabled')).toBe('false')
    expect(container.querySelector('[aria-live="polite"]')).toBe(announcement)
    expect(announcement.textContent).toBe('天气已更新')
  })
})
