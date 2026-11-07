import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import AIView from '../src/views/AIView'
import { clearSession, saveSession } from '../src/stores/session'
import { setLocale } from '../src/i18n'

let container: HTMLDivElement
let root: Root
let savedMemory = ''
let sent: Record<string, string> | null
let privateKnowledge: boolean
const completed = { id: 'conversation-00001', title: 'JWT 面试', update_at: '2026-11-07T08:00:00Z', messages: [
  { id: 'u1', role: 'user', status: 'completed', content: 'JWT', create_at: '', sources: [] },
  { id: 'a1', role: 'assistant', status: 'completed', content: '**账号级撤销**\n\n<img src="https://example.invalid/private" onerror="alert(1)"><script>alert(1)</script>', create_at: '', sources: [{ path: '.private/02-面试资料/README.md', line: 1, updated_at: '2026-11-07T08:00:00Z', excerpt: '进度：软删除与作者 ID' }] },
] }
const reply = (data: unknown, code = 200) => new Response(JSON.stringify({ code, data, msg: 'private' }))

beforeEach(() => {
  // jsdom 未实现原生对话框；浏览器验收另行检查焦点、Escape 和遮罩。
  Object.defineProperties(HTMLDialogElement.prototype, {
    showModal: { configurable: true, value: function (this: HTMLDialogElement) { this.open = true } },
    close: { configurable: true, value: function (this: HTMLDialogElement) { this.open = false } },
  })
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  clearSession('user'); clearSession('admin'); setLocale('zh-CN')
  saveSession('admin', { token: 'admin-test-token', expire: '', data: { id: 1, status: 1, role: 0, avatar: '' } })
  savedMemory = ''; sent = null; privateKnowledge = true
  vi.stubGlobal('fetch', vi.fn(async (url: string, options: RequestInit) => {
    expect((options.headers as Record<string, string>)['Admin-Authorization']).toBe('Bearer admin-test-token')
    expect(JSON.stringify(options.headers)).not.toContain('api_key')
    if (url.includes('/overview')) return reply({ model: 'gpt-6.1-sol', daily_limit: 50, used: sent ? 1 : 0, private_knowledge_enabled: privateKnowledge, knowledge_documents: privateKnowledge ? 12 : 0, memory: savedMemory, conversations: sent ? [{ ...completed, id: sent.conversation_id, messages: undefined }] : [] })
    if (url.includes('/memory')) { savedMemory = JSON.parse(options.body as string).memory; return reply(null) }
    if (url.includes('/imports')) return reply([])
    if (url.includes('/send')) { sent = JSON.parse(options.body as string); return reply({ ...completed, id: sent!.conversation_id, messages: [{ ...completed.messages[0] }, { ...completed.messages[1], content: '', status: 'pending' }] }) }
    if (url.includes('/conversation')) return reply({ ...completed, id: sent?.conversation_id || completed.id })
    return reply(null)
  }))
  container = document.createElement('div'); document.body.append(container); root = createRoot(container)
})
afterEach(async () => { await act(async () => root.unmount()); container.remove(); clearSession('admin'); clearSession('user'); vi.unstubAllGlobals(); vi.restoreAllMocks(); delete (HTMLDialogElement.prototype as Partial<HTMLDialogElement>).showModal; delete (HTMLDialogElement.prototype as Partial<HTMLDialogElement>).close })
async function open() { await act(async () => root.render(<MemoryRouter><AIView /></MemoryRouter>)) }
async function change(selector: string, text: string) {
  const element = container.querySelector<HTMLTextAreaElement>(selector)!
  await act(async () => { Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')!.set!.call(element, text); element.dispatchEvent(new Event('input', { bubbles: true })) })
}

it('provides a private chat and explicit memory that persists across reopened views', async () => {
  await open(); expect(container.textContent).toContain('不发送项目资料与项目记忆')
  await change('#ai-memory', '现在练习软删除与文章作者 ID')
  await act(async () => { Array.from(container.querySelectorAll('button')).find(button => button.textContent === '保存记忆')!.click() })
  expect(savedMemory).toBe('现在练习软删除与文章作者 ID')
  await act(async () => root.render(<MemoryRouter><AIView key="second-device" /></MemoryRouter>))
  expect(container.querySelector<HTMLTextAreaElement>('#ai-memory')!.value).toBe(savedMemory)
})

it('submits an idempotent request then reads the answer and source with safe Markdown', async () => {
  await open(); await change('#ai-question', 'JWT 为什么需要 token_version？')
  await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
  expect(sent).toMatchObject({ content: 'JWT 为什么需要 token_version？' })
  expect(sent!.request_id).toMatch(/^[a-z0-9-]{36}$/)
  expect(container.querySelector('.ai-answer strong')?.textContent).toBe('账号级撤销')
  expect(container.querySelector('.ai-answer img')).toBeNull()
  expect(container.querySelector('.ai-answer script')).toBeNull()
  expect(container.textContent).toContain('.private/02-面试资料/README.md:1')
  await act(async () => { Array.from(container.querySelectorAll('button')).find(button => button.textContent === '保存当前结论')!.click() })
  expect(savedMemory).toContain('账号级撤销')
  expect(container.querySelector<HTMLDialogElement>('.ai-memory')!.open).toBe(true)
})

it('keeps the same request and conversation IDs after the submit response is lost', async () => {
  const original = vi.mocked(fetch).getMockImplementation()!
  let attempts = 0; const requests: Record<string, string>[] = []
  vi.mocked(fetch).mockImplementation(async (url, options) => {
    if (String(url).includes('/send')) { requests.push(JSON.parse(options!.body as string)); if (++attempts === 1) throw new TypeError('connection lost') }
    return original(url, options)
  })
  await open(); await change('#ai-question', '解释 JWT')
  for (let i = 0; i < 2; i++) await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
  expect(requests).toHaveLength(2); expect(requests[1]).toEqual(requests[0])
})

it('does not request private data with only a regular user session', async () => {
  clearSession('admin'); saveSession('user', { token: 'user-token', expire: '', data: { id: 7, status: 1, avatar: '' } })
  await open(); expect(fetch).not.toHaveBeenCalled(); expect(container.textContent).toBe('')
})


it('shows general chat for a regular admin without owner study context', async () => {
  privateKnowledge = false
  saveSession('admin', { token: 'admin-test-token', expire: '', data: { id: 2, status: 1, role: 1, avatar: '' } })
  await open()
  expect(container.textContent).toContain('通用聊天 · 未关联站主私有资料')
  expect(container.textContent).not.toContain('已关联')
  expect(container.textContent).not.toContain('资料里的学习进度')
  expect(container.textContent).toContain('仅发送你的提问、对话和记忆')
  expect(container.textContent).not.toContain('同步本地历史')
  expect(vi.mocked(fetch).mock.calls.some(([url]) => String(url).includes('/import'))).toBe(false)
})

it('sends Enter, preserves Shift+Enter and input-method confirmation, and defaults to general context', async () => {
  await open(); await change('#ai-question', '面试问题')
  const text = container.querySelector<HTMLTextAreaElement>('#ai-question')!
  const submit = vi.spyOn(text.form!, 'requestSubmit').mockImplementation(() => { text.form!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
  for (const options of [{ shiftKey: true }, { isComposing: true }, { keyCode: 229 }]) {
    await act(async () => { text.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true, ...options })) })
  }
  expect(submit).not.toHaveBeenCalled(); expect(sent).toBeNull()
  await act(async () => { text.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true })) })
  expect(submit).toHaveBeenCalledOnce(); expect(sent).toMatchObject({ context_mode: 'general' })
})

it('reads imported history without sending it and requires an explicit project continuation', async () => {
  const original = vi.mocked(fetch).getMockImplementation()!
  vi.mocked(fetch).mockImplementation(async (url, options) => {
    if (String(url).includes('/imports')) return reply([{ id: 'import-00000000001', title: '本地项目讨论', updated_at: '2026-11-07T08:00:00Z' }])
    if (String(url).includes('/import?')) return reply({ id: 'import-00000000001', title: '本地项目讨论', updated_at: '2026-11-07T08:00:00Z', total_messages: 2, next_before: 0, messages: completed.messages })
    return original(url, options)
  })
  await open()
  await act(async () => { container.querySelector('.ai-import-rail nav button')!.dispatchEvent(new MouseEvent('click', { bubbles: true })) })
  expect(container.textContent).toContain('本地历史副本'); expect(sent).toBeNull()
  expect(container.querySelector('#ai-question')).not.toBeNull()
  expect(container.textContent).toContain('发送会带入这段历史')
  await change('#ai-question', '接着这个问题')
  await act(async () => { container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })) })
  expect(sent).toMatchObject({ context_mode: 'project', import_id: 'import-00000000001', content: '接着这个问题' })
})
