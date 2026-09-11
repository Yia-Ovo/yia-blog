import { useCallback, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { Link, Navigate, useLocation } from 'react-router-dom'
import { authApi } from '../api'
import type { Identity } from '../api/types'
import { useResource } from '../hooks/useResource'
import { useSessions } from '../hooks/useSessions'
import { refreshProfile } from '../stores/session'
import StatePanel from './StatePanel'

export default function RequireSession({ identity, superOnly = false, preserveEditor = false, children }: { identity: Identity; superOnly?: boolean; preserveEditor?: boolean; children: ReactNode }) {
  const sessions = useSessions()
  const token = sessions[identity]?.token
  const location = useLocation()
  const [openedIdentity, setOpenedIdentity] = useState<Identity | null>(null)
  const profile = useResource(useCallback((signal) => token ? authApi.profile(identity, signal) : Promise.resolve(null), [identity, token]))

  useEffect(() => {
    if (profile.data && token) {
      refreshProfile(identity, profile.data, token)
      setOpenedIdentity(identity)
    }
  }, [identity, token, profile.data])

  const next = encodeURIComponent(location.pathname + location.search + location.hash)
  const login = `${identity === 'admin' ? '/admin/login' : '/login'}?next=${next}`
  const expiredEditor = !token && preserveEditor && openedIdentity === identity
  if (!token && !expiredEditor) return <Navigate replace to={login} />
  if (!expiredEditor && profile.pending) return <StatePanel kind="loading" title="正在打开工作区…" />
  // 网络故障保留登录态，并允许重试；本地资料不作为管理权限依据。
  if (!expiredEditor && profile.error) return <StatePanel kind="error" message={profile.error} onRetry={profile.reload} />
  if (!expiredEditor && superOnly && profile.data?.role !== 0) return <StatePanel kind="empty" title="分类维护仅向超级管理员开放" message="当前账号可以继续管理文章和审核评论。"><Link className="button secondary small" to="/admin/articles">返回文章管理</Link></StatePanel>
  // 已打开的编辑器保持同一组件实例，401 后保留用户输入；新的私有页面仍必须登录。
  return <>
    {expiredEditor && <p className="container inline-note" role="alert">登录已失效，当前编辑内容仍保留在页面中。请先复制需要保留的内容，再<Link className="text-link" to={login}>重新登录</Link>。</p>}
    {children}
  </>
}
