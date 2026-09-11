import { describe, expect, it } from 'vitest'
import { renderMarkdown, safeImageURL } from '../src/lib/markdown'

describe('untrusted Markdown', () => {
  it('removes executable HTML, event handlers and page-overlay styles', () => {
    const html = renderMarkdown('<script>alert(1)</script><img src="https://example.com/a.png" onerror="alert(1)"><p style="position:fixed;inset:0">hello</p><iframe src="https://example.com"></iframe>')
    expect(html).toContain('hello')
    expect(html).not.toMatch(/<script|onerror|<iframe|position:fixed/i)
  })

  it('blocks script links and protects external navigation', () => {
    const html = renderMarkdown('[bad](javascript:alert%281%29)\n\n[good](https://example.com)')
    const node = document.createElement('div'); node.innerHTML = html
    expect(node.querySelector('a[href^="javascript:"]')).toBeNull()
    expect(node.querySelector('a[href="https://example.com"]')?.getAttribute('rel')).toBe('noopener noreferrer')
  })

  it('preserves code blocks but refuses non-web image protocols', () => {
    expect(renderMarkdown('```go\nfmt.Println("hello")\n```')).toContain('language-go')
    expect(safeImageURL('javascript:alert(1)')).toBe('')
    expect(safeImageURL('data:image/svg+xml,<svg onload="alert(1)"/>')).toBe('')
    expect(renderMarkdown('![bad](data:image/svg+xml,x)')).not.toContain('<img')
  })
})
