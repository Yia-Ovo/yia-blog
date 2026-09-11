import type { ReactNode } from 'react'
import UiIcon from './UiIcon'

interface Props {
  kind: 'loading' | 'error' | 'empty'
  title?: string
  message?: string
  onRetry?: () => void
  children?: ReactNode
}

export default function StatePanel({ kind, title, message, onRetry, children }: Props) {
  return <div className="state-panel" role={kind === 'error' ? 'alert' : 'status'} aria-busy={kind === 'loading'}>
    <span className={`state-icon${kind === 'loading' ? ' pulse' : ''}`}><UiIcon name={kind === 'loading' ? 'clock' : kind === 'error' ? 'code' : 'book'} size={28} /></span>
    <h3>{title || (kind === 'loading' ? '正在加载…' : kind === 'error' ? '这次没能加载出来' : '这里还没有内容')}</h3>
    {message && <p>{message}</p>}
    {kind === 'error' && onRetry && <button className="button secondary small" type="button" onClick={onRetry}>重新加载</button>}
    {children}
  </div>
}
