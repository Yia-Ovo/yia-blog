import { useCallback, useEffect, useReducer, useState } from 'react'
import type { ArticleInput } from '../api/types'
import { getLocale, t, useI18n } from '../i18n'
import {
  discardInvalidEditorDraft, editorDraftFingerprint, editorDraftKey, readEditorDraft, removeEditorDraft, writeEditorDraft,
} from '../lib/editorDraft'
import type { EditorDraftOwner, EditorDraftRecord } from '../lib/editorDraft'

export type EditorDraftStatus = 'idle' | 'pending' | 'saved' | 'recovery' | 'conflict' | 'unavailable' | 'invalid' | 'too-large' | 'cleared'
type Phase = 'active' | 'recovery' | 'invalid-record' | 'conflict' | 'cleared'

interface DraftSession {
  owner: EditorDraftOwner
  initialFingerprint: string
  baseline: string
  value: ArticleInput
  restoreValue: (value: ArticleInput) => void
  revision: string | null
  savedFingerprint: string | null
  savedAt: number | null
  recovery: EditorDraftRecord | null
  serverChanged: boolean
  phase: Phase
  status: EditorDraftStatus
  timer?: ReturnType<typeof setTimeout>
  mounted: boolean
  notify: () => void
}

function beginSession(owner: EditorDraftOwner, initial: ArticleInput, value: ArticleInput): DraftSession {
  const existing = readEditorDraft(owner)
  const record = existing.status === 'found' ? existing.record : null
  const baseline = editorDraftFingerprint(initial)
  return {
    owner: { ...owner }, initialFingerprint: baseline, baseline, value, restoreValue: () => {},
    revision: record?.revision ?? null, savedFingerprint: null, savedAt: null, recovery: record,
    serverChanged: Boolean(record && record.baseline !== baseline),
    phase: record ? 'recovery' : existing.status === 'invalid' ? 'invalid-record' : 'active',
    status: record ? 'recovery' : existing.status === 'invalid' ? 'invalid' : existing.status === 'unavailable' ? 'unavailable' : 'idle',
    mounted: false, notify: () => {},
  }
}

function notify(session: DraftSession) {
  if (session.mounted) session.notify()
}

function cancelTimer(session: DraftSession) {
  clearTimeout(session.timer)
  session.timer = undefined
}

function fail(session: DraftSession, kind: 'conflict' | 'unavailable' | 'invalid' | 'too-large') {
  session.status = kind
  if (kind === 'conflict') session.phase = 'conflict'
  notify(session)
}

/** 同步写入让 pagehide 和卸载也能保留最后一次已输入的内容。 */
function flushSession(session: DraftSession) {
  cancelTimer(session)
  if (session.phase !== 'active') return
  const fingerprint = editorDraftFingerprint(session.value)
  if (fingerprint === session.initialFingerprint) {
    if (session.status === 'unavailable' && !session.revision) return
    if (session.revision) {
      const result = removeEditorDraft(session.owner, session.revision)
      if (result.status !== 'removed') { fail(session, result.status); return }
      session.revision = null
    }
    session.savedFingerprint = null
    session.savedAt = null
    session.status = 'idle'
    notify(session)
    return
  }
  if (fingerprint === session.savedFingerprint) return
  const result = writeEditorDraft(session.owner, { value: session.value, baseline: session.baseline }, session.revision)
  if (result.status !== 'saved') { fail(session, result.status); return }
  session.revision = result.record.revision
  session.savedFingerprint = fingerprint
  session.savedAt = result.record.savedAt
  session.status = 'saved'
  notify(session)
}

function schedule(session: DraftSession) {
  cancelTimer(session)
  if (session.phase !== 'active') return
  const fingerprint = editorDraftFingerprint(session.value)
  if (fingerprint === session.initialFingerprint) {
    flushSession(session)
  } else if (fingerprint !== session.savedFingerprint) {
    session.status = 'pending'
    notify(session)
    session.timer = setTimeout(() => flushSession(session), 600)
  } else if (session.status !== 'saved') {
    session.status = 'saved'
    notify(session)
  }
}

function timeLabel(timestamp: number) {
  return new Intl.DateTimeFormat(getLocale(), { hour: '2-digit', minute: '2-digit', hour12: false }).format(timestamp)
}

function describe(session: DraftSession): string {
  switch (session.status) {
    case 'idle': return t('修改会自动暂存到这台设备，保存到博客仍需点击保存。')
    case 'pending': return t('正在暂存到本机…')
    case 'saved': return t('已暂存到本机 {time}，尚未保存到服务器。', { time: timeLabel(session.savedAt!) })
    case 'recovery': return t('发现上次未保存的内容（{time}）。恢复后仍需保存到博客。', { time: timeLabel(session.recovery!.savedAt) })
    case 'conflict': return t('另一个标签页修改了本机暂存，已暂停自动暂存。请先复制当前正文，再刷新核对。')
    case 'unavailable': return t('本机暂存不可用，当前修改尚未暂存。请复制正文或保存到博客。')
    case 'invalid': return session.phase === 'invalid-record'
      ? t('上次的本机暂存无法读取，已保留原记录。可放弃该暂存后继续。')
      : t('部分字段暂时无法暂存，请检查排序等设置；当前修改尚未暂存。')
    case 'too-large': return t('内容超过本机暂存容量，当前修改尚未暂存。请复制正文或保存到博客。')
    case 'cleared': return t('已保存到博客。')
  }
}

