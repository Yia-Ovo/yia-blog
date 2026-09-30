import { request } from './http'

export interface DailyAppearance {
  date: string
  image_url: string
  server_time: string
  next_change_at: string
}

export const appearanceApi = {
  today: (signal?: AbortSignal) => request<DailyAppearance>('/appearance/today', { signal }),
}

// 图片跟随 API 地址解析，兼容同源代理与独立 API 域名。
export function artworkURL(path: string): string {
  const base = new URL(import.meta.env.VITE_API_BASE_URL || '/api/blog/v1', window.location.href)
  const url = new URL(path, base)
  if (!['http:', 'https:'].includes(url.protocol) || url.origin !== base.origin || url.username || url.password) {
    throw new Error('Invalid artwork URL')
  }
  return url.href
}
