import { act } from 'react'
import { createRoot } from 'react-dom/client'
import type { Root } from 'react-dom/client'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { articleApi } from '../src/api'
import type { PublicArticle } from '../src/api/types'
import NovelReadingMenu from '../src/components/NovelReadingMenu'
import { novel } from '../src/lib/novel'
import { setLocale } from '../src/i18n'

let root: Root
let host: HTMLDivElement
let router: ReturnType<typeof createMemoryRouter>
const item = (index: number) => ({ ...novel.chapters[index], author: null } as unknown as PublicArticle)
const page = (indices: number[]) => ({ list: indices.map(item), pages: 1, total: indices.length, pageNum: 1, pageSize: 100 })
const menu = () => document.querySelector<HTMLElement>('.reading-menu')!
const button = (name: string) => Array.from(menu().querySelectorAll('button')).find((node) => node.textContent?.trim() === name)!
async function click(node: HTMLElement) { await act(async () => { node.click() }) }
async function open(id = 3) {
  router = createMemoryRouter([{ path: '/', element: <NovelReadingMenu chapterID={id} /> }, { path: '/articles/:id', element: <p>chapter destination</p> }, { path: '/novels', element: <p>novel destination</p> }])
  await act(async () => { root.render(<RouterProvider router={router} />) })
}
async function scroll(y: number) {
  Object.defineProperty(window, 'scrollY', { configurable: true, value: y })
  await act(async () => { window.dispatchEvent(new Event('scroll')) })
}
beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: true }))
  vi.stubGlobal('scrollTo', vi.fn())
  vi.spyOn(articleApi, 'list').mockResolvedValue(page([0, 1, 2]))
  sessionStorage.clear(); setLocale('zh-CN')
  Object.defineProperty(window, 'scrollY', { configurable: true, value: 0 })
  host = document.createElement('div'); document.body.append(host); root = createRoot(host)
})
afterEach(async () => {
  await act(async () => root.unmount())
  router?.dispose(); host.remove(); sessionStorage.clear()
  vi.restoreAllMocks(); vi.unstubAllGlobals(); setLocale('zh-CN')
})

describe('floating chapter menu', () => {
  it('appears after scrolling, opens current chapter, dismisses with Escape and follows the next chapter', async () => {
    await open()
    expect(menu().dataset.visible).toBe('false')
    expect(menu().hasAttribute('inert')).toBe(true)
    await scroll(400)
    expect(menu().dataset.visible).toBe('true')
    await click(button('目录'))
    const current = menu().querySelector<HTMLAnchorElement>('[aria-current="page"]')!
    expect(current.getAttribute('href')).toBe('/articles/3')
    expect(document.activeElement).toBe(current)
    await act(async () => document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })))
    expect(menu().querySelector('[role="region"]')).toBeNull()
    expect(document.activeElement).toBe(button('目录'))
    await click(menu().querySelector<HTMLAnchorElement>('nav a[href="/articles/4"]')!)
    expect(router.state.location.pathname).toBe('/articles/4')
    expect(document.querySelector('.reading-menu')).toBeNull()
  })
  it('only links available chapters and disables both unavailable directions when the current chapter is absent', async () => {
    vi.mocked(articleApi.list).mockResolvedValue(page([0, 2]))
    await open(3); await scroll(400); await click(button('目录'))
    expect(menu().querySelectorAll('nav a')).toHaveLength(0)
    expect(menu().querySelectorAll('ol a')).toHaveLength(2)
    expect(menu().querySelector('a[href="/articles/3"]')).toBeNull()
  })
  it.each([[2, '上一章'], [4, '下一章']])('disables the unavailable edge at chapter %s', async (id, label) => {
    await open(Number(id)); await scroll(400)
    expect(button(String(label)).disabled).toBe(true)
    expect(menu().querySelectorAll('nav a')).toHaveLength(1)
  })
  it('shows a recoverable catalogue error without guessed links, and closes on outside interaction', async () => {
    vi.mocked(articleApi.list).mockRejectedValueOnce(new Error('offline'))
    await open(); await scroll(400); await click(button('目录'))
    expect(menu().querySelector('[role="alert"]')).not.toBeNull()
    expect(menu().querySelectorAll('nav a')).toHaveLength(0)
    await click(button('重新加载目录'))
    expect(menu().querySelectorAll('ol a')).toHaveLength(3)
    await act(async () => document.body.dispatchEvent(new Event('pointerdown', { bubbles: true })))
    expect(menu().querySelector('[role="region"]')).toBeNull()
  })
  it('uses reduced-motion scrolling and translates controls without fetching the catalogue again', async () => {
    await open(); await scroll(400)
    await act(async () => setLocale('en'))
    expect(button('Chapters')).toBeDefined()
    await click(button('Top'))
    expect(window.scrollTo).toHaveBeenCalledWith({ top: 0, behavior: 'instant' })
    expect(articleApi.list).toHaveBeenCalledTimes(1)
  })
  it('clamps invalid saved positions and supports keyboard movement while storage is unavailable', async () => {
    sessionStorage.setItem('yia.reading.menu.position.v1', JSON.stringify({ x: 99999, y: -999 }))
    await open(); await scroll(400)
    expect(parseFloat(menu().style.left)).toBeLessThan(window.innerWidth)
    expect(parseFloat(menu().style.top)).toBe(12)
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('disabled') })
    const handle = menu().querySelector<HTMLButtonElement>('.reading-menu-drag')!
    await act(async () => handle.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true })))
    expect(parseFloat(menu().style.left)).toBe(window.innerWidth - 82)
  })
})
