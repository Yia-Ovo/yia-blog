import { novel } from '../lib/novel'
import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { articleApi, categoryApi } from '../api'
import { useResource } from '../hooks/useResource'
import { useI18n } from '../i18n'
import { formatDate, positiveNumber } from '../lib/article'
import { safeImageURL } from '../lib/markdown'
import Pagination from '../components/Pagination'
import StatePanel from '../components/StatePanel'
import UiIcon from '../components/UiIcon'

export default function HomeView() {
  const { t } = useI18n()
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const page = positiveNumber(params.get('page'))
  const category = positiveNumber(params.get('category'), 0)
  const title = params.get('q') || ''
  const [search, setSearch] = useState(title)
  useEffect(() => { setSearch(title) }, [title])
  const articles = useResource(useCallback((signal) => articleApi.list({ exclude_ids: novel.chapters.map((chapter) => chapter.id).join(','), page_num: page, page_size: 6, title, category_id: category || undefined }, signal), [page, title, category]))
  const categories = useResource(categoryApi.all)
  const categoryName = (id: number) => categories.data?.find((item) => item.id === id)?.name || t(id ? '文章' : '未分类')

  function go(nextPage = 1, nextCategory = category, query = search) {
    const next = new URLSearchParams()
    if (nextPage > 1) next.set('page', String(nextPage))
    if (nextCategory) next.set('category', String(nextCategory))
    if (query.trim()) next.set('q', query.trim())
    void navigate({ pathname: '/', search: next.toString(), hash: '#articles' })
  }

  return <div className="container home-page">
    <div className="journal-heading"><div><p className="eyebrow">A PERSONAL CORNER OF THE INTERNET</p><h2>{t('记录，思考，再出发。')}</h2></div><span>YIA'S JOURNAL <span aria-hidden="true">✳</span></span></div>
    <section className="hero" aria-labelledby="hero-title">
      <div className="hero-copy">
        <span className="hero-signature">A little space for big ideas.</span>
        <p className="eyebrow"><span className="tiny-dot" /> YIA'S JOURNAL</p>
        <h1 id="hero-title">{t('把想法写下来，')}<br /><span className="serif-accent">{t('把问题想明白。')}</span></h1>
        <p className="hero-description">{t('关于 Go、工程实践，也关于一路上的发现。')}<br className="desktop-break" />{' '}{t('记录走过的弯路，留下一点向前的痕迹。')}</p>
        <div className="hero-actions"><a className="button primary" href="#articles">{t('开始阅读')} <UiIcon name="arrow" size={18} /></a><Link className="text-link" to="/write/new">{t('写下新的想法')} <UiIcon name="edit" size={16} /></Link></div>
        <p className="hero-caption"><span /> {t('保持好奇，持续构建')}</p>
      </div>
      <div className="hero-art" aria-hidden="true" />
      <span className="hero-art-label" aria-hidden="true">THE QUIET HOURS<br /><small>write · learn · repeat</small></span>
    </section>

    <div id="articles" className="content-layout">
      <section className="article-feed" aria-labelledby="feed-title">
        <div className="section-heading"><div><p className="eyebrow">THE LATEST NOTES</p><h2 id="feed-title">{title ? t('搜索结果') : category ? categoryName(category) : t('最近的记录')}{articles.data && <span className="count-label">{t('{count} 篇', { count: articles.data.total })}</span>}</h2></div><span className="heading-rule" /></div>
        <form className="search-box" role="search" onSubmit={(event) => { event.preventDefault(); go() }}>
          <UiIcon name="search" size={19} /><label className="sr-only" htmlFor="article-search">{t('搜索文章标题')}</label>
          <input id="article-search" value={search} onChange={(event) => setSearch(event.target.value)} type="search" placeholder={t('搜索文章标题…')} maxLength={128} /><button type="submit">{t('搜索')}</button>
        </form>
        <div className="filter-row" aria-label={t('按分类筛选')}>
          <button className={category === 0 ? 'selected' : ''} aria-pressed={category === 0} onClick={() => go(1, 0)}>{t('全部文章')}</button>
          {categories.data?.map((item) => <button key={item.id} className={category === item.id ? 'selected' : ''} aria-pressed={category === item.id} onClick={() => go(1, item.id)}>{item.name}</button>)}
        </div>
        {categories.error && <p className="inline-note">{t('分类暂时没有加载成功。')}<button className="link-button" onClick={categories.reload}>{t('重试')}</button></p>}
        {articles.pending ? <StatePanel kind="loading" message={t('把文章整理好，马上就来。')} />
          : articles.error ? <StatePanel kind="error" message={t(articles.error)} onRetry={articles.reload} />
          : !articles.data?.list.length ? <StatePanel kind="empty" title={t(title || category ? '还没有找到匹配的文章' : '第一篇文章，从这里开始')} message={t(title || category ? '换一个标题关键词，或试试其他分类。' : '写一篇草稿，给今天的思考留个位置。')}>
            {title || category ? <button className="button secondary small" onClick={() => go(1, 0, '')}>{t('查看全部')}</button> : <Link className="button primary small" to="/write/new">{t('开始写作')}</Link>}
          </StatePanel>
          : <div className="article-list">{articles.data.list.map((article, index) => <article key={article.id} className={`article-card${index === 0 && page === 1 && !title && !category ? ' featured' : ''}`}>
            <div className="article-card-copy">
              <div className="article-meta"><span className="category-tag">{categoryName(article.category_id)}</span>{article.is_top === 1 && <span className="pin-label">{t('置顶')}</span>}<span>{formatDate(article.publish_at || article.created_at || article.create_at)}</span></div>
              <h3><Link to={`/articles/${article.id}`}>{article.title}</Link></h3>
              <p>{article.summary || t('打开文章，看看这次记录了什么。')}</p>
              <div className="article-bottom"><span>{article.author?.name || t('作者')}</span><Link className="read-link" to={`/articles/${article.id}`}>{t('阅读全文')} <UiIcon name="arrow" size={16} /></Link></div>
            </div>
            <Link to={`/articles/${article.id}`} className={`article-thumb thumb-${index % 3}`} tabIndex={-1} aria-hidden="true">
              {safeImageURL(article.cover) ? <img src={safeImageURL(article.cover)} alt="" loading="lazy" referrerPolicy="no-referrer" /> : <><span className="thumb-grid" /><UiIcon name={index % 2 === 0 ? 'code' : 'book'} size={40} /><small>YIA / NOTES</small></>}
            </Link>
          </article>)}</div>}
        {articles.data && <Pagination page={articles.data.pageNum} pages={articles.data.pages} busy={articles.pending} onChange={(next) => go(next)} />}
      </section>

      <aside className="home-sidebar">
        <div className="about-card">
          <div className="about-top"><span className="profile-monogram">y.</span><span className="side-note">HELLO, WORLD</span></div>
          <h3>{t('你好，我是 Yia')}<span className="accent-dot">.</span></h3><p>{t('用代码把问题一步步解决。')}<br />{' '}{t('这里记录学习、实践，')}<br />{' '}{t('以及回头看时的新理解。')}</p>
          <div className="about-tags"><span>Go</span><span>{t('工程实践')}</span><span>{t('持续学习')}</span></div><div className="about-signature">Keep building, keep writing.<span>↗</span></div>
        </div>
        <div className="sidebar-note"><span className="small-icon"><UiIcon name="edit" /></span><h3>{t('给想法一个落脚点')}</h3><p>{t('一个问题、一段代码，')}<br />{' '}{t('都值得好好记下来。')}</p><Link className="text-link" to="/write/new">{t('开始一篇草稿')} <UiIcon name="arrow" size={16} /></Link></div>
        <div className="sidebar-footer"><span className="tiny-dot" /><span>{t('慢慢积累，也慢慢生长。')}</span></div>
      </aside>
    </div>
  </div>
}
