type Notice = { text: string; kind: 'success' | 'error' } | null
let notice: Notice = null
const listeners = new Set<() => void>()
let timer: ReturnType<typeof setTimeout> | undefined

export const getNotice = () => notice
export function subscribeNotice(listener: () => void) {
  listeners.add(listener)
  return () => { listeners.delete(listener) }
}

export function dismissNotice() {
  clearTimeout(timer)
  notice = null
  listeners.forEach((listener) => listener())
}

export function notify(text: string, kind: 'success' | 'error' = 'success') {
  clearTimeout(timer)
  notice = { text, kind }
  listeners.forEach((listener) => listener())
  timer = setTimeout(dismissNotice, 6500)
}
