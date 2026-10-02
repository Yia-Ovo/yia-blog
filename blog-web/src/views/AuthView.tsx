import { useI18n } from '../i18n'
import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { authApi } from '../api'
import { errorMessage } from '../api/http'
import type { Identity } from '../api/types'
import { useMutation } from '../hooks/useMutation'
import { useSessions } from '../hooks/useSessions'
import { safeNext } from '../lib/article'
import { getSessions, saveSession } from '../stores/session'
import { notify } from '../stores/notice'
import UiIcon from '../components/UiIcon'
import Avatar from '../components/Avatar'

export default function AuthView({ identity, register = false }: { identity: Identity; register?: boolean }) {
  const { t } = useI18n()
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const sessions = useSessions()
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [showOtherForm, setShowOtherForm] = useState(false)
  const { busy, run } = useMutation(`${identity}:${register}:${sessions.user?.token || ''}:${sessions.admin?.token || ''}:${showOtherForm}`)
  const otherIdentity = identity === 'admin' ? 'user' : 'admin'
  const accountIdentity = sessions[identity] ? identity : otherIdentity
  const account = sessions[accountIdentity]
  const sameIdentity = !!sessions[identity]
  const showAccount = !!account && (sameIdentity || !showOtherForm)
  const accountLabel = accountIdentity === 'admin' ? '管理员' : '普通用户'
  const accountName = account?.profile[accountIdentity === 'admin' ? 'admin_name' : 'nick_name'] || t(accountLabel)
  const accountWorkspace = accountIdentity === 'admin' ? '/admin/articles' : '/me/articles'
  const formLabel = identity === 'admin' ? '管理员' : '普通用户'
  const heading = showAccount ? '继续使用当前账号' : register ? '开始你的第一篇记录' : identity === 'admin' ? '回到内容工作台' : '欢迎回来'
  const destination = safeNext(params.get('next'), identity === 'admin' ? '/admin/articles' : '/me/articles')
  const nextQuery = params.has('next') ? `?next=${encodeURIComponent(destination)}` : ''

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const startedSessions = getSessions()
    if (busy || startedSessions[identity]) return
    // 同一批渲染中登录态也可能更新；迟到响应不能覆盖新账号或切换后的选择。
    const sessionUnchanged = () => getSessions().user?.token === startedSessions.user?.token && getSessions().admin?.token === startedSessions.admin?.token
    setError('')
    if (!name.trim()) { setError('请填写账号名称。'); return }
    if (register && password !== confirm) { setError('两次输入的密码不一致。'); return }
    if (register && (!/[a-zA-Z]/.test(password) || !/\d/.test(password))) { setError('密码需要同时包含字母和数字。'); return }
    if (register) {
      void run(() => authApi.register(name.trim(), password), () => {
        if (!sessionUnchanged()) return
        setPassword(''); setConfirm('')
        notify('注册成功，现在可以登录开始写作了。')
        void navigate(`/login${nextQuery}`, { replace: true })
      }, (cause) => { if (sessionUnchanged()) setError(errorMessage(cause)) })
    } else {
      // 身份由本次表单闭包固定；切换登录页面后 useMutation 会丢弃旧响应。
      void run(() => authApi.login(identity, name.trim(), password), (result) => {
        if (!sessionUnchanged()) return
        saveSession(identity, result)
        setPassword('')
        notify('登录成功，欢迎回来。')
        void navigate(destination, { replace: true })
      }, (cause) => { if (sessionUnchanged()) setError(errorMessage(cause)) })
    }
  }

  return <div className="container auth-page">
    <section className="auth-story">
      <p className="eyebrow">{t('灵感的一方天地')}</p><h1>{t('每一次记录，')}<br /><span className="serif-accent">{t('都算数。')}</span></h1><p>{t('把今天想通的事留下来，明天就能走得更远一点。')}</p>
      <div className="auth-illustration" aria-hidden="true"><span>{t('小小空间。')}<br />{t('无限灵感。')}</span></div>
      <span className="auth-footnote">{t('写下值得记住的事。')}</span>
    </section>
    <section className="auth-form-panel">
      <Link className="back-link" to="/"><UiIcon name="back" size={17} />{t('返回博客')}</Link>
      <p className="eyebrow">{t(showAccount ? '你的账号' : register ? '创建账号' : identity === 'admin' ? '管理员登录' : '欢迎回来')}</p>
      <h2>{t(heading)}</h2><p className="muted">{showAccount ? sameIdentity ? t('无需重复登录，可以继续访问。') : t('你当前使用的是{account}账号，这里是{form}{action}入口。', { account: t(accountLabel), form: t(formLabel), action: t(register ? '注册' : '登录') }) : t(register ? '创建账号，开始阅读、交流与写作。' : identity === 'admin' ? '使用管理员账号，整理内容与审核评论。' : '登录后继续写作，也和他人交换想法。')}</p>
      {showAccount && account ? <>
        <div className="auth-session-card"><Avatar url={account.profile.avatar} name={accountName} /><div><strong>{accountName}</strong><span>{t('{name}账号', { name: t(accountLabel) })}</span></div></div>
        <div className="stack-form">
          <Link className="button primary full" to={sameIdentity ? destination : accountWorkspace}>{t(sameIdentity ? '继续访问' : accountIdentity === 'admin' ? '进入管理员工作区' : '进入我的创作')}<UiIcon name="arrow" size={18} /></Link>
          {!sameIdentity && <button className="button secondary full" type="button" onClick={() => setShowOtherForm(true)}>{register ? t('注册普通用户账号') : t('登录{name}账号', { name: t(formLabel) })}</button>}
        </div>
        <p className="small-help">{t(sameIdentity ? '如需更换账号，请先退出当前登录。' : identity === 'user' ? '普通用户账号用于个人创作和发表评论，可与管理员账号同时登录。' : '管理功能需要单独登录管理员账号。')}</p>
      </> : <>
      {account && <p className="inline-note">{t('当前仍使用{account}账号「{name}」。下方仅用于{action}{form}账号。', { account: t(accountLabel), name: accountName, action: t(register ? '注册' : '登录'), form: t(formLabel) })}<Link className="text-link" to={accountWorkspace}>{t('返回工作区')}</Link></p>}
      <form className="stack-form" onSubmit={submit}>
        <label>{t(identity === 'admin' ? '管理员名称' : '昵称')}<input value={name} onChange={(event) => setName(event.target.value)} disabled={busy} autoComplete="username" maxLength={16} required placeholder={t(identity === 'admin' ? '输入管理员名称' : '你的昵称')} /></label>
        <label>{t('密码')}<input value={password} onChange={(event) => setPassword(event.target.value)} disabled={busy} type="password" autoComplete={register ? 'new-password' : 'current-password'} minLength={6} maxLength={20} required placeholder={t('输入密码')} />{register && <small>{t('6–20 位，包含字母和数字。')}</small>}</label>
        {register && <label>{t('确认密码')}<input value={confirm} onChange={(event) => setConfirm(event.target.value)} disabled={busy} type="password" autoComplete="new-password" minLength={6} maxLength={20} required placeholder={t('再输入一次密码')} /></label>}
        {error && <p className="form-error" role="alert">{t(error)}</p>}
        <button className="button primary full" disabled={busy} type="submit">{t(busy ? '正在处理…' : register ? '创建账号' : '登录')}{!busy && <UiIcon name="arrow" size={18} />}</button>
      </form>
      {identity === 'user' && <p className="auth-switch">{t(register ? '已经有账号？' : '还没有账号？')}<Link to={`${register ? '/login' : '/register'}${nextQuery}`}>{t(register ? '直接登录' : '注册一个')}</Link></p>}
      </>}
      <div className="auth-divider" /><Link className="subtle-link" to={identity === 'admin' ? '/login' : '/admin/login'}>{t(identity === 'admin' ? '使用普通用户账号登录' : '管理员入口')} <span>↗</span></Link>
    </section>
  </div>
}
