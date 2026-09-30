import type { EditorDraftController } from '../hooks/useEditorDraft'

export default function DraftRecoveryPanel({ draft }: { draft: EditorDraftController }) {
  const warning = ['conflict', 'unavailable', 'invalid', 'too-large'].includes(draft.status)
  return <section className={`draft-recovery-panel${warning ? ' draft-recovery-warning' : ''}`} aria-label="本机暂存" data-status={draft.status}>
    <p className="draft-recovery-message" role={warning ? 'alert' : 'status'} aria-live="polite">{draft.message}</p>
    {draft.serverChanged && <p className="draft-recovery-meta">服务器上的文章与暂存时不同，请核对内容后再保存。</p>}
    {draft.canDiscard && <div className="draft-recovery-actions">
      {draft.recovery && <button className="button primary" type="button" onClick={draft.restore}>恢复暂存</button>}
      <button className="button" type="button" onClick={draft.discard}>放弃暂存</button>
    </div>}
  </section>
}
