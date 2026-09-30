import type { ArticleInput, Identity } from '../api/types'
import { articleInput } from './article'

export interface EditorDraftOwner { identity: Identity; accountId: number; articleId: number }
export interface EditorDraftRecord { version: 1; revision: string; savedAt: number; baseline: string; value: ArticleInput }
export type EditorDraftReadResult = { status: 'found'; record: EditorDraftRecord } | { status: 'none' | 'unavailable' | 'invalid' }
export type EditorDraftWriteResult = { status: 'saved'; record: EditorDraftRecord } | { status: 'conflict' | 'unavailable' | 'invalid' | 'too-large' }
export type EditorDraftRemoveResult = { status: 'removed' | 'conflict' | 'unavailable' | 'invalid' }

// 上限按整条记录的 UTF-8 字节计算，包含用于判断服务端变化的基线。
export const EDITOR_DRAFT_MAX_BYTES = 2 * 1024 * 1024
const prefix = 'yia-blog:editor-draft:v1:'
const nonnegative = (value: unknown): value is number => typeof value === 'number' && Number.isSafeInteger(value) && value >= 0
const object = (value: unknown): value is Record<string, unknown> => value !== null && typeof value === 'object' && !Array.isArray(value)
const bytes = (value: string) => new TextEncoder().encode(value).length

/** editorDraftOwner 将新文章统一为 0；身份和账号都进入 key，Token 不参与持久化。 */
export function editorDraftOwner(identity: Identity, accountId: number, articleId = 0): EditorDraftOwner | null {
  return (identity === 'user' || identity === 'admin') && nonnegative(accountId) && accountId > 0 && nonnegative(articleId) ? { identity, accountId, articleId } : null
}

export function editorDraftKey(owner: EditorDraftOwner): string {
  return `${prefix}${owner.identity}:${owner.accountId}:${owner.articleId || 'new'}`
}

/** editorDraftFingerprint 保留空白与未完成内容，并去掉作者、统计等不可编辑字段。 */
export function editorDraftFingerprint(value: ArticleInput): string {
  return JSON.stringify(articleInput(value))
}

function safeCover(value: string): boolean {
  if (!value.trim()) return true
  try { return ['http:', 'https:'].includes(new URL(value, 'https://draft.local').protocol) } catch { return false }
}

function parseValue(value: unknown): ArticleInput | null {
  if (!object(value)) return null
  for (const key of ['title', 'cover', 'summary', 'content', 'tag_ids']) if (typeof value[key] !== 'string') return null
  if (!nonnegative(value.category_id) || !nonnegative(value.sort) || ![0, 1, 2].includes(value.status as number) || ![0, 1].includes(value.is_top as number) || ![0, 1].includes(value.comment_status as number)) return null
  if (!safeCover(value.cover as string)) return null
  if (value.publish_at !== null && (typeof value.publish_at !== 'string' || value.publish_at.length > 64 || !/^\d{4}-\d{2}-\d{2}T/.test(value.publish_at) || !Number.isFinite(Date.parse(value.publish_at)))) return null
  return articleInput(value as unknown as ArticleInput)
}

function validBaseline(value: unknown): value is string {
  if (typeof value !== 'string' || bytes(value) > EDITOR_DRAFT_MAX_BYTES) return false
  try {
    const parsed = parseValue(JSON.parse(value))
    return parsed !== null && editorDraftFingerprint(parsed) === value
  } catch { return false }
}

function parseRecord(raw: string): EditorDraftRecord | null {
  if (bytes(raw) > EDITOR_DRAFT_MAX_BYTES) return null
  try {
    const stored: unknown = JSON.parse(raw)
    if (!object(stored) || stored.version !== 1 || typeof stored.revision !== 'string' || !/^[a-zA-Z0-9-]{1,128}$/.test(stored.revision) || !nonnegative(stored.savedAt) || !Number.isFinite(new Date(stored.savedAt).getTime()) || !validBaseline(stored.baseline)) return null
    const value = parseValue(stored.value)
    return value ? { version: 1, revision: stored.revision, savedAt: stored.savedAt, baseline: stored.baseline, value } : null
  } catch { return null }
}

