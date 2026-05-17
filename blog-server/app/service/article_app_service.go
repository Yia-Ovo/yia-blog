package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"

	"go.uber.org/zap"
)

type ArticleAppService struct {
	articleDomainSrv *service.ArticleDomainService
}

// NewArticleAppService 创建文章应用服务实例（IOC注入）
func NewArticleAppService(
	articleMysqlRepo repo.ArticleMysqlRepo,
	articleRedisRepo repo.ArticleRedisRepo,
) *ArticleAppService {
	// 初始化领域服务
	articleDomainSrv := service.NewArticleDomainService(articleMysqlRepo, articleRedisRepo)

	return &ArticleAppService{
		articleDomainSrv: articleDomainSrv,
	}
}

// CreateArticle 创建文章
func (s *ArticleAppService) CreateArticle(ctx context.Context, req *dto.CreateArticleReq) error {
	// 1. 应用层基础参数校验
	if req == nil {
		return common.ParamErr.WithMsg("创建文章请求参数不能为空")
	}

	// 2. DTO 转换为 DO（领域对象）
	articleDO := &do.ArticleDO{
		AdminID:       req.AdminID,
		Title:         req.Title,
		Cover:         req.Cover,
		Summary:       req.Summary,
		Content:       req.Content,
		CategoryID:    req.CategoryID,
		TagIDs:        req.TagIDs,
		Status:        req.Status,
		Sort:          req.Sort,
		IsTop:         req.IsTop,
		CommentStatus: req.CommentStatus,
		PublishAt:     req.PublishAt,
	}

	// 3. 调用领域服务（核心逻辑交给领域层）
	err := s.articleDomainSrv.CreateArticle(ctx, articleDO)
	if err != nil {
		logger.Error("创建文章应用服务失败",
			zap.String("article_title", req.Title),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// ListArticleByPage 分页查询文章
func (s *ArticleAppService) ListArticleByPage(ctx context.Context, req *dto.ArticlePageQuery) (*dto.PageResult, error) {
	// 1. DTO 转换为领域查询对象
	query := &do.ArticleQuery{
		PageNum:    req.PageNum,
		PageSize:   req.PageSize,
		Title:      req.Title,
		CategoryID: req.CategoryID,
		Status:     req.Status,
		IsTop:      req.IsTop,
	}

	// 2. 调用领域服务
	articleDOList, total, err := s.articleDomainSrv.ListArticleByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询文章应用服务失败",
			zap.Int("page_num", req.PageNum),
			zap.Int("page_size", req.PageSize),
			zap.Error(err),
		)
		return nil, err
	}

	// 3. DO 列表转换为响应数据
	var respList []map[string]interface{}
	for _, articleDO := range articleDOList {
		respList = append(respList, map[string]interface{}{
			"id":          articleDO.ID,
			"title":       articleDO.Title,
			"cover":       articleDO.Cover,
			"summary":     articleDO.Summary,
			"category_id": articleDO.CategoryID,
			"sort":        articleDO.Sort,
			"status":      articleDO.Status,
			"is_top":      articleDO.IsTop,
			"views":       articleDO.Views,
			"likes":       articleDO.Likes,
			"comments":    articleDO.Comments,
			"publish_at":  articleDO.PublishAt,
			"created_at":  articleDO.CreateAt,
			"updated_at":  articleDO.UpdateAt,
		})
	}

	// 4. 封装通用分页返回结果
	pages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)
	result := &dto.PageResult{
		Total:    total,
		Pages:    int(pages),
		List:     respList,
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	}

	return result, nil
}

// GetArticleById 根据ID查询文章详情
func (s *ArticleAppService) GetArticleById(ctx context.Context, id uint64) (map[string]interface{}, error) {
	// 1. 调用领域服务
	articleDO, err := s.articleDomainSrv.GetArticleById(ctx, id)
	if err != nil {
		logger.Error("根据ID查询文章应用服务失败",
			zap.Uint64("article_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	// 2. 组装返回数据（对齐分类返回格式）
	resp := map[string]interface{}{
		"id":             articleDO.ID,
		"admin_id":       articleDO.AdminID,
		"title":          articleDO.Title,
		"cover":          articleDO.Cover,
		"summary":        articleDO.Summary,
		"content":        articleDO.Content,
		"category_id":    articleDO.CategoryID,
		"tag_ids":        articleDO.TagIDs,
		"status":         articleDO.Status,
		"sort":           articleDO.Sort,
		"views":          articleDO.Views,
		"likes":          articleDO.Likes,
		"comments":       articleDO.Comments,
		"is_top":         articleDO.IsTop,
		"comment_status": articleDO.CommentStatus,
		"publish_at":     articleDO.PublishAt,
		"create_at":      articleDO.CreateAt,
		"update_at":      articleDO.UpdateAt,
	}

	return resp, nil
}

// UpdateArticle 更新文章
func (s *ArticleAppService) UpdateArticle(ctx context.Context, req *dto.UpdateArticleReq) error {
	// 1. 应用层参数校验
	if req.ID <= 0 {
		return common.ParamErr.WithMsg("文章ID不合法")
	}

	// 2. 组装 DO
	articleDO := &do.ArticleDO{
		ID:            req.ID,
		Title:         req.Title,
		Cover:         req.Cover,
		Summary:       req.Summary,
		Content:       req.Content,
		CategoryID:    req.CategoryID,
		TagIDs:        req.TagIDs,
		Status:        req.Status,
		Sort:          req.Sort,
		IsTop:         req.IsTop,
		CommentStatus: req.CommentStatus,
		PublishAt:     req.PublishAt,
	}

	// 3. 调用领域服务
	err := s.articleDomainSrv.UpdateArticle(ctx, articleDO)
	if err != nil {
		logger.Error("更新文章应用服务失败",
			zap.Uint64("article_id", req.ID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// DeleteArticle 删除文章
func (s *ArticleAppService) DeleteArticle(ctx context.Context, id uint64) error {
	// 1. 调用领域服务
	err := s.articleDomainSrv.DeleteArticle(ctx, id)
	if err != nil {
		logger.Error("删除文章应用服务失败",
			zap.Uint64("article_id", id),
			zap.Error(err),
		)
		return err
	}

	return nil
}
