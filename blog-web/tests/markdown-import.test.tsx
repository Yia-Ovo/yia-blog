import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import EditorView from '../src/views/EditorView'
import { emptyArticle } from '../src/lib/article'
import * as importer from '../src/lib/markdownImport'
import { clearSession, saveSession } from '../src/stores/session'
import { setLocale } from '../src/i18n'

const bytes = (text: string) => new TextEncoder().encode(text).buffer as ArrayBuffer
describe('Markdown file parsing', () => {
  it('reads UTF-8 files with BOM and CRLF without flattening code indentation', async () => {
    const result = await importer.readMarkdownFile(new File(['\uFEFF# 标题\r\n\r\n    code\r\n正文\r\n'], 'chapter.MD'))
    expect(result).toEqual({ filename: 'chapter.MD', title: '标题', content: '    code\n正文\n' })
  })
  it('uses filenames without treating headings inside code or metadata as the article title', () => {
    for (const source of ['\u0060\u0060\u0060md\n# code\n\u0060\u0060\u0060', '---\nstatus: published\n---\n# body']) {
      const value = importer.parseMarkdownFile('篇章.markdown', bytes(source))
      expect(value.title).toBe('篇章'); expect(value.content).toBe(source)
    }
    expect(importer.parseMarkdownFile('a.md', bytes('\n# Title ###\n\n正文')).title).toBe('Title')
  })
  it('rejects unsupported, empty, oversized, invalid UTF-8 and binary files', () => {
    expect(() => importer.parseMarkdownFile('a.exe', bytes('body'))).toThrow('.md')
    expect(() => importer.parseMarkdownFile('a.md', bytes('  '))).toThrow('正文')
    expect(() => importer.parseMarkdownFile('a.md', bytes('# title\n'))).toThrow('正文')
    expect(() => importer.parseMarkdownFile('a.md', new ArrayBuffer(importer.markdownImportLimit + 1))).toThrow('1 MB')
    expect(() => importer.parseMarkdownFile('a.md', new Uint8Array([0xff]).buffer)).toThrow('UTF-8')
    expect(() => importer.parseMarkdownFile('a.md', bytes('abc\0def'))).toThrow('非文本')
    expect(() => importer.parseMarkdownFile('a.md', bytes('# ' + '长'.repeat(43) + '\n正文'))).toThrow('标题太长')
  })
})

const reply = (data: unknown = null) => new Response(JSON.stringify({ code: 200, data }))
const login = (id = 7) => ({ token: 'token-' + id, expire: '', data: { id, avatar: '', status: 1, nick_name: 'Writer' } })
let root: Root
let box: HTMLDivElement
let router: ReturnType<typeof createMemoryRouter> | undefined

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.stubGlobal('fetch', vi.fn(async () => reply({ total: 0, pages: 0, list: [] })))
  clearSession('user'); clearSession('admin'); localStorage.clear(); setLocale('zh-CN')
  box = document.createElement('div'); document.body.append(box); root = createRoot(box)
})
afterEach(async () => {
  await act(async () => root.unmount())
  router?.dispose(); router = undefined; box.remove()
  clearSession('user'); clearSession('admin'); localStorage.clear()
  vi.restoreAllMocks(); vi.unstubAllGlobals()
})
async function open(identity: 'user' | 'admin' = 'user') {
  saveSession(identity, login())
  router = createMemoryRouter([{ path: '/edit', element: <EditorView identity={identity} /> },
    { path: '/me/articles', element: <p>saved</p> }, { path: '/admin/articles', element: <p>saved</p> }], { initialEntries: ['/edit'] })
  await act(async () => root.render(<RouterProvider router={router!} />))
}
async function change(selector: string, text: string) {
  const element = box.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector)!
  const proto = element instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype
  await act(async () => {
    Object.getOwnPropertyDescriptor(proto, 'value')!.set!.call(element, text)
    element.dispatchEvent(new Event('input', { bubbles: true }))
  })
}
async function choose() {
  const node = box.querySelector<HTMLInputElement>('input[aria-label="选择 Markdown 文件"]')!
  await act(async () => {
    Object.defineProperty(node, 'files', { configurable: true, value: [new File(['body'], 'chapter.md')] })
    node.dispatchEvent(new Event('change', { bubbles: true }))
  })
}
async function click(text: string) {
  const button = Array.from(box.querySelectorAll('button')).find((x) => x.textContent === text)!
  expect(button).toBeTruthy()
  await act(async () => button.click())
}
const imported = { filename: 'chapter.md', title: '导入标题', content: '导入正文' }

