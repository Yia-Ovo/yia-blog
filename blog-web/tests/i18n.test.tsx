import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from '../src/App'
import LanguageSwitcher from '../src/components/LanguageSwitcher'
import StatePanel from '../src/components/StatePanel'
import EditorView from '../src/views/EditorView'
import AuthView from '../src/views/AuthView'
import ArticleView from '../src/views/ArticleView'
import { getLocale, locales, localeStorageKey, messages, readSavedLocale, setLocale, t } from '../src/i18n'
import { formatDate } from '../src/lib/article'
import { request } from '../src/api/http'
import { clearSession, saveSession, sessions } from '../src/stores/session'
import { dismissNotice, notify } from '../src/stores/notice'

let container: HTMLDivElement
let root: Root
let router: ReturnType<typeof createMemoryRouter> | undefined
const reply = (data: unknown = null, code = 200, msg = 'Request failed') => new Response(JSON.stringify({ code, msg, err_msg: 'internal-secret', data }))
const login = (name: string) => ({ token: name + '-token', expire: '', data: { id: 7, avatar: '', status: 1, nick_name: name, admin_name: name, role: 0 as const } })

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(reply([])))
  localStorage.clear()
  setLocale('zh-CN')
  clearSession('user'); clearSession('admin')
  container = document.createElement('div'); document.body.append(container)
  root = createRoot(container)
})
afterEach(async () => {
  await act(async () => { root.unmount() })
  router?.dispose(); router = undefined
  container.remove()
  dismissNotice(); clearSession('user'); clearSession('admin')
  vi.restoreAllMocks(); vi.unstubAllGlobals()
  localStorage.clear(); setLocale('zh-CN')
})
async function render(element: React.ReactNode) { await act(async () => { root.render(element) }) }
async function switchTo(locale: typeof locales[number]) {
  const select = container.querySelector<HTMLSelectElement>('.language-switcher select')!
  expect(select).not.toBeNull()
  await act(async () => { select.value = locale; select.dispatchEvent(new Event('change', { bubbles: true })) })
}
async function input(selector: string, value: string) {
  const node = container.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector)!
  const prototype = node instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype
  await act(async () => {
    Object.getOwnPropertyDescriptor(prototype, 'value')!.set!.call(node, value)
    node.dispatchEvent(new Event('input', { bubbles: true }))
  })
}

describe('language preferences and messages', () => {
  it('has complete translations with the same interpolation fields in all languages', () => {
    const fields = (text: string) => [...text.matchAll(/\{(\w+)\}/g)].map((match) => match[1]).sort()
    expect(Object.keys(messages).length).toBeGreaterThan(200)
    for (const [source, values] of Object.entries(messages)) {
      for (const language of ['en', 'ja', 'vi'] as const) {
        expect(values[language].trim(), source + ' / ' + language).not.toBe('')
        expect(fields(values[language]), source + ' / ' + language).toEqual(fields(source))
      }
    }
  })

  it('remembers all four choices, updates HTML language and formats dates', async () => {
    await render(<LanguageSwitcher />)
    expect(Array.from(container.querySelectorAll('option')).map((node) => node.textContent)).toEqual(['中文', 'English', '日本語', 'Tiếng Việt'])
    for (const locale of locales) {
      await switchTo(locale)
      expect(document.documentElement.lang).toBe(locale)
      expect(readSavedLocale()).toBe(locale)
      expect(formatDate('2026-10-03T12:00:00')).toBe(new Intl.DateTimeFormat(locale, { year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date('2026-10-03T12:00:00')))
    }
    expect(t('登录')).toBe('Đăng nhập')
  })

  it('defaults safely for missing/invalid preferences and still switches when storage is blocked', async () => {
    localStorage.removeItem(localeStorageKey); expect(readSavedLocale()).toBe('zh-CN')
    localStorage.setItem(localeStorageKey, 'unsupported'); expect(readSavedLocale()).toBe('zh-CN')
    await render(<LanguageSwitcher />)
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new DOMException('blocked', 'SecurityError') })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new DOMException('blocked', 'SecurityError') })
    expect(readSavedLocale()).toBe('zh-CN')
    await switchTo('ja')
    expect(getLocale()).toBe('ja')
    expect(document.documentElement.lang).toBe('ja')
  })

  it('syncs other tabs and reset without reacting to unrelated storage', async () => {
    await render(<LanguageSwitcher />)
    await act(async () => { window.dispatchEvent(new StorageEvent('storage', { key: localeStorageKey, newValue: 'vi' })) })
    expect(getLocale()).toBe('vi')
    await act(async () => { window.dispatchEvent(new StorageEvent('storage', { key: 'yia.theme', newValue: 'light' })) })
    expect(getLocale()).toBe('vi')
    await act(async () => { window.dispatchEvent(new StorageEvent('storage', { key: null })) })
    expect(getLocale()).toBe('zh-CN')
    await act(async () => { window.dispatchEvent(new StorageEvent('storage', { key: localeStorageKey, newValue: 'invalid' })) })
    expect(getLocale()).toBe('zh-CN')
  })

  it('escapes message parameters through React and tolerates unknown keys', async () => {
    setLocale('en')
    const content = t('每日拾色 · {date}', { date: '<img src=x onerror=alert(1)>' })
    await render(<p>{content}</p>)
    expect(container.querySelector('img')).toBeNull()
    expect(container.textContent).toContain('<img')
    expect(t('Unknown fallback')).toBe('Unknown fallback')
    expect(t('__proto__')).toBe('__proto__')
    expect(t('constructor')).toBe('constructor')
  })

  it('translates an already displayed error after switching languages', async () => {
    await render(<><LanguageSwitcher /><StatePanel kind="error" message="登录已失效，请重新登录。" /></>)
    for (const locale of locales) {
      await switchTo(locale)
      expect(container.querySelector('[role="alert"] p')?.textContent).toBe(t('登录已失效，请重新登录。'))
    }
  })

  it('maps rate-limit and image-validation errors without revealing diagnostics', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(new Response('', { status: 429 }))
    let message = ''
    try { await request('/login') } catch (error) { message = (error as Error).message }
    expect(message).toBe('请求过于频繁，请稍后重试。')
    vi.mocked(fetch).mockResolvedValueOnce(reply(null, 400, '图片不能超过400万像素'))
    try { await request('/upload', { preferServerMessage: true }) } catch (error) { message = (error as Error).message }
    for (const locale of locales) {
      setLocale(locale)
      expect(t(message)).toBe(t('图片不能超过400万像素'))
      expect(t(message)).not.toContain('internal-secret')
    }
  })
})

