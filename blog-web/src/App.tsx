import { Suspense, useCallback, useEffect, useRef, useSyncExternalStore } from 'react'
import { Link, NavLink, Outlet, ScrollRestoration, useLocation, useMatches } from 'react-router-dom'
import { authApi } from './api'
import { ApiError, errorMessage } from './api/http'
import type { Identity } from './api/types'
import { useMutation } from './hooks/useMutation'
import { useSessions } from './hooks/useSessions'
import { clearSession } from './stores/session'
import { dismissNotice, getNotice, notify, subscribeNotice } from './stores/notice'
import { LeaveGuardContext } from './components/LeaveGuardContext'
import StatePanel from './components/StatePanel'
import UiIcon from './components/UiIcon'

export default function App() {
  const location = useLocation()
  const matches = useMatches()
  const sessions = useSessions()
  const notice = useSyncExternalStore(subscribeNotice, getNotice)
  const { busy, run } = useMutation()
  const leaveGuard = useRef<(() => boolean) | null>(null)
  const registerLeaveGuard = useCallback((guard: () => boolean) => {
    leaveGuard.current = guard
    return () => { if (leaveGuard.current === guard) leaveGuard.current = null }
  }, [])
  const title = (matches.at(-1)?.handle as { title?: string } | undefined)?.title || '博客'
  useEffect(() => { document.title = `${title} · Yia` }, [title])

  function logout(identity: Identity) {
    if (busy || (leaveGuard.current && !leaveGuard.current())) return
    const token = sessions[identity]?.token
    void run(() => authApi.logout(identity), () => {
      clearSession(identity, token)
      notify('已退出登录。')
    }, (cause) => {
      if (!(cause instanceof ApiError && cause.authExpired)) notify(errorMessage(cause), 'error')
    })
  }

  const adminArea = location.pathname.startsWith('/admin')
  return <LeaveGuardContext.Provider value={registerLeaveGuard}>
    <a className="skip-link" href="#main">跳到正文</a>
    <header className="site-header">
      <div className="container header-inner">
        <Link className="brand" to="/" aria-label="Yia 博客首页"><span className="brand-mark">y.</span><span>Yia<span className="brand-dot">.</span></span></Link>
        <nav className="main-nav" aria-label="主导航">
          <NavLink to="/" end>文章</NavLink>
          <NavLink to="/me/articles" className={location.pathname.startsWith('/write') || location.pathname.startsWith('/me') ? 'active' : undefined}>我的创作</NavLink>
          <NavLink to="/admin/articles" className={adminArea ? 'active' : undefined}>管理</NavLink>
        </nav>
        <div className="header-actions">
          {adminArea && sessions.admin ? <>
            <span className="account-name">{sessions.admin.profile.admin_name}</span>
            <button className="icon-button" disabled={busy} aria-label="退出管理员登录" onClick={() => logout('admin')}><UiIcon name="logout" /></button>
          </> : sessions.user ? <>
            <Link className="account-name" to="/me/articles">{sessions.user.profile.nick_name}</Link>
            <button className="icon-button" disabled={busy} aria-label="退出用户登录" onClick={() => logout('user')}><UiIcon name="logout" /></button>
          </> : <Link className="login-link" to={adminArea ? '/admin/login' : '/login'}>登录</Link>}
          <Link className="button primary small write-link" to={adminArea && sessions.admin ? '/admin/articles/new' : '/write/new'}><UiIcon name="edit" size={16} />写文章</Link>
        </div>
      </div>
    </header>
    <main id="main" tabIndex={-1}>
      <Suspense fallback={<StatePanel kind="loading" />}><Outlet key={location.pathname} /></Suspense>
    </main>
    <footer className="site-footer container"><Link className="footer-brand" to="/">Yia.</Link><span>保持好奇，持续记录。</span><span className="footer-end">用文字留下思考 · {new Date().getFullYear()}</span></footer>
    {notice && <div className={`toast ${notice.kind}`} role={notice.kind === 'error' ? 'alert' : 'status'}>
      <UiIcon name={notice.kind === 'success' ? 'check' : 'comment'} /><span>{notice.text}</span>
      <button className="icon-button" aria-label="关闭提示" onClick={dismissNotice}><UiIcon name="close" size={16} /></button>
    </div>}
    <ScrollRestoration />
  </LeaveGuardContext.Provider>
}
