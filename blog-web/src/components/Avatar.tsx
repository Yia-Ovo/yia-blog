import { useState } from 'react'
import { safeImageURL } from '../lib/markdown'
import { artworkURL } from '../api/appearance'

function avatarURL(value?: string): string {
  if (value?.startsWith('/api/blog/v1/avatar/images/')) {
    try { return artworkURL(value) } catch { return '' }
  }
  return safeImageURL(value)
}

export default function Avatar({ url, name, className = '' }: { url?: string; name?: string; className?: string }) {
  const source = avatarURL(url)
  const [failed, setFailed] = useState('')
  return <span className={`avatar ${className}`} aria-hidden="true">
    {source && source !== failed
      ? <img key={source} src={source} alt="" referrerPolicy="no-referrer" onError={() => setFailed(source)} />
      : <span>{Array.from(name?.trim() || 'Y')[0]}</span>}
  </span>
}
