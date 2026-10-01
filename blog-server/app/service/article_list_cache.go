package service

import (
	"context"
	"encoding/json"
	"time"

	"blog-server/common"
	"blog-server/domain/do"
	domainservice "blog-server/domain/service"
)

// articleListItem 只保存列表展示字段与作者引用，不将正文或作者资料复制进分页缓存。
type articleListItem struct {
	ID         uint64
	UserID     uint64
	AdminID    uint64
	Title      string
	Cover      string
	Summary    string
	CategoryID uint64
	Status     int
	Sort       int
	Views      uint64
	Likes      int
	Comments   int
	IsTop      int
	PublishAt  *time.Time
	CreateAt   time.Time
	UpdateAt   time.Time
}

type articleCachedPage struct {
	Items []articleListItem
	Total int64
}

func newArticleCachedPage(articles []do.ArticleDO, total int64) articleCachedPage {
	items := make([]articleListItem, 0, len(articles))
	for _, article := range articles {
		items = append(items, articleListItem{
			ID: article.ID, UserID: article.UserID, AdminID: article.AdminID,
			Title: article.Title, Cover: article.Cover, Summary: article.Summary,
			CategoryID: article.CategoryID, Status: article.Status, Sort: article.Sort,
			Views: article.Views, Likes: article.Likes, Comments: article.Comments,
			IsTop: article.IsTop, PublishAt: article.PublishAt,
			CreateAt: article.CreateAt, UpdateAt: article.UpdateAt,
		})
	}
	return articleCachedPage{Items: items, Total: total}
}

func validArticleCachedPage(page articleCachedPage, pageSize int) bool {
	if page.Total < 0 || int64(len(page.Items)) > page.Total || len(page.Items) > pageSize {
		return false
	}
	seen := make(map[uint64]struct{}, len(page.Items))
	for _, item := range page.Items {
		if item.ID == 0 {
			return false
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return false
		}
		seen[item.ID] = struct{}{}
	}
	return true
}

func (s *ArticleAppService) readPublicArticlePage(ctx context.Context, query *do.ArticleQuery) (articleCachedPage, error) {
	if err := domainservice.NormalizePublicArticleQuery(query); err != nil {
		return articleCachedPage{}, err
	}
	load := func(ctx context.Context) (articleCachedPage, error) {
		items, total, err := s.articleDomainService.ListPublicArticleByPage(ctx, query)
		return newArticleCachedPage(items, total), err
	}
	if s.cache == nil || s.cache.backend == nil {
		return load(ctx)
	}
	// 使用领域归一化后的所有条件，保留未传筛选与显式零值的区别。
	key, err := json.Marshal(query)
	if err != nil {
		return articleCachedPage{}, common.ParamErr
	}
	page, err := readThrough(ctx, s.cache, "list:"+string(key), nil, load, func(page articleCachedPage) bool {
		return validArticleCachedPage(page, query.PageSize)
	})
	if err != nil {
		return articleCachedPage{}, err
	}
	ids := make([]uint64, len(page.Items))
	for index, item := range page.Items {
		ids[index] = item.ID
	}
	current, err := s.articleDomainService.ValidatePublicArticlePage(ctx, query, ids, page.Total)
	if err != nil {
		return articleCachedPage{}, err
	}
	if !current {
		// 下架、排序或其他页变化会影响补位与总数，必须整页回源，不能仅过滤旧项。
		// 本次直接返回数据库结果，后续请求经正常代次检查回填，避免无界重试。
		s.cache.invalidate()
		return load(ctx)
	}
	return page, nil
}