function validOwner(owner: EditorDraftOwner): boolean {
  return object(owner) && editorDraftOwner(owner.identity, owner.accountId, owner.articleId) !== null
}

export function readEditorDraft(owner: EditorDraftOwner): EditorDraftReadResult {
  if (!validOwner(owner)) return { status: 'invalid' }
  try {
    const raw = window.localStorage.getItem(editorDraftKey(owner))
    if (raw === null) return { status: 'none' }
    const record = parseRecord(raw)
    return record ? { status: 'found', record } : { status: 'invalid' }
  } catch { return { status: 'unavailable' } }
}

/**
 * writeEditorDraft 拒绝覆盖已观察到的其他版本；写后再读回，不能确认时不报告成功。
 * localStorage 没有原子 CAS：检查与写入之间仍可能竞争；storage 事件由编辑器暂停自动写入。
 */
export function writeEditorDraft(owner: EditorDraftOwner, input: { value: ArticleInput; baseline: string }, expectedRevision: string | null): EditorDraftWriteResult {
  if (!validOwner(owner)) return { status: 'invalid' }
  const value = parseValue(input.value)
  if (!value || !validBaseline(input.baseline)) return { status: 'invalid' }
  try {
    const storage = window.localStorage
    const key = editorDraftKey(owner)
    const raw = storage.getItem(key)
    const current = raw === null ? null : parseRecord(raw)
    if (raw !== null && !current) return { status: 'invalid' }
    if ((current?.revision ?? null) !== expectedRevision) return { status: 'conflict' }
    const record: EditorDraftRecord = { version: 1, revision: crypto.randomUUID(), savedAt: Date.now(), baseline: input.baseline, value }
    const serialized = JSON.stringify(record)
    if (bytes(serialized) > EDITOR_DRAFT_MAX_BYTES) return { status: 'too-large' }
    if (storage.getItem(key) !== raw) return { status: 'conflict' }
    storage.setItem(key, serialized)
    return storage.getItem(key) === serialized ? { status: 'saved', record } : { status: 'conflict' }
  } catch { return { status: 'unavailable' } }
}

/** removeEditorDraft 仅清理调用方持有的版本，不删除另一个标签页的新暂存。 */
export function removeEditorDraft(owner: EditorDraftOwner, expectedRevision: string | null): EditorDraftRemoveResult {
  if (!validOwner(owner)) return { status: 'invalid' }
  try {
    const storage = window.localStorage
    const key = editorDraftKey(owner)
    const raw = storage.getItem(key)
    const current = raw === null ? null : parseRecord(raw)
    if (raw !== null && !current) return { status: 'invalid' }
    if ((current?.revision ?? null) !== expectedRevision) return { status: 'conflict' }
    if (storage.getItem(key) !== raw) return { status: 'conflict' }
    storage.removeItem(key)
    return storage.getItem(key) === null ? { status: 'removed' } : { status: 'conflict' }
  } catch { return { status: 'unavailable' } }
}

/** discardInvalidEditorDraft 供用户显式清除损坏记录；新出现的有效草稿必须保留。 */
export function discardInvalidEditorDraft(owner: EditorDraftOwner): EditorDraftRemoveResult {
  if (!validOwner(owner)) return { status: 'invalid' }
  try {
    const storage = window.localStorage
    const key = editorDraftKey(owner)
    const raw = storage.getItem(key)
    if (raw === null || parseRecord(raw) !== null || storage.getItem(key) !== raw) return { status: 'conflict' }
    storage.removeItem(key)
    return storage.getItem(key) === null ? { status: 'removed' } : { status: 'conflict' }
  } catch { return { status: 'unavailable' } }
}
