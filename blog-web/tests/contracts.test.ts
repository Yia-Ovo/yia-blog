import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { articleApi, commentApi } from '../src/api'
import { ApiError, request } from '../src/api/http'
import { articleInput, emptyArticle, safeNext, validateArticle } from '../src/lib/article'
import { clearSession, saveSession, sessions } from '../src/stores/session'

const reply = (code = 200, data: unknown = null) => new Response(JSON.stringify({ code, msg: 'Request failed', err_msg: 'internal-database-secret', data }), { status: 200 })
const login = (token: string) => ({ token, expire: '', data: { id: 7, avatar: '', status: 1 } })
beforeEach(() => { clearSession('user'); clearSession('admin'); vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => reply())) })
afterEach(() => { vi.unstubAllGlobals() })

describe('API contract boundaries', () => {
  it('sends only the explicitly requested identity, and no tokens on public requests', async () => {
    saveSession('user', login('user-token')); saveSession('admin', login('admin-token'))
    await request('/public')
    await request('/author', { identity: 'user' })
    await request('/management', { identity: 'admin' })
    const calls = vi.mocked(fetch).mock.calls
    expect(calls[0]?.[1]?.headers).toEqual({ Accept: 'application/json' })
    expect(calls[1]?.[1]?.headers).toEqual({ Accept: 'application/json', Authorization: 'Bearer user-token' })
    expect(calls[2]?.[1]?.headers).toEqual({ Accept: 'application/json', 'Admin-Authorization': 'Bearer admin-token' })
  })

  it('does not treat HTTP 200 as business success or reveal internal diagnostics', async () => {
    vi.mocked(fetch).mockResolvedValue(reply(500))
    await expect(request('/failure')).rejects.toMatchObject({ code: 500 })
    try { await request('/failure') } catch (error) { expect((error as ApiError).message).not.toContain('internal-database-secret') }
  })

  it('invalidates only the rejected identity and leaves a later login intact', async () => {
    saveSession('user', login('old-user')); saveSession('admin', login('admin-token'))
    vi.mocked(fetch).mockImplementationOnce(async () => { saveSession('user', login('new-user')); return reply(401) })
    await expect(request('/old-request', { identity: 'user' })).rejects.toMatchObject({ code: 401 })
    expect(sessions.user?.token).toBe('new-user')
    vi.mocked(fetch).mockResolvedValueOnce(reply(401))
    await expect(request('/expired', { identity: 'user' })).rejects.toMatchObject({ code: 401 })
    expect(sessions.user).toBeNull()
    expect(sessions.admin?.token).toBe('admin-token')
  })

  it('keeps explicit zero query filters and uses the comment API names', async () => {
    await request('/filter', { query: { status: 0, is_top: 0, missing: undefined } })
    expect(String(vi.mocked(fetch).mock.calls[0]?.[0])).toContain('status=0&is_top=0')
    await commentApi.list(31, 2, 10)
    const url = String(vi.mocked(fetch).mock.calls[1]?.[0])
    expect(url).toContain('articleID=31&page=2&size=10')
    expect(url).not.toContain('article_id=')
  })

  it('publishes using all editable fields without author and counter columns', async () => {
    saveSession('user', login('user-token'))
    const loaded = { ...emptyArticle(), id: 9, user_id: 7, admin_id: 0, title: '原题', content: '未改的正文', tag_ids: '4,5', comment_status: 0, likes: 18, views: 52 }
    const input = articleInput(loaded)
    input.status = 1
    await articleApi.save('user', input, loaded.id)
    const [url, init] = vi.mocked(fetch).mock.calls[0]!
    const body = JSON.parse(String(init?.body))
    expect(String(url)).toContain('/article/updateArticle')
    expect(body).toMatchObject({ id: 9, title: '原题', content: '未改的正文', tag_ids: '4,5', status: 1, sort: 0, is_top: 0, comment_status: 0, category_id: 0, publish_at: null })
    for (const field of ['user_id', 'admin_id', 'likes', 'views']) expect(body).not.toHaveProperty(field)
  })

  it('submits exactly the comment that the moderator saw', async () => {
    saveSession('admin', login('admin-token'))
    await commentApi.review({ id: 3, expected_content: 'Original text ', expected_status: 0, status: 1 })
    const [, init] = vi.mocked(fetch).mock.calls[0]!
    expect(JSON.parse(String(init?.body))).toEqual({ id: 3, expected_content: 'Original text ', expected_status: 0, status: 1 })
  })
})

it('keeps post-login redirects inside the application', () => {
  for (const value of ['https://evil.example', '//evil.example', '/\\evil.example', '/\n/evil.example', '/\t/evil.example', '/login?next=/admin', '/admin/login', '/LOGIN']) expect(safeNext(value, '/me/articles')).toBe('/me/articles')
  expect(safeNext('/write/12', '/')).toBe('/write/12')
  expect(safeNext('/articles/42?page=2&query=hello%20world#comments', '/')).toBe('/articles/42?page=2&query=hello%20world#comments')
})

it('excludes authentication destinations after path normalization', () => {
  for (const value of ['/register', '/register?next=/articles/42', '/REGISTER/', '/%6Cogin', '/admin/%6Cogin', '/articles/../login', '/admin/../register', '/%72egister#form']) {
    expect(safeNext(value, '/me/articles'), value).toBe('/me/articles')
  }
})

it('validates UTF-8 title bytes and safe numeric fields without dropping legal zeros', () => {
  const form = { ...emptyArticle(), title: '你'.repeat(42), content: '正文', sort: 0, comment_status: 0 }
  expect(validateArticle(form)).toBe('')
  expect(validateArticle({ ...form, title: '你'.repeat(43) })).not.toBe('')
  for (const sort of [-1, 1.5, NaN, Number.MAX_SAFE_INTEGER + 1]) expect(validateArticle({ ...form, sort })).not.toBe('')
  const { content: _content, ...historical } = form
  expect(articleInput(historical).content).toBe('')
})
