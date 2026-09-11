import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { authApi } from '../api'
import { errorMessage } from '../api/http'
import type { Identity } from '../api/types'
import { useMutation } from '../hooks/useMutation'
import { useSessions } from '../hooks/useSessions'
import { safeNext } from '../lib/article'
import { saveSession } from '../stores/session'
import { notify } from '../stores/notice'
import UiIcon from '../components/UiIcon'

export default function AuthView({ identity, register = false }: { identity: Identity; register?: boolean }) {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const sessions = useSessions()
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const { busy, run } = useMutation(`${identity}:${register}`)
  const heading = register ? '开始你的第一篇记录' : identity === 'admin' ? '回到内容工作台' : '欢迎回来'
  const destination = safeNext(params.get('next'), identity === 'admin' ? '/admin/articles' : '/me/articles')
  const nextQuery = params.has('next') ? `?next=${encodeURIComponent(destination)}` : ''

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy) return
    setError('')
    if (!name.trim()) { setError('请填写账号名称。'); return }
    if (register && password !== confirm) { setError('两次输入的密码不一致。'); return }
    if (register && (!/[a-zA-Z]/.test(password) || !/\d/.test(password))) { setError('密码需要同时包含字母和数字。'); return }
    if (register) {
      void run(() => authApi.register(name.trim(), password), () => {
        setPassword(''); setConfirm('')
        notify('注册成功，现在可以登录开始写作了。')
        void navigate(`/login${nextQuery}`, { replace: true })
      }, (cause) => setError(errorMessage(cause)))
    } else {
      // 身份由本次表单闭包固定；切换登录页面后 useMutation 会丢弃旧响应。
      void run(() => authApi.login(identity, name.trim(), password), (result) => {
        saveSession(identity, result)
        setPassword('')
        notify('登录成功，欢迎回来。')
        void navigate(destination, { replace: true })
      }, (cause) => setError(errorMessage(cause)))
    }
  }

  return <div className="container auth-page">
    <section className="auth-story">
      <p className="eyebrow">A SPACE FOR YOUR IDEAS</p><h1>每一次记录，<br />都<span className="serif-accent">算数</span>。</h1><p>把今天想通的事留下来，<br />明天就能走得更远一点。</p>
      <div className="auth-illustration" aria-hidden="true"><div className="paper-sheet"><span className="paper-title">A little note.</span><i /><i /><i /><span className="paper-sign">y.</span></div><div className="paper-leaf">✳</div></div>
      <span className="auth-footnote">WRITE SOMETHING WORTH REMEMBERING.</span>
    </section>
    <section className="auth-form-panel">
      <Link className="back-link" to="/"><UiIcon name="back" size={17} />返回博客</Link>
      <p className="eyebrow">{register ? 'CREATE AN ACCOUNT' : identity === 'admin' ? 'ADMIN SIGN IN' : 'WELCOME BACK'}</p>
      <h2>{heading}</h2><p className="muted">{register ? '创建账号，开始阅读、交流与写作。' : identity === 'admin' ? '使用管理员账号，整理内容与审核评论。' : '登录后继续写作，也和他人交换想法。'}</p>
      {sessions[identity] && <p className="inline-note">当前已有登录账号。<Link className="text-link" to={destination}>进入工作区</Link></p>}
      <form className="stack-form" onSubmit={submit}>
        <label>{identity === 'admin' ? '管理员名称' : '昵称'}<input value={name} onChange={(event) => setName(event.target.value)} disabled={busy} autoComplete="username" maxLength={16} required placeholder={identity === 'admin' ? '输入管理员名称' : '你的昵称'} /></label>
        <label>密码<input value={password} onChange={(event) => setPassword(event.target.value)} disabled={busy} type="password" autoComplete={register ? 'new-password' : 'current-password'} minLength={6} maxLength={20} required placeholder="输入密码" />{register && <small>6–20 位，包含字母和数字。</small>}</label>
        {register && <label>确认密码<input value={confirm} onChange={(event) => setConfirm(event.target.value)} disabled={busy} type="password" autoComplete="new-password" minLength={6} maxLength={20} required placeholder="再输入一次密码" /></label>}
        {error && <p className="form-error" role="alert">{error}</p>}
        <button className="button primary full" disabled={busy} type="submit">{busy ? '正在处理…' : register ? '创建账号' : '登录'}{!busy && <UiIcon name="arrow" size={18} />}</button>
      </form>
      {identity === 'user' && <p className="auth-switch">{register ? '已经有账号？' : '还没有账号？'}<Link to={`${register ? '/login' : '/register'}${nextQuery}`}>{register ? '直接登录' : '注册一个'}</Link></p>}
      <div className="auth-divider" /><Link className="subtle-link" to={identity === 'admin' ? '/login' : '/admin/login'}>{identity === 'admin' ? '使用普通用户账号登录' : '管理员入口'} <span>↗</span></Link>
    </section>
  </div>
}