interface Options {
  /** 身份和文章在编辑器实例中固定；切换账号或文章时由父组件重新挂载。 */
  owner: EditorDraftOwner
  initial: ArticleInput
  value: ArticleInput
  onRestore: (value: ArticleInput) => void
}

export function useEditorDraft({ owner, initial, value, onRestore }: Options) {
  // 语言只影响提示展示，不参与草稿会话与自动暂存的生命周期。
  useI18n()
  const [, redraw] = useReducer((number: number) => number + 1, 0)
  const [session] = useState(() => beginSession(owner, initial, value))
  session.value = value
  session.restoreValue = onRestore
  session.notify = redraw

  const flush = useCallback(() => flushSession(session), [session])
  const restore = useCallback(() => {
    const record = session.recovery
    if (session.phase !== 'recovery' || !record) return
    if (editorDraftFingerprint(session.value) !== session.initialFingerprint &&
      !window.confirm(t('恢复暂存会替换当前编辑器中的内容。请先复制需要保留的内容，确认恢复？'))) return
    // 选择之前再次核对，避免覆盖另一个标签页刚刚写入的版本。
    const current = readEditorDraft(session.owner)
    if (current.status !== 'found' || current.record.revision !== record.revision) {
      fail(session, current.status === 'unavailable' ? 'unavailable' : 'conflict')
      return
    }
    session.phase = 'active'
    session.baseline = record.baseline
    session.recovery = null
    session.value = { ...record.value }
    session.savedFingerprint = editorDraftFingerprint(record.value)
    session.savedAt = record.savedAt
    session.status = 'saved'
    session.restoreValue({ ...record.value })
    notify(session)
  }, [session])

  const discard = useCallback(() => {
    if (session.phase !== 'recovery' && session.phase !== 'invalid-record') return
    const result = session.phase === 'invalid-record'
      ? discardInvalidEditorDraft(session.owner)
      : removeEditorDraft(session.owner, session.revision)
    if (result.status !== 'removed') { fail(session, result.status); return }
    session.phase = 'active'
    session.recovery = null
    session.revision = null
    session.savedFingerprint = null
    session.savedAt = null
    session.serverChanged = false
    session.status = 'idle'
    schedule(session)
    notify(session)
  }, [session])

  const clearAfterSave = useCallback(() => {
    cancelTimer(session)
    const owned = session.phase === 'active' || session.phase === 'conflict'
    // 成功保存后先停止生命周期写入，卸载时不会把刚清除的暂存重新创建。
    session.phase = 'cleared'
    session.status = 'cleared'
    if (owned) {
      const result = removeEditorDraft(session.owner, session.revision)
      if (result.status !== 'removed') session.status = result.status
      else { session.revision = null; session.savedAt = null; session.savedFingerprint = null }
    }
    notify(session)
    return owned && session.status === 'cleared'
  }, [session])

  useEffect(() => {
    session.mounted = true
    const onHidden = () => { if (document.visibilityState === 'hidden') flushSession(session) }
    const onStorage = (event: StorageEvent) => {
      if (event.key !== null && event.key !== editorDraftKey(session.owner)) return
      if (session.phase === 'cleared' || session.phase === 'conflict') return
      const current = readEditorDraft(session.owner)
      if (current.status === 'unavailable') { fail(session, 'unavailable'); return }
      if ((current.status === 'none' && session.revision === null && session.phase !== 'invalid-record') ||
        (current.status === 'found' && current.record.revision === session.revision) ||
        (current.status === 'invalid' && session.phase === 'invalid-record')) return
      cancelTimer(session)
      fail(session, 'conflict')
    }
    window.addEventListener('beforeunload', flush)
    window.addEventListener('pagehide', flush)
    document.addEventListener('visibilitychange', onHidden)
    window.addEventListener('storage', onStorage)
    return () => {
      session.mounted = false
      flushSession(session)
      window.removeEventListener('beforeunload', flush)
      window.removeEventListener('pagehide', flush)
      document.removeEventListener('visibilitychange', onHidden)
      window.removeEventListener('storage', onStorage)
    }
  }, [session, flush])

  useEffect(() => {
    schedule(session)
    return () => cancelTimer(session)
  }, [session, value])

  return {
    status: session.status, message: describe(session), savedAt: session.savedAt,
    recovery: session.recovery, serverChanged: session.serverChanged,
    canDiscard: session.phase === 'recovery' || session.phase === 'invalid-record',
    restore, discard, clearAfterSave, flush,
  }
}

export type EditorDraftController = ReturnType<typeof useEditorDraft>
