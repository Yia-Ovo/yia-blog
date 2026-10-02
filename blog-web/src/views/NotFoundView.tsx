import { Link } from 'react-router-dom'
import StatePanel from '../components/StatePanel'
import { useI18n } from '../i18n'

export default function NotFoundView() {
  const { t } = useI18n()
  return <div className="container reading-page"><StatePanel kind="empty" title={t('这页记录还不存在')} message={t('链接可能有误，回到首页继续看看吧。')}><Link className="button primary small" to="/">{t('返回首页')}</Link></StatePanel></div>
}
