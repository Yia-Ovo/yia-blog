import { useI18n } from '../i18n'
import { announcementOverview, announcements, type AnnouncementStatus } from '../lib/announcements'
import UiIcon from './UiIcon'
import './home-updates.css'

const announcementLimit = 20
const announcementStatuses = [
  { value: 'default', label: '默认' },
  { value: 'progress', label: '进行中' },
  { value: 'success', label: '成功' },
  { value: 'warning', label: '警告' },
  { value: 'error', label: '异常' },
] as const
const statusLabel = (value: AnnouncementStatus) => announcementStatuses.find((status) => status.value === value)!.label
const shortDate = (date: string) => date.slice(5).replace('-', '.')

export default function HomeUpdates() {
  const { t } = useI18n()

  return <section className="home-updates" aria-labelledby="updates-title">
    <div className="home-updates-heading"><h2 id="updates-title">{t('小站动态')}</h2><p>{t('新的进展，也在这里记录。')}</p><span aria-hidden="true" /></div>
    <div className="home-updates-grid">
      <section className="home-update-card announcement-card" aria-labelledby="announcement-title">
        <header className="home-update-header">
          <div className="announcement-header-title"><h3 id="announcement-title"><UiIcon name="bell" size={17} />{t('系统公告')}</h3><span className="home-update-label">{t('最多显示 {count} 条', { count: announcementLimit })}</span></div>
          <ul className="announcement-status-legend" aria-label={t('公告状态说明')}>{announcementStatuses.map((status) => <li key={status.value} data-announcement-status={status.value}><span className="announcement-dot" aria-hidden="true" />{t(status.label)}</li>)}</ul>
        </header>
        <div className="announcement-body" role="region" aria-labelledby="announcement-title" tabIndex={0}>
          <div className="announcement-overview">
            <div className="announcement-date"><time dateTime={announcementOverview.date}>{shortDate(announcementOverview.date)} <span>{announcementOverview.date.slice(0, 4)}</span></time><span className="announcement-stage" data-announcement-status={announcementOverview.status}>{t(statusLabel(announcementOverview.status))}</span></div>
            <h4>{t(announcementOverview.title)}</h4>
            <p>{t(announcementOverview.description)}</p>
          </div>
          <ol className="announcement-plans">
            {announcements.slice(0, announcementLimit).map((item) => <li key={`${item.date}-${item.title}`} data-announcement-status={item.status}><span className="announcement-dot" aria-hidden="true" /><div className="announcement-entry"><strong>{t(item.title)}</strong><p>{t(item.description)}</p><time dateTime={item.date}>{shortDate(item.date)} · {t(statusLabel(item.status))}</time></div></li>)}
          </ol>
        </div>
        <footer className="home-update-footer">{t('动态公告功能未完成，当前展示维护者确认的公告。')}</footer>
      </section>

      <section className="home-update-card faq-card" aria-labelledby="faq-title">
        <header className="home-update-header"><h3 id="faq-title"><UiIcon name="question" size={17} />{t('常见问答')}</h3><span className="home-update-label unfinished-label">{t('未完成')}</span></header>
        <div className="faq-placeholder">
          <p>{t('正在整理小站的使用说明。')}</p>
          <ul>{['阅读与分类', '账号与评论', '写作与发布'].map((topic) => <li key={topic}><span>{t(topic)}</span><span className="faq-pending">{t('待整理')}</span></li>)}</ul>
        </div>
        <footer className="home-update-footer">{t('问答内容整理完成后开放。')}</footer>
      </section>

      <section className="home-update-card availability-card" aria-labelledby="availability-title">
        <header className="home-update-header"><h3 id="availability-title"><UiIcon name="activity" size={17} />{t('服务可用性')}</h3><span className="home-update-label unfinished-label">{t('未完成')}</span></header>
        <div className="availability-placeholder">
          <span className="availability-illustration" aria-hidden="true"><UiIcon name="activity" size={40} /><span /></span>
          <h4>{t('暂无监控数据')}</h4>
          <p>{t('服务监控尚未接入，暂不展示在线状态与可用率。')}</p>
        </div>
        <footer className="home-update-footer">{t('监控接入后开放。')}</footer>
      </section>
    </div>
  </section>
}
