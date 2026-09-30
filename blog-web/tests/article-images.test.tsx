import { act } from 'react'
import { createRoot } from 'react-dom/client'
import type { Root } from 'react-dom/client'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import EditorView from '../src/views/EditorView'
import { articleApi } from '../src/api'
import { emptyArticle } from '../src/lib/article'
import { insertArticleImage } from '../src/lib/articleImages'
import { renderMarkdown, safeImageURL } from '../src/lib/markdown'
import { clearSession, saveSession } from '../src/stores/session'

const url = '/api/blog/v1/article/images/' + 'a'.repeat(64) + '.png'
const image = { url, width: 1200, height: 800 }
const reply = (data: unknown = null, code = 200) => new Response(JSON.stringify({ code, msg: 'Request failed', err_msg: '', data }))
const login = (id = 7) => ({ token: 'token-' + id, expire: '', data: { id, avatar: '', status: 1, nick_name: 'Writer' } })
const photo = () => new File(['png bytes'], '工程截图.png', { type: 'image/png' })
let root: Root
let container: HTMLDivElement
let router: ReturnType<typeof createMemoryRouter>

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { promise, resolve }
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async (path) => {
    if (String(path).includes('/category/')) return reply({ total: 0, pages: 0, list: [], pageNum: 1, pageSize: 100 })
    if (String(path).includes('UploadImage') || String(path).includes('uploadImage')) return reply(image)
    return reply()
  }))
  clearSession('user'); clearSession('admin'); localStorage.clear()
  container = document.createElement('div'); document.body.append(container)
  root = createRoot(container)
})

afterEach(async () => {
  await act(async () => root.unmount())
  router?.dispose()
  container.remove(); clearSession('user'); clearSession('admin'); localStorage.clear()
  vi.restoreAllMocks(); vi.unstubAllGlobals(); vi.unstubAllEnvs()
})

async function open() {
  saveSession('user', login())
  router = createMemoryRouter([
    { path: '/write/new', element: <EditorView identity="user" /> },
    { path: '/me/articles', element: <p>article list</p> },
  ], { initialEntries: ['/write/new'] })
  await act(async () => root.render(<RouterProvider router={router} />))
}

async function change(selector: string, value: string) {
  const element = container.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector)!
  const proto = element instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype
  await act(async () => {
    Object.getOwnPropertyDescriptor(proto, 'value')!.set!.call(element, value)
    element.dispatchEvent(new Event('input', { bubbles: true }))
  })
}

async function choose(files: File[], target = '选择封面图片') {
  const input = container.querySelector<HTMLInputElement>('input[aria-label="' + target + '"]')!
  await act(async () => {
    Object.defineProperty(input, 'files', { configurable: true, value: files })
    input.dispatchEvent(new Event('change', { bubbles: true }))
  })
}

