import { act, StrictMode, useState } from 'react'
import { createRoot } from 'react-dom/client'
import type { Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import DraftRecoveryPanel from '../src/components/DraftRecoveryPanel'
import { useEditorDraft } from '../src/hooks/useEditorDraft'
import type { EditorDraftController } from '../src/hooks/useEditorDraft'
import { emptyArticle } from '../src/lib/article'
import { editorDraftFingerprint, editorDraftKey, readEditorDraft, writeEditorDraft } from '../src/lib/editorDraft'
import type { EditorDraftOwner } from '../src/lib/editorDraft'
import type { ArticleInput } from '../src/api/types'

let root: Root
let container: HTMLDivElement
let draft: EditorDraftController
let currentValue: ArticleInput
let change: (value: ArticleInput) => void
const owner: EditorDraftOwner = { identity: 'user', accountId: 7, articleId: 0 }

function Harness({ draftOwner, initial }: { draftOwner: EditorDraftOwner; initial: ArticleInput }) {
  const [value, setValue] = useState(initial)
  change = setValue
  currentValue = value
  draft = useEditorDraft({ owner: draftOwner, initial, value, onRestore: setValue })
  return <DraftRecoveryPanel draft={draft} />
}

function seed(value: ArticleInput, draftOwner = owner, initial = emptyArticle(), revision: string | null = null) {
  const result = writeEditorDraft(draftOwner, { value, baseline: editorDraftFingerprint(initial) }, revision)
  if (result.status !== 'saved') throw new Error(result.status)
  return result.record
}

function stored(draftOwner = owner) {
  const result = readEditorDraft(draftOwner)
  if (result.status !== 'found') throw new Error(result.status)
  return result.record
}

async function mount(draftOwner = owner, initial = emptyArticle()) {
  await act(async () => root.render(<StrictMode><Harness key={editorDraftKey(draftOwner)} draftOwner={draftOwner} initial={initial} /></StrictMode>))
}

async function edit(value: ArticleInput) {
  await act(async () => change(value))
}

async function click(label: string) {
  const button = Array.from(container.querySelectorAll('button')).find((item) => item.textContent === label)
  expect(button?.type).toBe('button')
  await act(async () => button!.click())
}

async function tick(milliseconds = 600) {
  await act(async () => vi.advanceTimersByTime(milliseconds))
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  vi.spyOn(window, 'confirm').mockReturnValue(true)
  localStorage.clear()
  container = document.createElement('div')
  document.body.append(container)
  root = createRoot(container)
})

