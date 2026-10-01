package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"slices"

	"go.uber.org/zap"
)

// NormalizeArticleQuery 让数据库查询、响应分页和缓存键采用相同的有效参数。
// 不改筛选字段，保持管理列表可以查询草稿等状态的既有语义。
func NormalizeArticleQuery(query *do.ArticleQuery) error {
	if query == nil {
		return common.ParamErr
	}
	if query.PageNum < 1 {
		query.PageNum = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 10
	}
	return nil
}

// NormalizePublicArticleQuery 限定公开范围，客户端提交的状态不能扩大可见性。
func NormalizePublicArticleQuery(query *do.ArticleQuery) error {
	if err := NormalizeArticleQuery(query); err != nil {
		return err
	}
	published := consts.ArticleStatusPublished
	query.Status = &published
	return nil
}

// ValidatePublicArticlePage 核对当前公开列表的完整分页身份，防止旧页过滤后缺项或总数错误。
// 比较有序 IDs 和 Total，而非逐篇检查状态：前一页变化也可能导致当前页整体偏移。
// 核对之后的并发写入仍遵循普通数据库读取的时间窗口，不提供数据库与缓存的强一致性。
func (a *ArticleDomainService) ValidatePublicArticlePage(ctx context.Context, query *do.ArticleQuery, ids []uint64, total int64) (bool, error) {
	if err := NormalizePublicArticleQuery(query); err != nil {
		return false, err
	}
	reader, ok := a.articleMysqlRepo.(repo.ArticlePageStateReader)
	if !ok {
		items, currentTotal, err := a.ListPublicArticleByPage(ctx, query)
		if err != nil {
			return false, err
		}
		currentIDs := make([]uint64, len(items))
		for i, item := range items {
			currentIDs[i] = item.ID
		}
		return total == currentTotal && slices.Equal(ids, currentIDs), nil
	}
	state, err := reader.GetArticlePageState(ctx, query)
	if err != nil || state == nil {
		logger.Error("[DOMAIN] 核对文章公开分页失败", zap.Error(err))
		return false, common.DatabaseErr
	}
	return total == state.Total && slices.Equal(ids, state.IDs), nil
}
