import DOMPurify from 'dompurify'
import { marked } from 'marked'

export function safeImageURL(value?: string | null): string {
  if (!value?.trim()) return ''
  try {
    const url = new URL(value, window.location.origin)
    return ['http:', 'https:'].includes(url.protocol) ? url.href : ''
  } catch { return '' }
}

export function renderMarkdown(source: string): string {
  const html = marked.parse(source, { async: false, gfm: true })
  const clean = DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
    FORBID_TAGS: ['style', 'form', 'input', 'textarea', 'button', 'iframe', 'object', 'embed'],
    FORBID_ATTR: ['style', 'srcset'],
  })
  const template = document.createElement('template')
  template.innerHTML = clean
  template.content.querySelectorAll('a[href]').forEach((link) => {
    try {
      const url = new URL(link.getAttribute('href') || '', window.location.origin)
      if (!['http:', 'https:', 'mailto:'].includes(url.protocol)) { link.removeAttribute('href'); return }
      if (url.origin !== window.location.origin) {
        link.setAttribute('target', '_blank')
        link.setAttribute('rel', 'noopener noreferrer')
      }
    } catch { link.removeAttribute('href') }
  })
  template.content.querySelectorAll('img').forEach((image) => {
    const url = safeImageURL(image.getAttribute('src'))
    if (!url) { image.remove(); return }
    image.setAttribute('src', url)
    image.setAttribute('loading', 'lazy')
    image.setAttribute('referrerpolicy', 'no-referrer')
  })
  return template.innerHTML
}
