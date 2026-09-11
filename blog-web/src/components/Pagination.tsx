interface Props { page: number; pages: number; busy?: boolean; onChange: (page: number) => void }

export default function Pagination({ page, pages, busy, onChange }: Props) {
  if (pages <= 1) return null
  return <nav className="pagination" aria-label="分页">
    <button type="button" className="button secondary small" disabled={page <= 1 || busy} onClick={() => onChange(page - 1)}>上一页</button>
    <span aria-live="polite">{page} <span className="muted">/ {pages}</span></span>
    <button type="button" className="button secondary small" disabled={page >= pages || busy} onClick={() => onChange(page + 1)}>下一页</button>
  </nav>
}
