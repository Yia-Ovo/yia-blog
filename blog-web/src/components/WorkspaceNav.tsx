import { useI18n } from '../i18n'
import { NavLink } from 'react-router-dom'
import type { Identity } from '../api/types'
import { useSessions } from '../hooks/useSessions'
import UiIcon from './UiIcon'

export default function WorkspaceNav({ identity }: { identity: Identity }) {
  const { t } = useI18n()
  const sessions = useSessions()
  return <>
    <div className="workspace-heading">
      <div><p className="eyebrow">{identity === 'admin' ? 'CONTENT STUDIO' : 'MY WRITING SPACE'}</p><h1>{identity === 'admin' ? t("内容管理") : t("我的创作空间")}<span className="accent-dot">.</span></h1></div>
      <p className="muted">{identity === 'admin' ? t("整理文章，也照顾每一次交流。") : t("让零散的想法，慢慢长成文章。")}</p>
    </div>
    <nav className="workspace-tabs" aria-label={t("创作导航")}>
      <NavLink to={identity === 'admin' ? '/admin/articles' : '/me/articles'}><UiIcon name="book" size={17} />{identity === 'admin' ? t("文章管理") : t("我的文章")}</NavLink>
      {identity === 'admin' && <NavLink to="/admin/comments"><UiIcon name="comment" size={17} />{t("评论审核")}</NavLink>}
      {identity === 'admin' && sessions.admin?.profile.role === 0 && <NavLink to="/admin/categories"><UiIcon name="folder" size={17} />{t("分类管理")}</NavLink>}
    </nav>
  </>
}
