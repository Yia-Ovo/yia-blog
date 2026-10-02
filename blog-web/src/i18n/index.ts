import { useSyncExternalStore } from 'react'
import { commonMessages } from './common'
import { publicMessages } from './public'
import { accountMessages } from './account'
import { editorMessages } from './editor'

export const locales = ['zh-CN', 'en', 'ja', 'vi'] as const
export type Locale = typeof locales[number]
export const localeNames: Record<Locale, string> = { 'zh-CN': '中文', en: 'English', ja: '日本語', vi: 'Tiếng Việt' }
export const localeStorageKey = 'yia.locale'
export const messages: Record<string, { en: string; ja: string; vi: string }> = {
  ...publicMessages, ...accountMessages, ...editorMessages, ...commonMessages,
}
export function isLocale(value: unknown): value is Locale {
  return typeof value === 'string' && (locales as readonly string[]).includes(value)
}
export function readSavedLocale(): Locale {
  try {
    const saved = localStorage.getItem(localeStorageKey)
    return isLocale(saved) ? saved : 'zh-CN'
  } catch { return 'zh-CN' }
}
let locale = readSavedLocale()
const listeners = new Set<() => void>()
export const getLocale = () => locale

function applyLocale(next: Locale) {
  const changed = next !== locale
  locale = next
  if (typeof document !== 'undefined') document.documentElement.lang = locale
  if (changed) listeners.forEach((listener) => listener())
}
applyLocale(locale)

export function setLocale(next: Locale) {
  if (!isLocale(next)) return
  applyLocale(next)
  try { localStorage.setItem(localeStorageKey, next) } catch { /* 禁用存储时仍可在当前页面切换。 */ }
}
function onStorage(event: StorageEvent) {
  if (event.key !== localeStorageKey && event.key !== null) return
  try { if (event.storageArea && event.storageArea !== localStorage) return } catch { return }
  const value = event.key === null ? null : event.newValue
  applyLocale(isLocale(value) ? value : 'zh-CN')
}
export function subscribeLocale(listener: () => void) {
  if (listeners.size === 0 && typeof window !== 'undefined') window.addEventListener('storage', onStorage)
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
    if (listeners.size === 0 && typeof window !== 'undefined') window.removeEventListener('storage', onStorage)
  }
}
/** 中文原文作为词条键；只用于界面文案，用户正文等内容不传入翻译。 */
export function t(source: string, params: Record<string, string | number> = {}): string {
  const translation = Object.hasOwn(messages, source) ? messages[source] : undefined
  const text = locale === 'zh-CN' ? source : translation?.[locale] || source
  return text.replace(/\{(\w+)\}/g, (placeholder, key: string) => Object.hasOwn(params, key) ? String(params[key]) : placeholder)
}
export function useI18n() {
  const current = useSyncExternalStore(subscribeLocale, getLocale, () => 'zh-CN' as Locale)
  return { locale: current, setLocale, t }
}
