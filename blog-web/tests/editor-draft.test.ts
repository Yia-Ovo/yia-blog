import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { emptyArticle } from '../src/lib/article'
import {
  discardInvalidEditorDraft, editorDraftFingerprint, editorDraftKey, editorDraftOwner,
  EDITOR_DRAFT_MAX_BYTES, readEditorDraft, removeEditorDraft, writeEditorDraft,
} from '../src/lib/editorDraft'
import type { ArticleInput } from '../src/api/types'
import type { EditorDraftRecord } from '../src/lib/editorDraft'

const owner = editorDraftOwner('user', 7)!
const baseline = editorDraftFingerprint(emptyArticle())
const input = (content = '还没写完的正文') => ({ value: { ...emptyArticle(), content }, baseline })
const save = (content = '还没写完的正文', expected: string | null = null): EditorDraftRecord => {
  const result = writeEditorDraft(owner, input(content), expected)
  expect(result.status).toBe('saved')
  if (result.status !== 'saved') throw new Error(result.status)
  return result.record
}

beforeEach(() => { window.localStorage.clear() })
afterEach(() => { vi.restoreAllMocks() })

describe('editor draft storage', () => {
  it('isolates identity, account and article without depending on a session token', () => {
    const owners = [
      owner, editorDraftOwner('admin', 7)!, editorDraftOwner('user', 8)!,
      editorDraftOwner('user', 7, 11)!, editorDraftOwner('user', 7, 12)!,
    ]
    expect(new Set(owners.map(editorDraftKey)).size).toBe(owners.length)
    owners.forEach((target, index) => {
      expect(writeEditorDraft(target, input(String(index)), null).status).toBe('saved')
    })
    owners.forEach((target, index) => {
      expect(readEditorDraft(target)).toMatchObject({ status: 'found', record: { value: { content: String(index) } } })
    })
    expect(editorDraftKey(editorDraftOwner('user', 7)!)).toBe(editorDraftKey(owner))
    for (const id of [0, -1, NaN, 1.5, Number.MAX_SAFE_INTEGER + 1]) expect(editorDraftOwner('user', id)).toBeNull()
    expect(editorDraftOwner('user', 7, -1)).toBeNull()
  })

  it('round-trips unfinished input and zero values without backend article validation', () => {
    const value = { ...emptyArticle(), title: '', content: '', summary: '  尚未写完  ', is_top: 0, comment_status: 0, status: 2 as const, cover: '/api/blog/v1/images/example.png', publish_at: '2026-10-01T08:00:00Z' }
    const result = writeEditorDraft(owner, { value, baseline }, null)
    expect(result.status).toBe('saved')
    const loaded = readEditorDraft(owner)
    expect(loaded).toMatchObject({ status: 'found', record: { version: 1, baseline, value } })
    if (loaded.status === 'found') {
      expect(loaded.record.savedAt).toBeGreaterThan(0)
      expect(loaded.record.revision).toBeTruthy()
    }
  })

  it('uses a stable editable-field fingerprint and strips fields outside the whitelist', () => {
    const extra = { ...emptyArticle(), title: '  标题  ', user_id: 99, likes: 5 }
    expect(editorDraftFingerprint(extra)).toBe(editorDraftFingerprint({ ...emptyArticle(), title: '  标题  ' }))
    expect(editorDraftFingerprint(extra)).not.toBe(editorDraftFingerprint({ ...emptyArticle(), title: '标题' }))
    const result = writeEditorDraft(owner, { value: extra, baseline }, null)
    expect(result.status).toBe('saved')
    if (result.status === 'saved') {
      expect(result.record.value).not.toHaveProperty('user_id')
      expect(result.record.value).not.toHaveProperty('likes')
    }
  })

  it('distinguishes an absent record from malformed JSON or unsupported schema', () => {
    expect(readEditorDraft(owner)).toEqual({ status: 'none' })
    const record = save()
    const invalid = ['{', 'null', '[]', '{}', JSON.stringify({ ...record, version: 2 }), JSON.stringify({ ...record, savedAt: -1 }), JSON.stringify({ ...record, savedAt: Number.MAX_SAFE_INTEGER }), JSON.stringify({ ...record, revision: '' }), JSON.stringify({ ...record, baseline: '{}' })]
    for (const raw of invalid) {
      localStorage.setItem(editorDraftKey(owner), raw)
      expect(readEditorDraft(owner)).toEqual({ status: 'invalid' })
      expect(writeEditorDraft(owner, input('new'), null)).toEqual({ status: 'invalid' })
      expect(localStorage.getItem(editorDraftKey(owner))).toBe(raw)
    }
  })

  it.each([
    { title: null }, { content: 7 }, { tag_ids: [] }, { category_id: -1 }, { sort: 1.5 }, { sort: NaN },
    { status: '1' }, { status: 3 }, { is_top: 2 }, { comment_status: -1 },
    { cover: 'javascript:alert(1)' }, { cover: 'data:image/svg+xml,invalid' }, { publish_at: 'bad date' },
  ])('rejects invalid values without replacing the last safe draft: %o', (patch) => {
    const record = save()
    const invalid = { ...emptyArticle(), ...patch } as unknown as ArticleInput
    expect(writeEditorDraft(owner, { value: invalid, baseline }, record.revision)).toEqual({ status: 'invalid' })
    expect(readEditorDraft(owner)).toMatchObject({ status: 'found', record })
    const raw = JSON.stringify({ ...record, value: invalid })
    localStorage.setItem(editorDraftKey(owner), raw)
    expect(readEditorDraft(owner)).toEqual({ status: 'invalid' })
  })

  it('reports over-sized input and keeps the previous record', () => {
    const record = save()
    expect(writeEditorDraft(owner, input('中'.repeat(EDITOR_DRAFT_MAX_BYTES / 2)), record.revision)).toEqual({ status: 'too-large' })
    expect(readEditorDraft(owner)).toMatchObject({ status: 'found', record })
  })

  it('reports quota failure instead of claiming the draft was saved', () => {
    const record = save()
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new DOMException('Quota reached', 'QuotaExceededError') })
    expect(writeEditorDraft(owner, input('更新'), record.revision)).toEqual({ status: 'unavailable' })
    expect(readEditorDraft(owner)).toMatchObject({ status: 'found', record })
  })

  it('handles an inaccessible storage object and failing reads', () => {
    const getter = vi.spyOn(window, 'localStorage', 'get').mockImplementation(() => { throw new DOMException('Blocked', 'SecurityError') })
    expect(readEditorDraft(owner)).toEqual({ status: 'unavailable' })
    expect(writeEditorDraft(owner, input(), null)).toEqual({ status: 'unavailable' })
    expect(removeEditorDraft(owner, null)).toEqual({ status: 'unavailable' })
    getter.mockRestore()
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('Read failed') })
    expect(readEditorDraft(owner)).toEqual({ status: 'unavailable' })
  })

  it('blocks stale writes and stale cleanup after another tab saves', () => {
    const first = save('第一份')
    const second = save('另一个标签的新内容', first.revision)
    expect(second.revision).not.toBe(first.revision)
    expect(writeEditorDraft(owner, input('旧标签的内容'), first.revision)).toEqual({ status: 'conflict' })
    expect(writeEditorDraft(owner, input('没读取过的内容'), null)).toEqual({ status: 'conflict' })
    expect(removeEditorDraft(owner, first.revision)).toEqual({ status: 'conflict' })
    expect(readEditorDraft(owner)).toMatchObject({ status: 'found', record: second })
    expect(removeEditorDraft(owner, second.revision)).toEqual({ status: 'removed' })
    expect(readEditorDraft(owner)).toEqual({ status: 'none' })
    expect(writeEditorDraft(owner, input('stale'), second.revision)).toEqual({ status: 'conflict' })
  })

  it('detects a record replaced between the first read and write', () => {
    const first = save()
    const second = { ...first, revision: 'other-tab', value: { ...first.value, content: '保留其他标签页' } }
    const originalGet = Storage.prototype.getItem
    let reads = 0
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(function (this: Storage, key) {
      if (++reads === 2) this.setItem(key, JSON.stringify(second))
      return originalGet.call(this, key)
    })
    expect(writeEditorDraft(owner, input('stale'), first.revision)).toEqual({ status: 'conflict' })
    expect(readEditorDraft(owner)).toMatchObject({ status: 'found', record: second })
  })

  it('does not claim success when a write cannot be read back', () => {
    const record = save()
    const originalGet = Storage.prototype.getItem
    let reads = 0
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(function (this: Storage, key) {
      if (++reads === 3) throw new Error('Failed readback')
      return originalGet.call(this, key)
    })
    expect(writeEditorDraft(owner, input('new'), record.revision)).toEqual({ status: 'unavailable' })
  })

  it('reports failed deletion and preserves the current record', () => {
    const record = save()
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => { throw new Error('Denied') })
    expect(removeEditorDraft(owner, record.revision)).toEqual({ status: 'unavailable' })
    expect(readEditorDraft(owner)).toMatchObject({ status: 'found', record })
  })

  it('clears damaged records only explicitly and never removes a valid replacement', () => {
    localStorage.setItem(editorDraftKey(owner), '{broken')
    expect(removeEditorDraft(owner, null)).toEqual({ status: 'invalid' })
    expect(discardInvalidEditorDraft(owner)).toEqual({ status: 'removed' })
    expect(readEditorDraft(owner)).toEqual({ status: 'none' })
    const record = save()
    expect(discardInvalidEditorDraft(owner)).toEqual({ status: 'conflict' })
    expect(readEditorDraft(owner)).toMatchObject({ status: 'found', record })
    localStorage.setItem(editorDraftKey(owner), '{broken-again')
    const originalGet = Storage.prototype.getItem
    let reads = 0
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(function (this: Storage, key) {
      if (++reads === 2) this.setItem(key, JSON.stringify(record))
      return originalGet.call(this, key)
    })
    expect(discardInvalidEditorDraft(owner)).toEqual({ status: 'conflict' })
    expect(readEditorDraft(owner)).toMatchObject({ status: 'found', record })
  })
})
