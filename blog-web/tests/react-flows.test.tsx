import { act, StrictMode } from 'react'
import type { ReactNode } from 'react'
import { createRoot } from 'react-dom/client'
import type { Root } from 'react-dom/client'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useResource } from '../src/hooks/useResource'
import { clearSession, saveSession, sessions } from '../src/stores/session'
import { dismissNotice } from '../src/stores/notice'
import { emptyArticle } from '../src/lib/article'
import AuthView from '../src/views/AuthView'
import EditorView from '../src/views/EditorView'
import RequireSession from '../src/components/RequireSession'

let container: HTMLDivElement
let root: Root
let router: ReturnType<typeof createMemoryRouter> | undefined
const reply = (data: unknown = null, code = 200) => new Response(JSON.stringify({ code, msg: 'Request failed', err_msg: '', data }), { status: 200 })
const login = (token: string, role?: 0 | 1) => ({ token, expire: '', data: { id: 7, avatar: '', status: 1, nick_name: 'Yia', role } })

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (cause: unknown) => void
  const promise = new Promise<T>((success, failure) => { resolve = success; reject = failure })
  return { promise, resolve, reject }
}

async function render(node: ReactNode) { await act(async () => { root.render(node) }) }
async function change(selector: string, value: string) {
  const element = container.querySelector<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>(selector)!
  const prototype = element instanceof HTMLSelectElement ? HTMLSelectElement.prototype : element instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype
  await act(async () => {
    Object.getOwnPropertyDescriptor(prototype, 'value')!.set!.call(element, value)
    element.dispatchEvent(new Event(element instanceof HTMLSelectElement ? 'change' : 'input', { bubbles: true }))
  })
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => reply()))
  clearSession('user'); clearSession('admin'); localStorage.clear()
  container = document.createElement('div'); document.body.append(container)
  root = createRoot(container)
})
afterEach(async () => {
  await act(async () => { root.unmount() })
  router?.dispose(); router = undefined
  container.remove(); dismissNotice(); clearSession('user'); clearSession('admin')
  vi.unstubAllGlobals()
})

describe('React request lifecycle', () => {
  it('cancels old loads and refuses out-of-order responses in StrictMode', async () => {
    const first = deferred<string>()
    const second = deferred<string>()
    const signals: AbortSignal[] = []
    const loadFirst = (signal: AbortSignal) => { signals.push(signal); return first.promise }
    const loadSecond = () => second.promise
    function Result({ load }: { load: (signal: AbortSignal) => Promise<string> }) {
      const result = useResource(load)
      return <p>{result.pending ? 'loading' : result.error || result.data}</p>
    }
    await render(<StrictMode><Result load={loadFirst} /></StrictMode>)
    await render(<StrictMode><Result load={loadSecond} /></StrictMode>)
    expect(signals.every((signal) => signal.aborted)).toBe(true)
    expect(container.textContent).toBe('loading')
    await act(async () => { second.resolve('new result') })
    await act(async () => { first.resolve('old result') })
    expect(container.textContent).toBe('new result')
  })

  it('does not save a late user login after switching to the admin form', async () => {
    const pending = deferred<Response>()
    vi.mocked(fetch).mockReturnValue(pending.promise)
    router = createMemoryRouter([
      { path: '/login', element: <AuthView key="user" identity="user" /> },
      { path: '/admin/login', element: <AuthView key="admin" identity="admin" /> },
      { path: '/me/articles', element: <p>user workspace</p> },
    ], { initialEntries: ['/login'] })
    await render(<RouterProvider router={router} />)
    await change('input[autocomplete="username"]', 'Yia')
    await change('input[type="password"]', 'password12')
    await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
    expect(fetch).toHaveBeenCalledTimes(1)
    await act(async () => { await router!.navigate('/admin/login') })
    await act(async () => { pending.resolve(reply(login('late-user'))) })
    expect(sessions.user).toBeNull()
    expect(sessions.admin).toBeNull()
    expect(router.state.location.pathname).toBe('/admin/login')
  })
})