describe('language switch preserves application state', () => {
  it('updates shell, title and notices without navigating or refetching', async () => {
    router = createMemoryRouter([{ element: <App />, children: [{ path: '/', element: <p>Original content</p>, handle: { title: '文章' } }] }], { initialEntries: ['/?q=keep#articles'] })
    await render(<RouterProvider router={router} />)
    const calls = vi.mocked(fetch).mock.calls.length
    await act(async () => { notify('已退出登录。') })
    for (const locale of locales) {
      await switchTo(locale)
      expect(document.title).toBe(t('文章') + ' · Yia')
      expect(container.querySelector('.toast')?.textContent).toContain(t('已退出登录。'))
      expect(router.state.location.search).toBe('?q=keep')
      expect(router.state.location.hash).toBe('#articles')
    }
    expect(fetch).toHaveBeenCalledTimes(calls)
  })

  it('preserves both sessions and unsaved editor content without remounting or saving', async () => {
    saveSession('user', login('写作者')); saveSession('admin', login('super_admin'))
    router = createMemoryRouter([{ path: '/write/new', element: <><LanguageSwitcher /><EditorView identity="user" /></> }], { initialEntries: ['/write/new'] })
    await render(<RouterProvider router={router} />)
    await input('#article-title', '登录')
    await input('#article-content', '原文 remains unchanged 日本語 Tiếng Việt')
    const editor = container.querySelector<HTMLTextAreaElement>('#article-content')!
    const calls = vi.mocked(fetch).mock.calls.length
    for (const locale of locales) {
      await switchTo(locale)
      expect(container.querySelector('#article-content')).toBe(editor)
      expect(editor.value).toBe('原文 remains unchanged 日本語 Tiếng Việt')
      expect(container.querySelector<HTMLInputElement>('#article-title')!.value).toBe('登录')
    }
    expect(fetch).toHaveBeenCalledTimes(calls)
    expect(sessions.user?.token).toBe('写作者-token')
    expect(sessions.admin?.token).toBe('super_admin-token')
  })

  it('keeps typed login values and the post-login destination', async () => {
    router = createMemoryRouter([{ path: '/login', element: <><LanguageSwitcher /><AuthView identity="user" /></> }], { initialEntries: ['/login?next=%2Farticles%2F42%23comments'] })
    await render(<RouterProvider router={router} />)
    await input('input[autocomplete="username"]', 'typed-user')
    await input('input[type="password"]', 'typed-password')
    for (const locale of locales) {
      await switchTo(locale)
      expect(container.querySelector<HTMLInputElement>('input[autocomplete="username"]')!.value).toBe('typed-user')
      expect(container.querySelector<HTMLInputElement>('input[type="password"]')!.value).toBe('typed-password')
      expect(router.state.location.search).toBe('?next=%2Farticles%2F42%23comments')
    }
    expect(fetch).not.toHaveBeenCalled()
  })
})


  it('keeps the article title and reading position across language changes, including trailing slashes', async () => {
    const scroll = vi.fn()
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', { configurable: true, value: scroll })
    vi.mocked(fetch).mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('getArticleById')) return reply({ id: 42, title: '登录', content: '原文 English 日本語 Tiếng Việt', summary: '', cover: '', category_id: 0, comment_status: 1, create_at: '2026-10-03T12:00:00', author: { name: '原作者' } })
      if (url.includes('listCategory')) return reply({ list: [], pages: 1 })
      if (url.includes('listComment')) return reply({ list: [], total: 0 })
      return reply(null)
    })
    router = createMemoryRouter([{ element: <App />, children: [{ path: '/articles/:id', element: <ArticleView />, handle: { title: '阅读' } }] }], { initialEntries: ['/articles/42/#comments'] })
    await render(<RouterProvider router={router} />)
    expect(document.title).toBe('登录 · Yia')
    const calls = vi.mocked(fetch).mock.calls.length
    const scrolls = scroll.mock.calls.length
    for (const locale of locales) {
      await switchTo(locale)
      expect(document.title).toBe('登录 · Yia')
      expect(container.querySelector('.reading-header h1')?.textContent).toBe('登录')
      expect(container.querySelector('.prose')?.textContent).toContain('原文 English 日本語 Tiếng Việt')
    }
    expect(scroll).toHaveBeenCalledTimes(scrolls)
    expect(fetch).toHaveBeenCalledTimes(calls)
    Reflect.deleteProperty(HTMLElement.prototype, 'scrollIntoView')
  })
