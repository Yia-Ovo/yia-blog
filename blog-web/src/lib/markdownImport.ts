export const markdownImportLimit = 1024 * 1024

export interface ImportedMarkdown {
  filename: string
  title: string
  content: string
}

export function parseMarkdownFile(filename: string, bytes: ArrayBuffer): ImportedMarkdown {
  if (!/\.(md|markdown)$/i.test(filename)) throw new Error('请选择 .md 或 .markdown 文件。')
  if (!bytes.byteLength || bytes.byteLength > markdownImportLimit) throw new Error('Markdown 文件不能为空，大小不能超过 1 MB。')
  let text: string
  try { text = new TextDecoder('utf-8', { fatal: true }).decode(bytes) }
  catch { throw new Error('文件不是有效的 UTF-8 文本，请转换编码后重试。') }
  text = text.replace(/^\uFEFF/, '').replace(/\r\n?/g, '\n')
  if (/[\u0000-\u0008\u000b\u000c\u000e-\u001f]/.test(text)) throw new Error('文件包含非文本内容，请选择 Markdown 文本。')
  const lines = text.split('\n')
  const first = lines.findIndex((line) => line.trim())
  const heading = first >= 0 ? /^ {0,3}#[ \t]+(.+)$/.exec(lines[first]!) : null
  const title = (heading ? heading[1]!.replace(/[ \t]+#+[ \t]*$/, '') : filename.replace(/\.(md|markdown)$/i, '')).trim()
  // 仅提取文件开头的一级标题；正文中的标题、代码缩进和元数据都保留为文本。
  const content = heading ? lines.slice(first + 1).join('\n').replace(/^\n/, '') : text
  if (!title) throw new Error('请先给文章起个标题。')
  if (new TextEncoder().encode(title).length > 128) throw new Error('标题太长，请适当缩短（中文约 42 个字）。')
  if (!content.trim()) throw new Error('请填写文章正文，草稿也需要有内容。')
  return { filename, title, content }
}

export async function readMarkdownFile(file: File): Promise<ImportedMarkdown> {
  if (!/\.(md|markdown)$/i.test(file.name)) throw new Error('请选择 .md 或 .markdown 文件。')
  if (!file.size || file.size > markdownImportLimit) throw new Error('Markdown 文件不能为空，大小不能超过 1 MB。')
  const bytes = await new Promise<ArrayBuffer>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => reader.result instanceof ArrayBuffer ? resolve(reader.result) : reject(new Error('无法读取文件，请重新选择。'))
    reader.onerror = () => reject(new Error('无法读取文件，请重新选择。'))
    reader.onabort = () => reject(new Error('无法读取文件，请重新选择。'))
    reader.readAsArrayBuffer(file)
  })
  return parseMarkdownFile(file.name, bytes)
}
