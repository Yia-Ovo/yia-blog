import { artworkURL } from '../api/appearance'

// 图片地址校验供头像与首页复用，不依赖文章正文的 Markdown 解析库。
export function safeImageURL(value?: string | null): string {
  if (!value?.trim()) return ''
  try {
    if (value.trim().startsWith('/api/blog/v1/article/images/')) return artworkURL(value.trim())
    const url = new URL(value, window.location.origin)
    return ['http:', 'https:'].includes(url.protocol) ? url.href : ''
  } catch { return '' }
}
