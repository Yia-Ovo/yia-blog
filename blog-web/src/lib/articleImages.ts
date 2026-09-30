import type { ArticleImage } from '../api/types'

export function validArticleImage(value: ArticleImage): boolean {
  return !!value && typeof value.url === 'string' &&
    /^\/api\/blog\/v1\/article\/images\/[a-f0-9]{64}\.(?:png|jpg)$/.test(value.url) &&
    Number.isSafeInteger(value.width) && value.width > 0 &&
    Number.isSafeInteger(value.height) && value.height > 0
}

export interface ImageInsertion { content: string; position: number }

/** 上传期间继续写作时追加到末尾，避免旧光标位置覆盖新输入。 */
export function insertArticleImage(current: string, image: ArticleImage, name: string, selection: ImageInsertion): string {
  const alt = name.replace(/\.[^.]+$/, '').replace(/[\\\[\]\r\n]/g, '').trim().slice(0, 100) || '图片'
  const position = current === selection.content ? Math.min(Math.max(0, selection.position), current.length) : current.length
  return current.slice(0, position) + '\n\n![' + alt + '](' + image.url + ')\n\n' + current.slice(position)
}
