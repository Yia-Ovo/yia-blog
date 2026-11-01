export type AnnouncementStatus = 'default' | 'progress' | 'success' | 'warning' | 'error'

// 公告内容集中维护；完成状态由维护者确认，不根据日期自动判定。
export const announcementOverview = {
  date: '2026-11-02',
  title: 'Yia Blog 第一版正式上线',
  description: '2026 年 11 月 2 日，Yia Blog 第一版正式上线，和大家分享这个持续构建的小站。',
  status: 'success' as AnnouncementStatus,
}

export const announcements: { date: string; title: string; description: string; status: AnnouncementStatus }[] = [
  { date: '2026-11-02', title: '第一版正式上线', description: 'Yia Blog 第一版已上线，开启新的记录。', status: 'success' },
  { date: '2026-11-02', title: 'GitHub 仓库公开', description: '项目源码与 README 已公开，分享开发过程。', status: 'success' },
  { date: '2026-11-02', title: 'Instagram · X 宣传', description: '已在 Instagram、X 等平台分享上线消息与项目介绍。', status: 'success' },
]