describe('React editor navigation and full updates', () => {
  const article = { ...emptyArticle(), id: 42, user_id: 7, admin_id: 0, title: '原来的草稿', content: '需要保留的正文', tag_ids: '4,5', comment_status: 0, publish_at: '2026-09-01T08:00:00Z', create_at: '', update_at: '' }

  async function openEditor(saveResponse = Promise.resolve(reply())) {
    saveSession('user', login('user-token'))
    vi.mocked(fetch).mockImplementation(async (url) => {
      if (String(url).includes('/category/')) return reply({ total: 0, pages: 0, list: [], pageNum: 1, pageSize: 100 })
      if (String(url).includes('getMyArticle')) return reply(article)
      if (String(url).includes('updateArticle')) return saveResponse
      throw new Error(`Unexpected request: ${String(url)}`)
    })
    router = createMemoryRouter([
      { path: '/write/:id', element: <EditorView identity="user" /> },
      { path: '/me/articles', element: <p>article list</p> },
    ], { initialEntries: ['/write/42'] })
    await render(<RouterProvider router={router} />)
  }

  it('keeps unsaved changes when leaving or changing articles is cancelled', async () => {
    await openEditor()
    await change('#article-title', '未保存的新标题')
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const beforeUnload = new Event('beforeunload', { cancelable: true })
    await act(async () => { window.dispatchEvent(beforeUnload) })
    expect(beforeUnload.defaultPrevented).toBe(true)
    await act(async () => { await router!.navigate('/write/43') })
    expect(confirm).toHaveBeenCalledOnce()
    expect(router!.state.location.pathname).toBe('/write/42')
    expect(container.querySelector<HTMLInputElement>('#article-title')?.value).toBe('未保存的新标题')
  })

  it('blocks leaving during save and navigates after success without a false dirty prompt', async () => {
    const saved = deferred<Response>()
    await openEditor(saved.promise)
    await change('.settings-card select', '1')
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
    await act(async () => { await router!.navigate('/me/articles') })
    expect(router!.state.location.pathname).toBe('/write/42')
    expect(confirm).not.toHaveBeenCalled()
    const saveCall = vi.mocked(fetch).mock.calls.find(([url]) => String(url).includes('updateArticle'))!
    expect(JSON.parse(String(saveCall[1]?.body))).toEqual({ ...emptyArticle(), id: 42, title: article.title, content: article.content, status: 1, tag_ids: '4,5', comment_status: 0, publish_at: article.publish_at })
    expect(saveCall[1]?.headers).toMatchObject({ Authorization: 'Bearer user-token' })
    await act(async () => { saved.resolve(reply()) })
    expect(router!.state.location.pathname).toBe('/me/articles')
    expect(confirm).not.toHaveBeenCalled()
  })

  it('keeps the edited text and exit protection after a failed save', async () => {
    await openEditor(Promise.resolve(reply(null, 500)))
    await change('#article-title', '保存失败也要保留')
    await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
    expect(container.querySelector('[role="alert"]')?.textContent).toContain('服务暂时不可用')
    expect(container.querySelector<HTMLInputElement>('#article-title')?.value).toBe('保存失败也要保留')
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    await act(async () => { await router!.navigate('/me/articles') })
    expect(router!.state.location.pathname).toBe('/write/42')
  })

  it('preserves the mounted editor when the save response expires its session', async () => {
    saveSession('user', login('user-token'))
    vi.mocked(fetch).mockImplementation(async (url) => {
      if (String(url).includes('/getInfo')) return reply(login('user-token').data)
      if (String(url).includes('/category/')) return reply({ total: 0, pages: 0, list: [], pageNum: 1, pageSize: 100 })
      if (String(url).includes('getMyArticle')) return reply(article)
      if (String(url).includes('updateArticle')) return reply(null, 401)
      throw new Error(`Unexpected request: ${String(url)}`)
    })
    router = createMemoryRouter([
      { path: '/write/:id', element: <RequireSession identity="user" preserveEditor><EditorView identity="user" /></RequireSession> },
      { path: '/login', element: <p>login page</p> },
    ], { initialEntries: ['/write/42'] })
    await render(<RouterProvider router={router} />)
    await change('#article-title', '令牌过期也不能丢失的修改')
    await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
    expect(sessions.user).toBeNull()
    expect(router.state.location.pathname).toBe('/write/42')
    expect(container.querySelector<HTMLInputElement>('#article-title')?.value).toBe('令牌过期也不能丢失的修改')
    expect(container.textContent).toContain('当前编辑内容仍保留')
    expect(container.querySelector<HTMLButtonElement>('button[type="submit"]')?.disabled).toBe(true)
    const beforeUnload = new Event('beforeunload', { cancelable: true })
    await act(async () => { window.dispatchEvent(beforeUnload) })
    expect(beforeUnload.defaultPrevented).toBe(true)
  })
})

describe('fresh profile checks', () => {
  it('does not trust a stored super-admin role', async () => {
    saveSession('admin', login('admin-token', 0))
    vi.mocked(fetch).mockResolvedValue(reply(login('admin-token', 1).data))
    router = createMemoryRouter([{ path: '/admin/categories', element: <RequireSession identity="admin" superOnly><p>category actions</p></RequireSession> }], { initialEntries: ['/admin/categories'] })
    await render(<RouterProvider router={router} />)
    expect(container.textContent).toContain('分类维护仅向超级管理员开放')
    expect(container.textContent).not.toContain('category actions')
    expect(sessions.admin?.profile.role).toBe(1)
  })

  it('keeps the token on a network failure and lets the user retry', async () => {
    saveSession('user', login('user-token'))
    vi.mocked(fetch).mockRejectedValueOnce(new TypeError('offline')).mockResolvedValueOnce(reply(login('user-token').data))
    router = createMemoryRouter([{ path: '/me/articles', element: <RequireSession identity="user"><p>my articles</p></RequireSession> }], { initialEntries: ['/me/articles'] })
    await render(<RouterProvider router={router} />)
    expect(container.querySelector('[role="alert"]')).not.toBeNull()
    expect(sessions.user?.token).toBe('user-token')
    await act(async () => { container.querySelector<HTMLButtonElement>('button')!.click() })
    expect(container.textContent).toContain('my articles')
  })
})
