import { useI18n } from '../i18n'
import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { authApi } from '../api'
import { errorMessage } from '../api/http'
import type { Identity } from '../api/types'
import Avatar from '../components/Avatar'
import UiIcon from '../components/UiIcon'
import { useMutation } from '../hooks/useMutation'
import { useSessions } from '../hooks/useSessions'
import { getSessions, refreshProfile } from '../stores/session'

const maxAvatarBytes = 5 * 1024 * 1024

export default function ProfileView({ identity }: { identity: Identity }) {
  const sessions = useSessions()
  const token = sessions[identity]?.token
  // 切换身份或重新登录时销毁旧预览，旧上传响应不能更新新账号。
  return token ? <ProfileEditor key={`${identity}:${token}`} identity={identity} token={token} /> : null
}

function ProfileEditor({ identity, token }: { identity: Identity; token: string }) {
  const { t } = useI18n()
  const sessions = useSessions()
  const profile = sessions[identity]?.profile
  const name = (identity === 'admin' ? profile?.admin_name : profile?.nick_name) || t('我的账号')
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState('')
  const [previewReady, setPreviewReady] = useState(false)
  const [formError, setFormError] = useState('')
  const [saved, setSaved] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  const { busy, run } = useMutation()

  useEffect(() => {
    if (!file) { setPreview(''); return }
    const url = URL.createObjectURL(file)
    setPreview(url)
    return () => URL.revokeObjectURL(url)
  }, [file])

  function selectFile(selected?: File) {
    if (busy || !selected) return
    setSaved(false); setFormError(''); setFile(null); setPreviewReady(false)
    if (!['image/jpeg', 'image/png'].includes(selected.type)) { setFormError('请选择 JPG 或 PNG 图片。'); return }
    if (!selected.size || selected.size > maxAvatarBytes) { setFormError('图片大小需要在 5 MB 以内，且不能为空。'); return }
    setFile(selected)
  }

  function save() {
    if (busy || !file || !previewReady) return
    setFormError(''); setSaved(false)
    void run(() => authApi.uploadAvatar(identity, file), ({ avatar }) => {
      const current = getSessions()[identity]
      if (current?.token !== token) return
      refreshProfile(identity, { ...current.profile, avatar }, token)
      setFile(null); setPreviewReady(false); setSaved(true)
      if (fileInput.current) fileInput.current.value = ''
    }, (cause) => setFormError(errorMessage(cause)))
  }

  return <div className="container workspace-page profile-page">
    <Link className="back-link" to={identity === 'admin' ? '/admin/articles' : '/me/articles'}><UiIcon name="back" size={17} />{t(identity === 'admin' ? '返回内容管理' : '返回我的创作')}</Link>
    <div className="workspace-heading"><div><p className="eyebrow">{t('你的个人空间')}</p><h1>{t('个人资料')}<span className="accent-dot">.</span></h1></div><p className="muted">{t('给你的记录，添上熟悉的面孔。')}</p></div>
    <section className="workspace-panel profile-panel" aria-labelledby="profile-avatar-heading">
      <div className="profile-identity"><Avatar className="profile-current-avatar" url={profile?.avatar} name={name} /><div><strong>{name}</strong><p>{t(identity === 'admin' ? '管理员账号' : '用户账号')}</p></div></div>
      <div className="panel-heading"><div><h2 id="profile-avatar-heading">{t('设置头像')}</h2><p>{t('选择一张喜欢的图片，让大家更容易认出你。')}</p></div></div>
      <form className="avatar-form" onSubmit={(event) => { event.preventDefault(); save() }} aria-busy={busy}>
        <div className="avatar-preview-area">
          {preview ? <span className="avatar avatar-preview"><img key={preview} src={preview} alt={t('新头像预览')} onLoad={() => setPreviewReady(true)} onError={() => { setPreviewReady(false); setFormError('无法读取这张图片，请重新选择 JPG 或 PNG 图片。') }} /></span>
            : <Avatar className="avatar-preview" url={profile?.avatar} name={name} />}
          <span>{t(preview ? '新头像预览' : '当前头像')}</span>
        </div>
        <div className="avatar-controls">
          <label className="avatar-file-label" htmlFor="avatar-file">{t('选择头像图片')}</label>
          <input ref={fileInput} id="avatar-file" type="file" accept="image/jpeg,image/png,.jpg,.jpeg,.png" disabled={busy} aria-describedby="avatar-help" onChange={(event) => { selectFile(event.currentTarget.files?.[0]); event.currentTarget.value = '' }} />
          <p id="avatar-help" className="small-help">{t('支持 JPG、PNG，最大 5 MB。图片将居中裁为正方形，并以圆形头像展示。')}</p>
          {file && <p className="avatar-file-name" title={file.name}>{file.name}</p>}
          <button className="button primary" type="submit" disabled={busy || !file || !previewReady}><UiIcon name="check" size={17} />{t(busy ? '正在保存…' : '保存头像')}</button>
          {formError && <p className="form-error" role="alert">{t(formError)}</p>}
          {saved && <p className="avatar-success" role="status"><UiIcon name="check" size={17} />{t('头像已更新。')}</p>}
        </div>
      </form>
    </section>
  </div>
}
