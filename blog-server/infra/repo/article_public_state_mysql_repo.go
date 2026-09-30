package repo

import (
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ repo.ArticlePublicStateReader = (*articleMysqlRepo)(nil)

// GetArticlePublicState 只读取公开可见性所需字段，并保留 ArticleModel 的软删除查询条件。
func (a *articleMysqlRepo) GetArticlePublicState(ctx context.Context, id uint64) (*do.ArticlePublicState, error) {
	var article model.ArticleModel
	err := a.db.WithContext(ctx).Select("id", "status").First(&article, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		logger.Error("[MYSQL] 查询文章公开状态失败", zap.Uint64("article_id", id), zap.Error(err))
		return nil, err
	}
	return &do.ArticlePublicState{ID: article.ID, Status: article.Status}, nil
}
