import { act } from 'react'
import { createRoot } from 'react-dom/client'
import type { Root } from 'react-dom/client'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from '../src/App'
import Avatar from '../src/components/Avatar'
import RequireSession from '../src/components/RequireSession'
import ProfileView from '../src/views/ProfileView'
import { authApi } from '../src/api'
import { clearSession, saveSession, sessions } from '../src/stores/session'
import type { Identity } from '../src/api/types'

vi.mock('../src/hooks/useDailyAppearance', () => ({ useDailyAppearance: () => null }))

let root: Root
let container: HTMLDivElement
let router: ReturnType<typeof createMemoryRouter> | undefined
const revoke = vi.fn()
const create = vi.fn()
const reply = (data: unknown = null, code = 200) => new Response(JSON.stringify({ code, msg: 'Request failed', err_msg: '', data }))
const login = (token: string, avatar = '') => ({ token, expire: '', data: { id: 7, avatar, status: 1, nick_name: 'Yia', admin_name: 'super_admin' } })
const photo = (name = 'portrait.jpg') => new File(['image bytes'], name, { type: 'image/jpeg' })

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { promise, resolve }
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => reply({ avatar: '/api/blog/v1/avatar/images/new.png' })))
  const NativeURL = URL
  vi.stubGlobal('URL', class extends NativeURL { static createObjectURL = create; static revokeObjectURL = revoke })
  let sequence = 0
  create.mockReset().mockImplementation(() => `blob:avatar-${++sequence}`)
  revoke.mockReset()
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  clearSession('user'); clearSession('admin')
  container = document.createElement('div'); document.body.append(container); root = createRoot(container)
})

afterEach(async () => {
  await act(async () => root.unmount())
  router?.dispose(); router = undefined
  container.remove(); clearSession('user'); clearSession('admin')
  vi.restoreAllMocks(); vi.unstubAllGlobals(); vi.unstubAllEnvs()
})

async function openProfile(identity: Identity = 'admin', withSession = true) {
  if (withSession) saveSession(identity, login(`${identity}-token`))
  const path = identity === 'admin' ? '/admin/profile' : '/me/profile'
  router = createMemoryRouter([{ element: <App />, children: [
    { path, element: <ProfileView identity={identity} /> },
    { path: '/', element: <p>Home</p> },
  ] }], { initialEntries: [path] })
  await act(async () => root.render(<RouterProvider router={router!} />))
}

async function choose(file: File, load = true) {
  const input = container.querySelector<HTMLInputElement>('#avatar-file')!
  await act(async () => {
    Object.defineProperty(input, 'files', { configurable: true, value: [file] })
    input.dispatchEvent(new Event('change', { bubbles: true }))
  })
  if (load) await act(async () => container.querySelector('img[alt="新头像预览"]')?.dispatchEvent(new Event('load')))
}