describe('Markdown import in both editors', () => {
  it.each(['user', 'admin'] as const)('requires explicit apply and preserves settings for %s', async (identity) => {
    vi.spyOn(importer, 'readMarkdownFile').mockResolvedValue(imported)
    await open(identity)
    await change('#article-title', '旧标题'); await change('#article-content', '旧内容')
    await change('#article-summary', '保留摘要'); await change('#article-cover', 'https://example.com/cover.png')
    await choose()
    expect(box.querySelector<HTMLInputElement>('#article-title')!.value).toBe('旧标题')
    expect(box.querySelector<HTMLTextAreaElement>('#article-content')!.value).toBe('旧内容')
    await click('取消导入')
    expect(box.textContent).not.toContain('替换标题和正文')
    await choose(); await click('替换标题和正文')
    expect(box.querySelector<HTMLInputElement>('#article-title')!.value).toBe(imported.title)
    expect(box.querySelector<HTMLTextAreaElement>('#article-content')!.value).toBe(imported.content)
    expect(vi.mocked(fetch).mock.calls.every(([path]) => !String(path).includes('createArticle'))).toBe(true)
    await act(async () => box.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
    const saved = vi.mocked(fetch).mock.calls.find(([path]) => /(?:createArticle|adminCreateArticle)/.test(String(path)))!
    expect(JSON.parse(String(saved[1]?.body))).toEqual({ ...emptyArticle(), title: imported.title, content: imported.content, summary: '保留摘要', cover: 'https://example.com/cover.png' })
    expect(saved[1]?.headers).toHaveProperty(identity === 'user' ? 'Authorization' : 'Admin-Authorization', 'Bearer token-7')
  })
  it('discards a cancelled read even if the file reader finishes later', async () => {
    let finish!: (result: importer.ImportedMarkdown) => void
    vi.spyOn(importer, 'readMarkdownFile').mockReturnValue(new Promise((resolve) => { finish = resolve }))
    await open(); await choose(); await click('取消导入')
    await act(async () => finish(imported))
    expect(box.textContent).not.toContain('使用这份内容')
    expect(box.querySelector<HTMLInputElement>('#article-title')!.value).toBe('')
  })
  it('does not apply a pending read to another account and sanitizes previews', async () => {
    let finish!: (result: importer.ImportedMarkdown) => void
    vi.spyOn(importer, 'readMarkdownFile').mockReturnValueOnce(new Promise((resolve) => { finish = resolve }))
    await open(); await choose()
    await act(async () => saveSession('user', login(9)))
    await act(async () => finish(imported))
    expect(box.querySelector<HTMLInputElement>('#article-title')!.value).toBe('')
    vi.mocked(importer.readMarkdownFile).mockResolvedValue({ ...imported, content: '<script>alert(1)</script><img src="x" onerror="alert(1)">正文' })
    await choose()
    expect(box.querySelector('.markdown-import-preview script')).toBeNull()
    expect(box.querySelector('.markdown-import-preview [onerror]')).toBeNull()
    await click('使用这份内容')
    expect(box.querySelector<HTMLTextAreaElement>('#article-content')!.value).toContain('<script>')
    // 保存的是原 Markdown；显示仍经过既有 sanitizer，不在导入时执行脚本。
  })
})
