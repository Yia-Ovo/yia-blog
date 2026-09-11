import type { Identity, LoginResult, Profile } from '../api/types'

export interface Session { token: string; expire: string; profile: Profile }
const storageKey = (identity: Identity) => `yia.session.${identity}`

function read(identity: Identity): Session | null {
  try {
    const raw: unknown = JSON.parse(sessionStorage.getItem(storageKey(identity)) || 'null')
    if (!raw || typeof raw !== 'object') return null
    const value = raw as Session
    if (typeof value.token !== 'string' || !value.token || !value.profile || !Number.isSafeInteger(value.profile.id) || value.profile.id <= 0) return null
    return value
  } catch { return null }
}

// 两套令牌只保留在当前标签页；本地资料只用于展示，权限仍由 API 判断。
export let sessions: Readonly<Record<Identity, Session | null>> = { user: read('user'), admin: read('admin') }
const listeners = new Set<() => void>()

export const getSessions = () => sessions
export function subscribeSessions(listener: () => void) {
  listeners.add(listener)
  return () => { listeners.delete(listener) }
}

function persist(identity: Identity) {
  try {
    const value = sessions[identity]
    if (value) sessionStorage.setItem(storageKey(identity), JSON.stringify(value))
    else sessionStorage.removeItem(storageKey(identity))
  } catch { /* 浏览器禁用存储时，当前页面仍可使用内存登录态。 */ }
}

export function saveSession(identity: Identity, result: LoginResult) {
  update(identity, { token: result.token, expire: result.expire, profile: result.data })
}

export function refreshProfile(identity: Identity, profile: Profile, token: string) {
  if (sessions[identity]?.token !== token) return
  update(identity, { ...sessions[identity]!, profile })
}

export function clearSession(identity: Identity, expectedToken?: string) {
  // 旧请求的失败响应不能清掉后来重新登录得到的新令牌。
  if (expectedToken !== undefined && sessions[identity]?.token !== expectedToken) return
  if (!sessions[identity]) return
  update(identity, null)
}

function update(identity: Identity, value: Session | null) {
  // 每次变更产生新快照，React 的外部状态订阅才能可靠识别变化。
  sessions = { ...sessions, [identity]: value }
  persist(identity)
  listeners.forEach((listener) => listener())
}