describe('article images in the editor', () => {
  it('sends a single multipart file using only the selected identity', async () => {
    saveSession('user', login()); saveSession('admin', login(8))
    await articleApi.uploadImage('user', photo())
    await articleApi.uploadImage('admin', photo())
    const calls = vi.mocked(fetch).mock.calls
    expect(String(calls[0]![0])).toContain('/article/uploadImage')
    expect(calls[0]![1]?.headers).toEqual({ Accept: 'application/json', Authorization: 'Bearer token-7' })
    expect(String(calls[1]![0])).toContain('/article/adminUploadImage')
    expect(calls[1]![1]?.headers).toEqual({ Accept: 'application/json', 'Admin-Authorization': 'Bearer token-8' })
    expect(calls[0]![1]?.body).toBeInstanceOf(FormData)
    expect(Array.from((calls[0]![1]?.body as FormData).keys())).toEqual(['file'])
  })

  it('sets an uploaded cover and sends its portable path in the complete saved article', async () => {
    await open()
    await change('#article-title', '图片文章'); await change('#article-content', '正文')
    await choose([photo()])
    expect(container.querySelector<HTMLInputElement>('#article-cover')?.value).toBe(url)
    expect(container.querySelector<HTMLImageElement>('.cover-preview')?.src).toBe(new URL(url, location.origin).href)
    await act(async () => container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
    const saved = vi.mocked(fetch).mock.calls.find(([path]) => String(path).includes('createArticle'))!
    expect(JSON.parse(String(saved[1]?.body))).toEqual({ ...emptyArticle(), title: '图片文章', content: '正文', cover: url })
    expect(router.state.location.pathname).toBe('/me/articles')
    expect(Object.keys(localStorage).filter((key) => key.startsWith('yia-blog:editor-draft:v1:'))).toEqual([])
  })

  it('keeps typing during paste uploads and prevents saving an unfinished upload', async () => {
    const pending = deferred<Response>()
    await open()
    await change('#article-title', '粘贴测试'); await change('#article-content', '原始内容')
    vi.mocked(fetch).mockReturnValue(pending.promise)
    const textarea = container.querySelector<HTMLTextAreaElement>('#article-content')!
    textarea.setSelectionRange(2, 2)
    const paste = new Event('paste', { bubbles: true, cancelable: true })
    Object.defineProperty(paste, 'clipboardData', { value: { files: [photo()] } })
    await act(async () => textarea.dispatchEvent(paste))
    expect(paste.defaultPrevented).toBe(true)
    expect(container.querySelector<HTMLButtonElement>('button[type="submit"]')?.disabled).toBe(true)
    await change('#article-content', '上传期间继续写作')
    await act(async () => container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
    expect(vi.mocked(fetch).mock.calls.some(([path]) => String(path).includes('createArticle'))).toBe(false)
    await act(async () => pending.resolve(reply(image)))
    expect(textarea.value).toBe('上传期间继续写作\n\n![工程截图](' + url + ')\n\n')
    expect(container.querySelector<HTMLButtonElement>('button[type="submit"]')?.disabled).toBe(false)
  })

  it('inserts a dropped image at the unchanged cursor without removing selected text', async () => {
    await open(); await change('#article-content', '前文后文')
    const textarea = container.querySelector<HTMLTextAreaElement>('#article-content')!
    textarea.setSelectionRange(0, 2)
    const drop = new Event('drop', { bubbles: true, cancelable: true })
    Object.defineProperty(drop, 'dataTransfer', { value: { files: [photo()] } })
    await act(async () => textarea.dispatchEvent(drop))
    expect(drop.defaultPrevented).toBe(true)
    expect(textarea.value).toBe('前文\n\n![工程截图](' + url + ')\n\n后文')
  })

  it('rejects unsupported, oversized and multiple files before making requests', async () => {
    await open()
    const count = vi.mocked(fetch).mock.calls.length
    await choose([new File(['<svg/>'], 'bad.svg', { type: 'image/svg+xml' })])
    expect(container.textContent).toContain('请选择 JPG 或 PNG')
    const large = photo(); Object.defineProperty(large, 'size', { value: 5 * 1024 * 1024 + 1 })
    await choose([large]); expect(container.textContent).toContain('不能超过 5 MB')
    await choose([photo(), photo()]); expect(container.textContent).toContain('一次上传一张')
    expect(vi.mocked(fetch).mock.calls.length).toBe(count)
  })

  it('preserves content and allows retry after a failed or malformed upload', async () => {
    await open(); await change('#article-content', '保留正文')
    vi.mocked(fetch).mockResolvedValueOnce(reply(null, 500))
    await choose([photo()], '选择正文图片')
    expect(container.textContent).toContain('服务暂时不可用')
    expect(container.querySelector<HTMLTextAreaElement>('#article-content')?.value).toBe('保留正文')
    vi.mocked(fetch).mockResolvedValueOnce(reply({ ...image, url: 'javascript:alert(1)' }))
    await choose([photo()], '选择正文图片')
    expect(container.textContent).toContain('图片响应无效')
    await choose([photo()], '选择正文图片')
    expect(container.querySelector<HTMLTextAreaElement>('#article-content')?.value).toContain(url)
  })

  it('cancels old uploads and separates the editor when the account changes', async () => {
    await open(); await change('#article-content', '账号七的正文')
    const pending = deferred<Response>()
    const normalFetch = vi.mocked(fetch).getMockImplementation()!
    vi.mocked(fetch).mockImplementation((path, init) => String(path).includes('uploadImage') ? pending.promise : normalFetch(path, init))
    await choose([photo()])
    const call = vi.mocked(fetch).mock.calls.find(([path]) => String(path).includes('uploadImage'))!
    await act(async () => saveSession('user', login(8)))
    expect(call[1]?.signal?.aborted).toBe(true)
    await act(async () => pending.resolve(reply(image)))
    expect(container.querySelector<HTMLInputElement>('#article-cover')?.value).toBe('')
    expect(container.querySelector<HTMLTextAreaElement>('#article-content')?.value).toBe('')
  })

  it('resolves stored image paths against a separate API origin in previews and Markdown', () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/api/blog/v1')
    expect(safeImageURL(url)).toBe('https://api.example.com' + url)
    expect(renderMarkdown('![示例](' + url + ')')).toContain('src="https://api.example.com' + url + '"')
    expect(insertArticleImage('文字', image, 'bad]\\\nname.png', { content: '文字', position: 2 })).not.toContain('bad]')
  })
})
