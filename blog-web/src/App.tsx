import { Suspense, useCallback, useEffect, useLayoutEffect, useRef, useState, useSyncExternalStore } from 'react'
import { Link, NavLink, Outlet, ScrollRestoration, useLocation, useMatches } from 'react-router-dom'
import { authApi } from './api'
import { ApiError, errorMessage } from './api/http'
import type { Identity } from './api/types'
import { useMutation } from './hooks/useMutation'
import { useSessions } from './hooks/useSessions'
import { useDailyAppearance } from './hooks/useDailyAppearance'
import { clearSession } from './stores/session'
import { dismissNotice, getNotice, notify, subscribeNotice } from './stores/notice'
import { LeaveGuardContext } from './components/LeaveGuardContext'
import StatePanel from './components/StatePanel'
import UiIcon from './components/UiIcon'
import Avatar from './components/Avatar'

export default function App() {
  const location = useLocation()
  const matches = useMatches()
  const sessions = useSessions()
  const notice = useSyncExternalStore(subscribeNotice, getNotice)
  const { busy, run } = useMutation()
  const [theme, setTheme] = useState<'dark' | 'light'>(() => {
    try { return localStorage.getItem('yia.theme') === 'light' ? 'light' : 'dark' } catch { return 'dark' }
  })
  useLayoutEffect(() => {
    document.documentElement.dataset.theme = theme
    try { localStorage.setItem('yia.theme', theme) } catch { /* 禁用存储时仍可在当前页面切换外观。 */ }
  }, [theme])
  const appearance = useDailyAppearance(theme)
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
  const activeIdentity: Identity | null = adminArea ? (sessions.admin ? 'admin' : null) : sessions.user ? 'user' : sessions.admin ? 'admin' : null
  return <LeaveGuardContext.Provider value={registerLeaveGuard}>
    <a className="skip-link" href="#main">跳到正文</a>
    <div className="studio-shell">
    <aside className="site-sidebar">
      <Link className="brand" to="/" aria-label="Yia 博客首页"><span>Yia<span className="brand-dot">.</span></span><small>JOURNAL</small></Link>
      <p className="nav-label">你的记录空间</p>
      <nav className="main-nav" aria-label="主导航">
        <NavLink to="/" end><UiIcon name="book" /><span>文章<small>阅读与发现</small></span></NavLink>
        <NavLink to="/me/articles" className={location.pathname.startsWith('/write') || location.pathname.startsWith('/me') ? 'active' : undefined}><UiIcon name="edit" /><span>我的创作<small>从一个想法开始</small></span></NavLink>
        <NavLink to="/admin/articles" className={adminArea ? 'active' : undefined}><UiIcon name="folder" /><span>管理<small>整理内容与交流</small></span></NavLink>
      </nav>
      <div className="sidebar-bottom">
        <div className="sidebar-writing-note"><UiIcon name="edit" size={19} /><p>给想法一点时间，<br />也给它一个落脚点。</p><Link className="text-link" to={activeIdentity === 'admin' ? '/admin/articles/new' : '/write/new'}>写下一篇记录 <UiIcon name="arrow" size={15} /></Link></div>
        <span className="sidebar-signature">Keep curious. Keep creating.</span>
        <span className="sidebar-theme"><span className="tiny-dot" />{appearance ? `每日拾色 · ${appearance.date.slice(5).replace('-', '.')}` : theme === 'dark' ? '暖夜咖啡' : '暖纸日光'}</span>
      </div>
    </aside>
    <div className="studio-workspace">
    <header className="site-header">
      <div className="container header-inner">
        <div className="page-breadcrumb"><span>Yia Journal</span><span aria-hidden="true">/</span><strong>{title}</strong></div>
        <div className="header-actions">
          <button className="icon-button theme-toggle" aria-label={theme === 'dark' ? '切换到浅色外观' : '切换到深色外观'} title={theme === 'dark' ? '切换到浅色外观' : '切换到深色外观'} onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}><UiIcon name={theme === 'dark' ? 'sun' : 'moon'} size={18} /></button>
          {activeIdentity === 'admin' && sessions.admin ? <>
            <Link className="account-profile" to="/admin/profile" aria-label="管理员个人资料"><Avatar url={sessions.admin.profile.avatar} name={sessions.admin.profile.admin_name} /><span className="account-name">{sessions.admin.profile.admin_name}</span></Link>
            <button className="icon-button" disabled={busy} aria-label="退出管理员登录" onClick={() => logout('admin')}><UiIcon name="logout" /></button>
          </> : activeIdentity === 'user' && sessions.user ? <>
            <Link className="account-profile" to="/me/profile" aria-label="用户个人资料"><Avatar url={sessions.user.profile.avatar} name={sessions.user.profile.nick_name} /><span className="account-name">{sessions.user.profile.nick_name}</span></Link>
            <button className="icon-button" disabled={busy} aria-label="退出用户登录" onClick={() => logout('user')}><UiIcon name="logout" /></button>
          </> : <Link className="login-link" to={adminArea ? '/admin/login' : '/login'}>登录</Link>}
          <Link className="button primary small write-link" to={activeIdentity === 'admin' ? '/admin/articles/new' : '/write/new'}><UiIcon name="edit" size={16} />写文章</Link>
        </div>
      </div>
    </header>
    <main id="main" tabIndex={-1}>
      <Suspense fallback={<StatePanel kind="loading" />}>
        <div className="page-transition" key={location.pathname}><Outlet /></div>
      </Suspense>
    </main>
    <footer className="site-footer container"><Link className="footer-brand" to="/">Yia.</Link><span>保持好奇，持续记录。</span><span className="footer-end">用文字留下思考 · {new Date().getFullYear()}</span></footer>
    </div>
    </div>
    {notice && <div className={`toast ${notice.kind}`} role={notice.kind === 'error' ? 'alert' : 'status'}>
      <UiIcon name={notice.kind === 'success' ? 'check' : 'comment'} /><span>{notice.text}</span>
      <button className="icon-button" aria-label="关闭提示" onClick={dismissNotice}><UiIcon name="close" size={16} /></button>
    </div>}
    <ScrollRestoration />
  </LeaveGuardContext.Provider>
}
