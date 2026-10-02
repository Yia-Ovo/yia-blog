import { clearSession, sessions } from '../stores/session'
import type { Identity, Query } from './types'

const base = (import.meta.env.VITE_API_BASE_URL || '/api/blog/v1').replace(/\/$/, '')
const expiredCodes = new Set([401, 11007, 11008, 11009, 11019, 16006])
const messages: Record<number, string> = {
  400: '请求参数不正确，请检查后重试。', 401: '登录已失效，请重新登录。', 403: '当前账号没有操作权限。',
  429: '请求过于频繁，请稍后重试。',
  500: '服务暂时不可用，请稍后重试。', 10000: '数据服务暂时不可用，请稍后重试。',
  11002: '昵称已被使用，请换一个。', 11004: '账号或密码不正确。', 11005: '登录过于频繁，请稍后重试。',
  11006: '账号已锁定，请稍后再试。', 11007: '登录已过期，请重新登录。', 11008: '登录已失效，请重新登录。',
  11009: '请先登录。', 11019: '此账号已被停用。', 11025: '昵称已被使用，请换一个。',
  12001: '文章不存在，或暂未公开。', 12002: '请填写文章标题。', 12003: '文章标题太长，请适当缩短。',
  12004: '排序值需要是非负整数。',
  12005: '请填写文章正文。', 13001: '分类不存在，请刷新后重试。', 13002: '分类名称已存在。',
  14003: '请填写评论内容。', 14004: '你已经赞过这条评论。', 14005: '你还没有赞过这条评论。',
  14001: '评论不存在，或暂未公开。', 15001: '当前账号没有操作权限。', 16006: '此管理员账号已被停用。',
}

export class ApiError extends Error {
  constructor(public readonly code: number, message: string) { super(message); this.name = 'ApiError' }
  get authExpired() { return expiredCodes.has(this.code) }
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '操作未完成，请稍后重试。'
}

export interface RequestOptions {
  method?: 'GET' | 'POST'
  body?: unknown
  query?: Query
  identity?: Identity
  signal?: AbortSignal
  preferServerMessage?: boolean
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(options.query || {})) {
    if (value !== undefined && value !== null) params.set(key, String(value))
  }
  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = options.identity ? sessions[options.identity]?.token : undefined
  if (options.identity && !token) throw new ApiError(11009, messages[11009]!)
  if (options.identity && token) headers[options.identity === 'user' ? 'Authorization' : 'Admin-Authorization'] = `Bearer ${token}`
  const multipart = options.body instanceof FormData
  // FormData 的 boundary 必须由浏览器随请求体生成。
  if (options.body !== undefined && !multipart) headers['Content-Type'] = 'application/json'
  const controller = new AbortController()
  const abort = () => controller.abort()
  options.signal?.addEventListener('abort', abort, { once: true })
  if (options.signal?.aborted) controller.abort()
  let timedOut = false
  const timer = window.setTimeout(() => { timedOut = true; controller.abort() }, 12000)
  const invalidate = (code: number) => {
    if (options.identity && expiredCodes.has(code)) clearSession(options.identity, token)
  }
  try {
    const query = params.toString()
    const response = await fetch(`${base}${path}${query ? `?${query}` : ''}`, {
      method: options.method || 'GET', headers, credentials: 'omit', signal: controller.signal,
      body: multipart ? options.body as FormData : options.body === undefined ? undefined : JSON.stringify(options.body),
    })
    if (!response.ok) {
      invalidate(response.status)
      throw new ApiError(response.status, messages[response.status] || '服务暂时无法完成请求，请稍后重试。')
    }
    let envelope: unknown
    try { envelope = await response.json() } catch { throw new ApiError(0, '服务响应格式不正确，请稍后重试。') }
    if (!envelope || typeof envelope !== 'object' || !('code' in envelope) || typeof envelope.code !== 'number' || !('data' in envelope)) {
      throw new ApiError(0, '服务响应格式不正确，请稍后重试。')
    }
    if (envelope.code !== 200) {
      invalidate(envelope.code)
      const message = 'msg' in envelope && typeof envelope.msg === 'string' ? envelope.msg : ''
      const publicMessage = options.preferServerMessage && envelope.code === 400 ? message : ''
      throw new ApiError(envelope.code, publicMessage || messages[envelope.code] || message || '操作未完成，请稍后重试。')
    }
    return envelope.data as T
  } catch (error) {
    if (timedOut) throw new ApiError(0, '请求超时，请检查服务状态后重试。')
    if (options.signal?.aborted) throw new DOMException('Request cancelled', 'AbortError')
    if (error instanceof ApiError) throw error
    throw new ApiError(0, '暂时连接不上服务，请确认后端已启动后重试。')
  } finally {
    window.clearTimeout(timer)
    options.signal?.removeEventListener('abort', abort)
  }
}
