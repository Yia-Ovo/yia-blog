import { act } from 'react'
import { createRoot } from 'react-dom/client'
import type { Root } from 'react-dom/client'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import HomeView from '../src/views/HomeView'
import { novel } from '../src/lib/novel'

vi.mock('../src/components/WeatherCard', () => ({ default: () => null }))
vi.mock('../src/components/DailyQuoteCard', () => ({ default: () => null }))

const chapter = novel.chapters[0]!
const records = [
  { id: chapter.id, title: chapter.title, category_id: 2 },
  { id: 77, title: 'Go 接口学习笔记', category_id: 1 },
  { id: 78, title: '小说创作随笔', category_id: 2 },
].map((record) => ({ ...record, cover: '', summary: '', is_top: 0, publish_at: null, author: null }))
const reply = (data: unknown) => new Response(JSON.stringify({ code: 200, msg: 'OK', err_msg: '', data }))
let container: HTMLDivElement
let root: Root
let router: ReturnType<typeof createMemoryRouter>
let queries: URLSearchParams[]

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true)
  localStorage.clear()
  queries = []
  vi.stubGlobal('fetch', vi.fn(async (input) => {
    const url = new URL(String(input), 'https://example.test')
    if (url.pathname.endsWith('/category/listCategory')) return reply({
      list: [{ id: 1, name: 'GoLang', sort: 0, status: 1 }, { id: 2, name: '小说', sort: 0, status: 1 }],
      total: 2, pages: 1, pageNum: 1, pageSize: 50,
    })
    if (!url.pathname.endsWith('/article/listArticle')) throw new Error(`Unexpected request: ${url.pathname}`)
    queries.push(url.searchParams)
    const category = Number(url.searchParams.get('category_id'))
    const excluded = (url.searchParams.get('exclude_ids') || '').split(',').map(Number)
    const title = url.searchParams.get('title') || ''
    const list = records.filter((record) => (!category || record.category_id === category) && !excluded.includes(record.id) && record.title.includes(title))
    return reply({ list, total: list.length, pages: 1, pageNum: 1, pageSize: 6 })
  }))
  container = document.createElement('div')
  document.body.append(container)
  root = createRoot(container)
})

afterEach(async () => {
  await act(async () => { root.unmount() })
  router?.dispose()
  container.remove()
  vi.unstubAllGlobals()
  localStorage.clear()
})

async function open(path = '/') {
  router = createMemoryRouter([{ path: '/', element: <HomeView /> }], { initialEntries: [path] })
  await act(async () => { root.render(<RouterProvider router={router} />) })
}

async function select(name: string) {
  const button = Array.from(container.querySelectorAll<HTMLButtonElement>('.filter-row button')).find((item) => item.textContent === name)!
  await act(async () => { button.click() })
}

describe('home category filtering with novel chapters', () => {
  it('includes registered chapters in their category and restores exclusion when returning to all articles', async () => {
    await open()
    expect(queries.at(-1)?.get('exclude_ids')?.split(',')).toContain(String(chapter.id))
    expect(container.querySelector('.article-list')?.textContent).not.toContain(chapter.title)
    await select('小说')
    expect(queries.at(-1)?.get('category_id')).toBe('2')
    expect(queries.at(-1)?.has('exclude_ids')).toBe(false)
    expect(container.querySelector('.article-list')?.textContent).toContain(chapter.title)
    expect(container.querySelector('.article-list')?.textContent).toContain('小说创作随笔')
    expect(container.querySelector('.article-list')?.textContent).not.toContain('Go 接口学习笔记')
    expect(container.querySelector('#feed-title .count-label')?.textContent).toBe('2 篇')
    await select('全部文章')
    expect(queries.at(-1)?.has('category_id')).toBe(false)
    expect(queries.at(-1)?.get('exclude_ids')?.split(',')).toContain(String(chapter.id))
    expect(container.querySelector('.article-list')?.textContent).not.toContain(chapter.title)
  })

  it('keeps chapters visible when opening a category URL directly', async () => {
    await open('/?category=2')
    expect(queries.at(-1)?.has('exclude_ids')).toBe(false)
    expect(container.querySelector('.article-list')?.textContent).toContain(chapter.title)
  })

  it('searches within the selected category without excluding chapters', async () => {
    await open('/?category=2&q=' + encodeURIComponent('门外'))
    expect(queries.at(-1)?.get('category_id')).toBe('2')
    expect(queries.at(-1)?.get('title')).toBe('门外')
    expect(queries.at(-1)?.has('exclude_ids')).toBe(false)
    expect(container.querySelector('.article-list')?.textContent).toContain(chapter.title)
    expect(container.querySelector('.article-list')?.textContent).not.toContain('小说创作随笔')
    expect(container.querySelector('#feed-title .count-label')?.textContent).toBe('1 篇')
  })
})
