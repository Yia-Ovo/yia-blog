import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import AICodePanel from '../src/views/AICodePanel'
import { saveSession, clearSession } from '../src/stores/session'
import { setLocale } from '../src/i18n'

let container: HTMLDivElement, root: Root
let published = 0
const job = { id: 'code-task-00000001', prompt: '调整界面', status: 'review', context_mode: 'general', branch: 'codex/ai-code-task-00000001', diff: 'diff --git a/a.ts b/a.ts', checks: 'web: exit 0', tests_passed: true, review_digest: 'a'.repeat(64), summary: '', created_at: '', updated_at: '' }
beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true); setLocale('zh-CN'); published = 0
  saveSession('admin', { token: 'fixture', expire: '', data: { id: 1, role: 0, status: 1, avatar: '' } })
  vi.stubGlobal('fetch', vi.fn(async (url: string, options: RequestInit) => {
    if (url.endsWith('/job/publish')) { expect(JSON.parse(options.body as string).review_digest).toBe(job.review_digest); published++ }
    return new Response(JSON.stringify({ code: 200, data: url.endsWith('/workspace') ? { enabled: true, daily_limit: 2, jobs: [job] } : null }))
  }))
  container = document.createElement('div'); document.body.append(container); root = createRoot(container)
})
afterEach(async () => { await act(async () => root.unmount()); container.remove(); clearSession('admin'); vi.unstubAllGlobals() })

it('requires explicit review of the exact diff before publishing a branch', async () => {
  await act(async () => root.render(<AICodePanel contextMode="general" />))
  await act(async () => container.querySelector<HTMLButtonElement>('.ai-job-list button')!.click())
  const publish = Array.from(container.querySelectorAll('button')).find(button => button.textContent === '发布 GitHub 分支')!
  expect(publish.disabled).toBe(true); expect(published).toBe(0)
  await act(async () => container.querySelector<HTMLInputElement>('input[type="checkbox"]')!.click())
  expect(publish.disabled).toBe(false)
  await act(async () => publish.click())
  expect(published).toBe(1)
})
