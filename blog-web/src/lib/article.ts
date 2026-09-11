import type { ArticleInput } from '../api/types'

export function emptyArticle(): ArticleInput {
  return { title: '', cover: '', summary: '', content: '', category_id: 0, tag_ids: '', status: 0, sort: 0, is_top: 0, comment_status: 1, publish_at: null }
}

/** 显式字段白名单避免把详情中的作者、计数或时间戳带回更新接口。 */
export function articleInput(value: Omit<ArticleInput, 'content'> & { content?: string }): ArticleInput {
  return {
    title: value.title, cover: value.cover, summary: value.summary, content: value.content ?? '',
    category_id: value.category_id, tag_ids: value.tag_ids, status: value.status, sort: value.sort,
    is_top: value.is_top, comment_status: value.comment_status, publish_at: value.publish_at,
  }
}

export function validateArticle(value: ArticleInput): string {
  if (!value.title.trim()) return '请先给文章起个标题。'
  if (new TextEncoder().encode(value.title.trim()).length > 128) return '标题太长，请适当缩短（中文约 42 个字）。'
  if (!value.content.trim()) return '请填写文章正文，草稿也需要有内容。'
  if (!Number.isSafeInteger(value.sort) || value.sort < 0) return '排序值需要是非负整数。'
  if (!Number.isSafeInteger(value.category_id) || value.category_id < 0) return '请选择有效的分类。'
  return ''
}

export const articleStatuses = ['草稿', '已发布', '已下架'] as const
export const commentStatuses = ['待审核', '已通过', '已拒绝'] as const

export function positiveNumber(value: unknown, fallback = 1): number {
  const number = Number(value)
  return Number.isSafeInteger(number) && number > 0 ? number : fallback
}

export function formatDate(value?: string | null): string {
  if (!value) return '未标注日期'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '未标注日期' : new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).format(date)
}

export function safeNext(value: unknown, fallback: string): string {
  return typeof value === 'string' && value.startsWith('/') && !value.startsWith('//') && !/[\\\u0000-\u001f\u007f]/.test(value) && !/^\/(admin\/)?login(?:[/?#]|$)/i.test(value) ? value : fallback
}
