import { articleApi } from '../api'
import type { PublicArticle } from '../api/types'

// 登记已正式发布的章节；续写发布后更新清单，不按相似标题猜测归属。
export const novel = { title: '未归之人', author: 'Iris', chapters: [
  {
    "id": 2,
    "number": 1,
    "title": "《未归之人》第一章 门外不是楼道"
  },
  {
    "id": 3,
    "number": 2,
    "title": "《未归之人》第二章 水落下来之前"
  },
  {
    "id": 4,
    "number": 3,
    "title": "《未归之人》第三章 路牌上的字"
  },
  {
    "id": 5,
    "number": 4,
    "title": "《未归之人》第四章 你带不走"
  },
  {
    "id": 6,
    "number": 5,
    "title": "《未归之人》第五章 见一面也有价"
  },
  {
    "id": 7,
    "number": 6,
    "title": "《未归之人》第六章 别走正门"
  },
  {
    "id": 8,
    "number": 7,
    "title": "《未归之人》第七章 旧城的名字"
  },
  {
    "id": 9,
    "number": 8,
    "title": "《未归之人》第八章 门在身后"
  },
  {
    "id": 10,
    "number": 9,
    "title": "《未归之人》第九章 两个人的路"
  },
  {
    "id": 11,
    "number": 10,
    "title": "《未归之人》第十章 这件不卖"
  },
  {
    "id": 12,
    "number": 11,
    "title": "《未归之人》第十一章 天亮以前"
  },
  {
    "id": 13,
    "number": 12,
    "title": "《未归之人》第十二章 河上不收活人"
  },
  {
    "id": 14,
    "number": 13,
    "title": "《未归之人》第十三章 空出来的那一半"
  },
  {
    "id": 15,
    "number": 14,
    "title": "《未归之人》第十四章 水线以下"
  },
  {
    "id": 16,
    "number": 15,
    "title": "《未归之人》第十五章 名字不能划掉"
  },
  {
    "id": 17,
    "number": 16,
    "title": "《未归之人》第十六章 这一岸的人"
  },
  {
    "id": 18,
    "number": 17,
    "title": "《未归之人》第十七章 先让水过去"
  },
  {
    "id": 19,
    "number": 18,
    "title": "《未归之人》第十八章 水停下以后"
  },
  {
    "id": 20,
    "number": 19,
    "title": "《未归之人》第十九章 有人来领名字"
  },
  {
    "id": 21,
    "number": 20,
    "title": "《未归之人》第二十章 先往北走"
  }
] }
export const readingKey = 'yia.reading.weiguizhiren.v1'
export function findChapter(article: Pick<PublicArticle, 'id' | 'title'>) {
  return novel.chapters.find((chapter) => chapter.id === article.id && chapter.title === article.title)
}
export async function loadChapters(signal: AbortSignal) {
  const found = new Map<number, PublicArticle>()
  // 公开接口决定可见性，清单不能使已下架或删除的文章重新可读。
  for (let page = 1; page <= 20; page++) {
    const result = await articleApi.list({ title: '《未归之人》', page_num: page, page_size: 100 }, signal)
    for (const article of result.list) if (findChapter(article)) found.set(article.id, article)
    if (page >= result.pages) return novel.chapters.flatMap((chapter) => {
      const article = found.get(chapter.id)
      return article ? [{ ...chapter, article }] : []
    })
  }
  throw new Error('章节目录暂时无法加载，请稍后重试。')
}
export function readProgress(): number | null {
  try {
    const value = Number(localStorage.getItem(readingKey))
    return novel.chapters.some((chapter) => chapter.id === value) ? value : null
  } catch { return null }
}
export function rememberChapter(article: Pick<PublicArticle, 'id' | 'title'>) {
  if (!findChapter(article)) return
  try { localStorage.setItem(readingKey, String(article.id)) } catch { /* 禁用存储不阻止阅读。 */ }
}
