import HomeUpdates from '../components/HomeUpdates'
import { useI18n } from '../i18n'

export default function DashboardView() {
  const { t } = useI18n()
  return <div className="container workspace-page dashboard-page">
    <div className="workspace-heading"><div><p className="eyebrow">YIA / UPDATES</p><h1>{t('数据看板')}</h1></div></div>
    <HomeUpdates />
  </div>
}
