package repo

import (
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ repo.ArticlePageStateReader = (*articleMysqlRepo)(nil)

// GetArticlePageState 只统计总数并读取有序分页 ID，保留完整列表的筛选和软删除条件。
// 两条查询与原列表一样不构成事务快照；此能力用于发现旧缓存页，不承诺强一致性。
func (a *articleMysqlRepo) GetArticlePageState(ctx context.Context, query *do.ArticleQuery) (*do.ArticlePageState, error) {
	if query == nil {
		return nil, gorm.ErrInvalidData
	}
	state := &do.ArticlePageState{IDs: make([]uint64, 0)}
	db := articleQueryScope(a.db.WithContext(ctx), query)
	if err := db.Count(&state.Total).Error; err != nil {
		logger.Error("[MYSQL] 文章分页身份统计失败", zap.Error(err))
		return nil, err
	}
	if state.Total == 0 {
		return state, nil
	}
	err := db.Order(articlePageOrder).
		Limit(query.PageSize).
		Offset((query.PageNum-1)*query.PageSize).
		Pluck("id", &state.IDs).Error
	if err != nil {
		logger.Error("[MYSQL] 文章分页身份查询失败", zap.Error(err))
		return nil, err
	}
	return state, nil
}
