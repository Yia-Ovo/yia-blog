import { act } from 'react'
import { createRoot } from 'react-dom/client'
import type { Root } from 'react-dom/client'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Identity } from '../src/api/types'
import RequireSession from '../src/components/RequireSession'
import { dismissNotice } from '../src/stores/notice'
import { clearSession, saveSession, sessions } from '../src/stores/session'
import AuthView from '../src/views/AuthView'

let container: HTMLDivElement
let root: Root
let router: ReturnType<typeof createMemoryRouter> | undefined
const reply = (data: unknown = null, code = 200) => new Response(JSON.stringify({ code, msg: 'Request failed', data }), { status: 200 })
const login = (identity: Identity, token = identity + '-token') => ({
  token, expire: '', data: { id: identity === 'admin' ? 8 : 7, avatar: '', status: 1, nick_name: 'Yia', admin_name: 'super_admin' },
})

async function open(path: string) {
  router = createMemoryRouter([
    { path: '/login', element: <AuthView key="user" identity="user" /> },
    { path: '/register', element: <AuthView key="register" identity="user" register /> },
    { path: '/admin/login', element: <AuthView key="admin" identity="admin" /> },
    { path: '/me/articles', element: <RequireSession identity="user"><p>my articles</p></RequireSession> },
    { path: '/admin/articles', element: <RequireSession identity="admin"><p>admin articles</p></RequireSession> },
    { path: '/articles/:id', element: <p>public article and comments</p> },
  ], { initialEntries: [path] })
  await act(async () => { root.render(<RouterProvider router={router!} />) })
}

function button(text: string) {
  const result = Array.from(container.querySelectorAll('button')).find((element) => element.textContent?.trim() === text)
  expect(result, 'button: ' + text).toBeDefined()
  return result!
}

function link(text: string) {
  const result = Array.from(container.querySelectorAll('a')).find((element) => element.textContent?.trim() === text)
  expect(result, 'link: ' + text).toBeDefined()
  return result!
}

async function change(selector: string, value: string) {
  const element = container.querySelector<HTMLInputElement>(selector)!
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(element, value)
    element.dispatchEvent(new Event('input', { bubbles: true }))
  })
}

async function submit() {
  await change('input[autocomplete="username"]', 'Yia')
  await change('input[type="password"]', 'password12')
  await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(reply()))
  clearSession('user'); clearSession('admin')
  container = document.createElement('div'); document.body.append(container)
  root = createRoot(container)
})

afterEach(async () => {
  await act(async () => { root.unmount() })
  router?.dispose(); router = undefined
  container.remove(); dismissNotice(); clearSession('user'); clearSession('admin')
  vi.unstubAllGlobals()
})