async function submit() {
  await act(async () => container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
}

describe('avatar upload', () => {
  it('uses multipart without a Content-Type override and sends only the requested identity', async () => {
    saveSession('user', login('user-token')); saveSession('admin', login('admin-token'))
    const file = photo()
    await authApi.uploadAvatar('user', file)
    await authApi.uploadAvatar('admin', file)
    const calls = vi.mocked(fetch).mock.calls
    expect(String(calls[0]?.[0])).toContain('/user/uploadAvatar')
    expect(calls[0]?.[1]?.headers).toEqual({ Accept: 'application/json', Authorization: 'Bearer user-token' })
    expect(calls[1]?.[1]?.headers).toEqual({ Accept: 'application/json', 'Admin-Authorization': 'Bearer admin-token' })
    expect(calls[0]?.[1]?.body).toBeInstanceOf(FormData)
    expect((calls[0]?.[1]?.body as FormData).get('file')).toBe(file)
  })

  it('rejects unsupported, empty and oversized files before uploading', async () => {
    await openProfile()
    await choose(new File(['svg'], 'fake.svg', { type: 'image/svg+xml' }))
    expect(container.querySelector('[role="alert"]')?.textContent).toContain('JPG 或 PNG')
    await choose(new File([], 'empty.png', { type: 'image/png' }))
    expect(container.querySelector('[role="alert"]')?.textContent).toContain('不能为空')
    const large = photo(); Object.defineProperty(large, 'size', { value: 5 * 1024 * 1024 + 1 })
    await choose(large)
    expect(container.querySelector('[role="alert"]')?.textContent).toContain('5 MB')
    await submit()
    expect(fetch).not.toHaveBeenCalled()
    expect(create).not.toHaveBeenCalled()
  })

  it('prevents duplicate saves, persists the avatar and immediately refreshes the header', async () => {
    const saved = deferred<Response>()
    vi.mocked(fetch).mockReturnValue(saved.promise)
    await openProfile()
    await choose(photo())
    await submit(); await submit()
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(container.querySelector('form')?.getAttribute('aria-busy')).toBe('true')
    await act(async () => saved.resolve(reply({ avatar: '/api/blog/v1/avatar/images/new.png' })))
    expect(container.querySelector('[role="status"]')?.textContent).toContain('头像已更新')
    expect(sessions.admin?.profile.avatar).toBe('/api/blog/v1/avatar/images/new.png')
    expect(JSON.parse(sessionStorage.getItem('yia.session.admin')!).profile.avatar).toBe('/api/blog/v1/avatar/images/new.png')
    expect(container.querySelector('.account-profile img')?.getAttribute('src')).toContain('/api/blog/v1/avatar/images/new.png')
    expect(revoke).toHaveBeenCalledWith('blob:avatar-1')
  })

  it('retains the selected preview on failure and allows a successful retry', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(reply(null, 500)).mockResolvedValueOnce(reply({ avatar: '/api/blog/v1/avatar/images/retry.png' }))
    await openProfile('user')
    await choose(photo())
    await submit()
    expect(container.querySelector('[role="alert"]')?.textContent).toContain('服务暂时不可用')
    expect(container.querySelector('img[alt="新头像预览"]')).not.toBeNull()
    expect(sessions.user?.profile.avatar).toBe('')
    await submit()
    expect(sessions.user?.profile.avatar).toContain('retry.png')
  })

  it('shows the public image validation message without displaying internal details', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({ code: 400, msg: '图片不能超过400万像素', err_msg: 'internal image path', data: null })))
    await openProfile()
    await choose(photo())
    await submit()
    expect(container.querySelector('[role="alert"]')?.textContent).toBe('图片不能超过400万像素')
    expect(container.textContent).not.toContain('internal image path')
  })

  it('ignores a late upload after a new login and revokes replaced previews', async () => {
    const pending = deferred<Response>()
    vi.mocked(fetch).mockReturnValue(pending.promise)
    await openProfile()
    await choose(photo('first.jpg')); await choose(photo('second.jpg'))
    expect(revoke).toHaveBeenCalledWith('blob:avatar-1')
    await submit()
    await act(async () => saveSession('admin', login('new-login', '/new-account.png')))
    expect(revoke).toHaveBeenCalledWith('blob:avatar-2')
    await act(async () => pending.resolve(reply({ avatar: '/old-account.png' })))
    expect(sessions.admin?.profile.avatar).toBe('/new-account.png')
    expect(container.querySelector('[role="status"]')).toBeNull()
  })

  it('requires the corresponding session and does not accept the other identity', async () => {
    saveSession('user', login('user-token'))
    router = createMemoryRouter([
      { path: '/admin/profile', element: <RequireSession identity="admin"><ProfileView identity="admin" /></RequireSession> },
      { path: '/admin/login', element: <p>Admin login</p> },
    ], { initialEntries: ['/admin/profile'] })
    await act(async () => root.render(<RouterProvider router={router!} />))
    expect(router.state.location.pathname).toBe('/admin/login')
    expect(fetch).not.toHaveBeenCalled()
  })

  it('shows an admin profile on public pages, with the user taking priority when both are logged in', async () => {
    await openProfile()
    await act(async () => { await router!.navigate('/') })
    expect(container.querySelector('.account-profile')?.getAttribute('href')).toBe('/admin/profile')
    expect(container.querySelector('.write-link')?.getAttribute('href')).toBe('/admin/articles/new')
    await act(async () => saveSession('user', login('user-token')))
    expect(container.querySelector('.account-profile')?.getAttribute('href')).toBe('/me/profile')
    expect(container.querySelector('.write-link')?.getAttribute('href')).toBe('/write/new')
  })
})

it('resolves stored avatars against the API origin and falls back after an image load error', async () => {
  vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.test/api/blog/v1')
  await act(async () => root.render(<Avatar url="/api/blog/v1/avatar/images/picture.png" name="Yia" />))
  expect(container.querySelector('img')?.getAttribute('src')).toBe('https://api.example.test/api/blog/v1/avatar/images/picture.png')
  await act(async () => container.querySelector('img')!.dispatchEvent(new Event('error')))
  expect(container.querySelector('img')).toBeNull()
  expect(container.textContent).toBe('Y')
  await act(async () => root.render(<Avatar url="/api/blog/v1/avatar/images/next.png" name="Yia" />))
  expect(container.querySelector('img')?.getAttribute('src')).toContain('next.png')
})
