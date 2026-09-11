import { useMemo } from 'react'
import { renderMarkdown } from '../lib/markdown'

export default function MarkdownBody({ source }: { source: string }) {
  const html = useMemo(() => renderMarkdown(source), [source])
  // 唯一的 HTML 注入点：先经过 DOMPurify 和 URL 协议过滤。
  return <div className="prose" dangerouslySetInnerHTML={{ __html: html }} />
}