describe('authentication pages with existing sessions', () => {
  it('keeps login and registration forms directly available when signed out', async () => {
    await open('/login')
    expect(container.querySelector('form')).not.toBeNull()
    expect(container.querySelector('.auth-session-card')).toBeNull()
    await act(async () => { await router!.navigate('/admin/login') })
    expect(container.querySelector('form')).not.toBeNull()
    expect(container.textContent).toContain('管理员名称')
    await act(async () => { await router!.navigate('/register') })
    expect(container.querySelectorAll('input[type="password"]')).toHaveLength(2)
  })

  it('replaces user login and registration forms with the saved user and a safe destination', async () => {
    saveSession('user', login('user'))
    await open('/login?next=%2Farticles%2F42%3Fpage%3D2%23comments')
    expect(container.querySelector('form')).toBeNull()
    expect(container.textContent).toContain('继续使用当前账号')
    expect(container.querySelector('.auth-session-card')?.textContent).toContain('Yia')
    expect(link('继续访问').getAttribute('href')).toBe('/articles/42?page=2#comments')
    await act(async () => { await router!.navigate('/register?next=%2Fregister') })
    expect(container.querySelector('form')).toBeNull()
    expect(link('继续访问').getAttribute('href')).toBe('/me/articles')
    expect(fetch).not.toHaveBeenCalled()
  })

  it('replaces the admin login form with the saved administrator', async () => {
    saveSession('admin', login('admin'))
    await open('/admin/login')
    expect(container.querySelector('form')).toBeNull()
    expect(container.querySelector('.auth-session-card')?.textContent).toContain('super_admin')
    expect(link('继续访问').getAttribute('href')).toBe('/admin/articles')
  })

  it('uses the requested identity when both sessions are present', async () => {
    saveSession('user', login('user')); saveSession('admin', login('admin'))
    await open('/login')
    expect(container.querySelector('.auth-session-card')?.textContent).toContain('Yia')
    expect(link('继续访问').getAttribute('href')).toBe('/me/articles')
    await act(async () => { await router!.navigate('/admin/login') })
    expect(container.querySelector('.auth-session-card')?.textContent).toContain('super_admin')
    expect(link('继续访问').getAttribute('href')).toBe('/admin/articles')
    expect(container.querySelector('form')).toBeNull()
  })

  it('requires an explicit choice to open user login or registration while an administrator is signed in', async () => {
    saveSession('admin', login('admin'))
    await open('/login?next=%2Farticles%2F42%23comments')
    expect(container.querySelector('form')).toBeNull()
    expect(link('进入管理员工作区').getAttribute('href')).toBe('/admin/articles')
    await act(async () => { button('登录普通用户账号').click() })
    expect(container.querySelector('form')).not.toBeNull()
    expect(container.querySelector('.inline-note')?.textContent).toContain('管理员')
    await act(async () => { await router!.navigate('/register') })
    expect(container.querySelector('form')).toBeNull()
    await act(async () => { button('注册普通用户账号').click() })
    expect(container.querySelectorAll('input[type="password"]')).toHaveLength(2)
    expect(sessions.admin?.token).toBe('admin-token')
    expect(sessions.user).toBeNull()
  })

  it('requires an explicit choice to open administrator login while a user is signed in', async () => {
    saveSession('user', login('user'))
    await open('/admin/login?next=%2Fadmin%2Farticles')
    expect(container.querySelector('form')).toBeNull()
    expect(link('进入我的创作').getAttribute('href')).toBe('/me/articles')
    await act(async () => { button('登录管理员账号').click() })
    expect(container.querySelector('form')).not.toBeNull()
    expect(container.querySelector('.inline-note')?.textContent).toContain('普通用户')
    expect(sessions.user?.token).toBe('user-token')
    expect(sessions.admin).toBeNull()
  })

  it('restores the form when the displayed session is logged out', async () => {
    saveSession('user', login('user'))
    await open('/login')
    expect(container.querySelector('form')).toBeNull()
    await act(async () => { clearSession('user', 'user-token') })
    expect(container.querySelector('form')).not.toBeNull()
    expect(container.querySelector('.auth-session-card')).toBeNull()
  })

  it('returns from an expired workspace session without looping or clearing the other identity', async () => {
    saveSession('user', login('user')); saveSession('admin', login('admin'))
    vi.mocked(fetch).mockResolvedValue(reply(null, 401))
    await open('/login')
    await act(async () => { link('继续访问').click() })
    expect(router!.state.location.pathname).toBe('/login')
    expect(sessions.user).toBeNull()
    expect(sessions.admin?.token).toBe('admin-token')
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(String(vi.mocked(fetch).mock.calls[0]?.[0])).toContain('/user/getInfo')
    await act(async () => { button('登录普通用户账号').click() })
    expect(container.querySelector('form')).not.toBeNull()
  })

  it('keeps the admin session and returns to article comments after explicit user login', async () => {
    saveSession('admin', login('admin'))
    vi.mocked(fetch).mockResolvedValue(reply(login('user')))
    await open('/login?next=%2Farticles%2F42%3Fpage%3D2%23comments')
    await act(async () => { button('登录普通用户账号').click() })
    await submit()
    expect(sessions.admin?.token).toBe('admin-token')
    expect(sessions.user?.token).toBe('user-token')
    expect(router!.state.location).toMatchObject({ pathname: '/articles/42', search: '?page=2', hash: '#comments' })
    const [url, options] = vi.mocked(fetch).mock.calls[0]!
    expect(String(url)).toContain('/user/login')
    expect(options?.headers).not.toHaveProperty('Admin-Authorization')
    expect(options?.headers).not.toHaveProperty('Authorization')
  })

  it('does not overwrite a new session with a login response resolved in the same update batch', async () => {
    let resolve!: (value: Response) => void
    const pending = new Promise<Response>((done) => { resolve = done })
    vi.mocked(fetch).mockReturnValue(pending)
    await open('/login')
    await submit()
    await act(async () => {
      saveSession('user', login('user', 'new-user-token'))
      resolve(reply(login('user', 'late-user-token')))
    })
    expect(sessions.user?.token).toBe('new-user-token')
    expect(router!.state.location.pathname).toBe('/login')
    expect(container.querySelector('form')).toBeNull()
    expect(container.querySelector('.auth-session-card')).not.toBeNull()
  })
})

