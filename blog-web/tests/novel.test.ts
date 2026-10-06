import { afterEach, describe, expect, it, vi } from 'vitest'
import { articleApi } from '../src/api'
import { findChapter, loadChapters, novel, readingKey, readProgress, rememberChapter } from '../src/lib/novel'
import type { PublicArticle } from '../src/api/types'
const chapter = (index: number) => ({ ...novel.chapters[index], author: null } as unknown as PublicArticle)
afterEach(() => { vi.restoreAllMocks(); localStorage.clear() })
describe('public novel catalogue', () => {
  it('sorts registered chapters across pages, deduplicates and excludes unrelated or missing articles', async () => {
    const list = vi.spyOn(articleApi, 'list')
      .mockResolvedValueOnce({ list: [chapter(2), { ...chapter(0), id: 999 }], pages: 2, total: 4, pageNum: 1, pageSize: 2 })
      .mockResolvedValueOnce({ list: [chapter(0), chapter(2)], pages: 2, total: 4, pageNum: 2, pageSize: 2 })
    const controller = new AbortController()
    const result = await loadChapters(controller.signal)
    expect(result.map((item) => item.number)).toEqual([1, 3])
    expect(list.mock.calls[1]?.[0].page_num).toBe(2)
    expect(list.mock.calls[1]?.[1]).toBe(controller.signal)
    expect(findChapter({ ...chapter(0), title: 'Other article' })).toBeUndefined()
  })
  it('does not turn a failed catalogue into a partial success', async () => {
    vi.spyOn(articleApi, 'list').mockRejectedValue(new Error('offline'))
    await expect(loadChapters(new AbortController().signal)).rejects.toThrow('offline')
  })
  it('stores only recognised opened chapters and tolerates disabled or corrupt storage', () => {
    localStorage.setItem(readingKey, 'corrupt'); expect(readProgress()).toBeNull()
    rememberChapter(chapter(1)); expect(readProgress()).toBe(novel.chapters[1]!.id)
    rememberChapter({ id: 999, title: 'unrelated' }); expect(readProgress()).toBe(novel.chapters[1]!.id)
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('denied') })
    expect(() => rememberChapter(chapter(0))).not.toThrow()
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('denied') })
    expect(readProgress()).toBeNull()
  })
})