afterEach(async () => {
  await act(async () => root.unmount())
  container.remove()
  vi.restoreAllMocks()
  localStorage.clear()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('editor draft recovery', () => {
  it('keeps an existing draft until the author explicitly restores or discards it', async () => {
    const previous = { ...emptyArticle(), content: '上次的未保存正文' }
    const record = seed(previous)
    await mount()
    expect(draft.status).toBe('recovery')
    await edit({ ...emptyArticle(), content: '当前新输入' })
    await tick(1200)
    await act(async () => window.dispatchEvent(new Event('pagehide')))
    expect(stored().revision).toBe(record.revision)
    expect(stored().value).toEqual(previous)
    await click('恢复暂存')
    expect(window.confirm).toHaveBeenCalledTimes(1)
    expect(currentValue).toEqual(previous)
    expect(draft.status).toBe('saved')
  })

  it('debounces complete unfinished input, including an empty title, then recovers after reopening', async () => {
    await mount()
    const value: ArticleInput = {
      title: '', content: '未写完的正文', cover: 'https://example.test/image.png', summary: '简介',
      category_id: 3, tag_ids: '1,2', status: 0, sort: 4, is_top: 1, comment_status: 0, publish_at: null,
    }
    await edit(value)
    await tick(599)
    expect(readEditorDraft(owner).status).toBe('none')
    await tick(1)
    expect(stored().value).toEqual(value)
    expect(container.textContent).toContain('已暂存到本机')
    expect(container.textContent).toContain('尚未保存到服务器')
    await act(async () => root.unmount())
    root = createRoot(container)
    await mount()
    expect(draft.status).toBe('recovery')
    await click('恢复暂存')
    expect(currentValue).toEqual(value)
    expect(stored().value).toEqual(value)
  })

  it('flushes the latest input on beforeunload, pagehide, hidden visibility and unmount', async () => {
    await mount()
    for (const eventName of ['beforeunload', 'pagehide']) {
      await edit({ ...emptyArticle(), content: eventName })
      await act(async () => window.dispatchEvent(new Event(eventName)))
      expect(stored().value.content).toBe(eventName)
    }
    await edit({ ...emptyArticle(), content: 'hidden' })
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden')
    await act(async () => document.dispatchEvent(new Event('visibilitychange')))
    expect(stored().value.content).toBe('hidden')
    await edit({ ...emptyArticle(), content: '离开前最后一次输入' })
    await act(async () => root.unmount())
    root = createRoot(container)
    expect(stored().value.content).toBe('离开前最后一次输入')
  })

  it('does not recreate a cleared draft during cleanup or a later input effect', async () => {
    await mount()
    await edit({ ...emptyArticle(), content: '待提交内容' })
    await tick()
    let cleared = false
    await act(async () => { cleared = draft.clearAfterSave() })
    expect(cleared).toBe(true)
    expect(readEditorDraft(owner).status).toBe('none')
    await edit({ ...emptyArticle(), content: '保存成功回调引起的表单更新' })
    await tick()
    await act(async () => window.dispatchEvent(new Event('pagehide')))
    await act(async () => root.unmount())
    root = createRoot(container)
    expect(readEditorDraft(owner).status).toBe('none')
  })

  it('preserves local content and another tab’s newer version when a storage event arrives', async () => {
    await mount()
    await edit({ ...emptyArticle(), content: '本标签页内容' })
    await tick()
    const latest = seed({ ...emptyArticle(), content: '另一标签页内容' }, owner, emptyArticle(), stored().revision)
    await act(async () => window.dispatchEvent(new StorageEvent('storage', { key: editorDraftKey(owner) })))
    expect(draft.status).toBe('conflict')
    expect(currentValue.content).toBe('本标签页内容')
    await edit({ ...emptyArticle(), content: '冲突后继续输入的内容' })
    await tick()
    await act(async () => draft.flush())
    expect(stored().revision).toBe(latest.revision)
    let cleared = true
    await act(async () => { cleared = draft.clearAfterSave() })
    expect(cleared).toBe(false)
    expect(stored().revision).toBe(latest.revision)
  })

  it('detects a write conflict even before a storage event is delivered', async () => {
    await mount()
    await edit({ ...emptyArticle(), content: '本机当前输入' })
    const remote = seed({ ...emptyArticle(), content: '其他标签先写入' })
    await tick()
    expect(draft.status).toBe('conflict')
    expect(currentValue.content).toBe('本机当前输入')
    expect(stored().revision).toBe(remote.revision)
  })

  it('shows storage failures and retries only after subsequent editing', async () => {
    await mount()
    const unavailable = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new DOMException('blocked', 'SecurityError') })
    await edit({ ...emptyArticle(), content: '无法保存的输入' })
    await tick()
    expect(draft.status).toBe('unavailable')
    expect(container.querySelector('[role="alert"]')?.textContent).toContain('尚未暂存')
    expect(readEditorDraft(owner).status).toBe('none')
    unavailable.mockRestore()
    await edit({ ...emptyArticle(), content: '重试后的输入' })
    await tick()
    expect(draft.status).toBe('saved')
    expect(stored().value.content).toBe('重试后的输入')
  })

  it('retains an initial storage-unavailable warning and reports unsuccessful cleanup', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new DOMException('blocked', 'SecurityError') })
    await mount()
    expect(draft.status).toBe('unavailable')
    expect(container.textContent).toContain('本机暂存不可用')
    let cleared = true
    await act(async () => { cleared = draft.clearAfterSave() })
    expect(cleared).toBe(false)
  })

  it('leaves malformed data untouched until discard, then saves current editor content', async () => {
    localStorage.setItem(editorDraftKey(owner), '{invalid')
    await mount()
    expect(draft.status).toBe('invalid')
    await edit({ ...emptyArticle(), content: '新输入' })
    await tick()
    expect(localStorage.getItem(editorDraftKey(owner))).toBe('{invalid')
    await click('放弃暂存')
    await tick()
    expect(stored().value.content).toBe('新输入')
  })

  it('does not discard a valid version that replaced malformed data in another tab', async () => {
    localStorage.setItem(editorDraftKey(owner), '{invalid')
    await mount()
    localStorage.removeItem(editorDraftKey(owner))
    const remote = seed({ ...emptyArticle(), content: '另一标签的新版本' })
    await click('放弃暂存')
    expect(draft.status).toBe('conflict')
    expect(stored().revision).toBe(remote.revision)
  })

  it('isolates identity, account and article while flushing the original owner on remount', async () => {
    const owners: EditorDraftOwner[] = [
      owner,
      { ...owner, identity: 'admin' },
      { ...owner, accountId: 8 },
      { ...owner, articleId: 9 },
    ]
    for (const [index, item] of owners.entries()) {
      await mount(item)
      expect(draft.recovery).toBeNull()
      await edit({ ...emptyArticle(), content: `记录${index}` })
    }
    await act(async () => window.dispatchEvent(new Event('pagehide')))
    owners.forEach((item, index) => expect(stored(item).value.content).toBe(`记录${index}`))
  })

  it('warns about a changed server baseline before and after recovery', async () => {
    const original = { ...emptyArticle(), title: '服务器旧标题', content: '旧正文' }
    seed({ ...original, content: '本机继续写的正文' }, owner, original)
    const updated = { ...original, content: '服务器更新的正文' }
    await mount(owner, updated)
    expect(container.textContent).toContain('服务器上的文章与暂存时不同')
    await click('恢复暂存')
    expect(currentValue.content).toBe('本机继续写的正文')
    expect(container.textContent).toContain('服务器上的文章与暂存时不同')
  })

  it('keeps the previous snapshot when a later edit is too large to store', async () => {
    await mount()
    await edit({ ...emptyArticle(), content: '可以恢复的旧正文' })
    await tick()
    const previous = stored()
    await edit({ ...emptyArticle(), content: 'x'.repeat(2 * 1024 * 1024) })
    await tick()
    expect(draft.status).toBe('too-large')
    expect(container.querySelector('[role="alert"]')?.textContent).toContain('当前修改尚未暂存')
    expect(stored().revision).toBe(previous.revision)
    expect(currentValue.content.length).toBe(2 * 1024 * 1024)
  })

  it('does not report pending after undoing back to the already stored value', async () => {
    await mount()
    const value = { ...emptyArticle(), content: '已暂存正文' }
    await edit(value)
    await tick()
    const previous = stored()
    await edit({ ...value, content: '准备撤销的输入' })
    expect(draft.status).toBe('pending')
    await edit(value)
    expect(draft.status).toBe('saved')
    await tick()
    expect(stored().revision).toBe(previous.revision)
  })

  it('removes its own snapshot when edits return to the initial server content', async () => {
    const initial = { ...emptyArticle(), title: '文章标题', content: '服务器正文' }
    await mount(owner, initial)
    await edit({ ...initial, content: '临时改动' })
    await tick()
    expect(readEditorDraft(owner).status).toBe('found')
    await edit(initial)
    expect(readEditorDraft(owner).status).toBe('none')
    expect(draft.status).toBe('idle')
  })
})
