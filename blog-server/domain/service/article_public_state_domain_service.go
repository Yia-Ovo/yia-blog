package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/logger"
	"blog-server/domain/repo"
	"context"

	"go.uber.org/zap"
)

// EnsurePublicArticle 从数据库确认文章仍可公开，不信任缓存中可能过期的发布状态。
// 轻量读取避免加载正文；旧仓储不支持该能力时仍通过完整查询执行可见性规则。
func (a *ArticleDomainService) EnsurePublicArticle(ctx context.Context, id uint64) error {
	if id == 0 {
		return common.ParamErr
	}
	reader, ok := a.articleMysqlRepo.(repo.ArticlePublicStateReader)
	if !ok {
		_, err := a.GetPublicArticleByID(ctx, id)
		return err
	}
	state, err := reader.GetArticlePublicState(ctx, id)
	if err != nil {
		logger.Error("[DOMAIN] 查询文章公开状态失败", zap.Uint64("article_id", id), zap.Error(err))
		return common.DatabaseErr
	}
	if state == nil || state.ID != id || state.Status != consts.ArticleStatusPublished {
		return common.ArticleNotFoundErr
	}
	return nil
}
