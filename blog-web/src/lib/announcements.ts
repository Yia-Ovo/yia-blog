export type AnnouncementStatus = 'default' | 'progress' | 'success' | 'warning' | 'error'

// 公告内容集中维护；完成状态由维护者确认，不根据日期自动判定。
export const announcementOverview = {
  date: '2026-11-06',
  title: '社区反馈已记录',
  description: '感谢 X 社区反馈。阅读链接无障碍建议已列入待处理计划，后续进展会在这里更新。',
  status: 'success' as AnnouncementStatus,
}

export const announcements: { date: string; title: string; description: string; status: AnnouncementStatus; href?: string }[] = [
  { date: '2026-11-06', title: '阅读链接无障碍优化', description: '计划为“阅读全文”补充文章标题，方便屏幕阅读器用户区分链接。详见 GitHub Issue，处理时间待定。', status: 'default', href: 'https://github.com/Yia-Ovo/yia-blog/issues/1' },
  { date: '2026-11-06', title: '草稿恢复提醒获肯定', description: '收到对草稿恢复差异提醒的正面反馈。后续改动会保留这项保护，感谢认真体验与建议。', status: 'success' },
  { date: '2026-11-06', title: '首次访问体验优化', description: '精简首页加载资源，减少首次访问的下载量与等待。', status: 'success' },
  { date: '2026-11-02', title: '第一版正式上线', description: 'Yia Blog 第一版已上线，开启新的记录。', status: 'success' },
  { date: '2026-11-02', title: 'GitHub 仓库公开', description: '项目源码与 README 已公开，分享开发过程。', status: 'success' },
  { date: '2026-11-02', title: 'Instagram · X 宣传', description: '已在 Instagram、X 等平台分享上线消息与项目介绍。', status: 'success' },
]
