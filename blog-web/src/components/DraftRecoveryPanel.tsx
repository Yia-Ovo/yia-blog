import type { EditorDraftController } from '../hooks/useEditorDraft'
import { useI18n } from '../i18n'

export default function DraftRecoveryPanel({ draft }: { draft: EditorDraftController }) {
  const { t } = useI18n()
  const warning = ['conflict', 'unavailable', 'invalid', 'too-large'].includes(draft.status)
  return <section className={`draft-recovery-panel${warning ? ' draft-recovery-warning' : ''}`} aria-label={t('本机暂存')} data-status={draft.status}>
    <p className="draft-recovery-message" role={warning ? 'alert' : 'status'} aria-live="polite">{draft.message}</p>
    {draft.serverChanged && <p className="draft-recovery-meta">{t('服务器上的文章与暂存时不同，请核对内容后再保存。')}</p>}
    {draft.canDiscard && <div className="draft-recovery-actions">
      {draft.recovery && <button className="button primary" type="button" onClick={draft.restore}>{t('恢复暂存')}</button>}
      <button className="button" type="button" onClick={draft.discard}>{t('放弃暂存')}</button>
    </div>}
  </section>
}
