/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-29 17:00:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-09 09:05:36
 * @FilePath: \yia-blog\blog-server\app\service\article_app_service.go
 * @Description: 文章应用服务 - 业务编排、缓存处理、流程调度
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"

	"go.uber.org/zap"
)

// ArticleAppService 应用服务：仅做流程编排 + 缓存处理
type ArticleAppService struct {
	articleDomainService *service.ArticleDomainService // 领域服务（纯业务）
	articleRedisRepo     repo.ArticleRedisRepo         // 缓存仓储（应用层编排）
	userDomainService    *service.UserDomainService    // 关联文章作者（普通用户）信息
	adminDomainService   *service.AdminDomainService   // 关联文章作者（管理员）信息
}

// NewArticleAppService 构造函数：依赖外部注入，无强耦合
func NewArticleAppService(
	domainService *service.ArticleDomainService,
	redisRepo repo.ArticleRedisRepo,
	userDomainService *service.UserDomainService,
	adminDomainService *service.AdminDomainService,
) *ArticleAppService {
	return &ArticleAppService{
		articleDomainService: domainService,
		articleRedisRepo:     redisRepo,
		userDomainService:    userDomainService,
		adminDomainService:   adminDomainService,
	}
}

// CreateArticle 创建文章：调用领域服务
func (s *ArticleAppService) CreateArticle(ctx context.Context, userID uint64, req *dto.CreateArticleReq) error {
	if req == nil {
		logger.Warn("创建文章参数为空")
		return common.ParamErr
	}

	// DTO 转换为领域对象
	articleDO := &do.ArticleDO{
		UserID:        userID,
		AdminID:       0,
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

	// 1. 执行核心业务逻辑
	err := s.articleDomainService.CreateArticle(ctx, articleDO)
	if err != nil {
		logger.Error("创建文章失败", zap.String("title", req.Title), zap.Error(err))
		return err
	}

	logger.Info("创建文章成功", zap.String("title", req.Title))
	return nil
}

// AdminCreateArticle 创建文章（管理员）：作者 = 当前管理员（admin_id=adminID, user_id=0）
func (s *ArticleAppService) AdminCreateArticle(ctx context.Context, adminID uint64, req *dto.CreateArticleReq) error {
	if req == nil {
		logger.Warn("创建文章参数为空")
		return common.ParamErr
	}

	// DTO 转换为领域对象：作者为管理员
	articleDO := &do.ArticleDO{
		UserID:        0,
		AdminID:       adminID,
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

	err := s.articleDomainService.CreateArticle(ctx, articleDO)
	if err != nil {
		logger.Error("管理员创建文章失败", zap.String("title", req.Title), zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	logger.Info("管理员创建文章成功", zap.String("title", req.Title), zap.Uint64("admin_id", adminID))
	return nil
}

// ListArticleByPage 分页查询文章
func (s *ArticleAppService) ListArticleByPage(ctx context.Context, req *dto.ArticlePageQuery) (*dto.PageResult, error) {
	if req == nil {
		logger.Warn("文章分页查询参数为空")
		return nil, common.ParamErr
	}

	query := &do.ArticleQuery{
		PageNum:    req.PageNum,
		PageSize:   req.PageSize,
		Title:      req.Title,
		CategoryID: req.CategoryID,
		Status:     req.Status,
		IsTop:      req.IsTop,
	}

	// 调用领域服务
	articleDOList, total, err := s.articleDomainService.ListPublicArticleByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询文章失败", zap.Error(err))
		return nil, err
	}

	// 组装返回
	authorMemo := make(map[authorRef]*dto.AuthorInfo) // 同页去重，避免同一作者多篇文章重复查询
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
			"author":      s.resolveAuthorMemo(ctx, articleDO.UserID, articleDO.AdminID, authorMemo),
		})
	}

	pages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)
	return &dto.PageResult{
		Total:    total,
		Pages:    int(pages),
		List:     respList,
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	}, nil
}

// GetArticleById 根据ID查询文章详情
func (s *ArticleAppService) GetArticleById(ctx context.Context, id uint64) (map[string]interface{}, error) {
	if id <= 0 {
		logger.Warn("文章ID非法")
		return nil, common.ParamErr
	}

	articleDO, err := s.articleDomainService.GetPublicArticleByID(ctx, id)
	if err != nil {
		logger.Error("查询文章详情失败", zap.Uint64("id", id), zap.Error(err))
		return nil, err
	}

	return map[string]interface{}{
		"id":             articleDO.ID,
		"user_id":        articleDO.UserID,
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
		"author":         s.resolveAuthor(ctx, articleDO.UserID, articleDO.AdminID),
	}, nil
}

// UpdateArticle 更新文章（普通用户）：仅能更新本人文章
func (s *ArticleAppService) UpdateArticle(ctx context.Context, userID uint64, req *dto.UpdateArticleReq) error {
	if req.ID <= 0 {
		logger.Warn("文章ID非法")
		return common.ParamErr
	}

	articleDO := buildArticleUpdateDO(req)

	err := s.articleDomainService.UpdateArticleByUser(ctx, userID, articleDO)
	if err != nil {
		logger.Error("更新文章失败", zap.Uint64("id", req.ID), zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}

	logger.Info("更新文章成功", zap.Uint64("id", req.ID), zap.Uint64("user_id", userID))
	return nil
}

// AdminUpdateArticle 更新文章（管理员）：可更新任意文章
func (s *ArticleAppService) AdminUpdateArticle(ctx context.Context, adminID uint64, req *dto.UpdateArticleReq) error {
	if req.ID <= 0 {
		logger.Warn("文章ID非法")
		return common.ParamErr
	}

	articleDO := buildArticleUpdateDO(req)

	err := s.articleDomainService.UpdateArticle(ctx, articleDO)
	if err != nil {
		logger.Error("管理员更新文章失败", zap.Uint64("id", req.ID), zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	logger.Info("管理员更新文章成功", zap.Uint64("id", req.ID), zap.Uint64("admin_id", adminID))
	return nil
}

// DeleteArticle 删除文章（普通用户）：仅能删除本人文章
func (s *ArticleAppService) DeleteArticle(ctx context.Context, userID uint64, id uint64) error {
	if id <= 0 {
		logger.Warn("文章ID非法")
		return common.ParamErr
	}

	err := s.articleDomainService.DeleteArticleByUser(ctx, userID, id)
	if err != nil {
		logger.Error("删除文章失败", zap.Uint64("id", id), zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}

	logger.Info("删除文章成功", zap.Uint64("id", id), zap.Uint64("user_id", userID))
	return nil
}

// AdminDeleteArticle 删除文章（管理员）：可删除任意文章
func (s *ArticleAppService) AdminDeleteArticle(ctx context.Context, adminID uint64, id uint64) error {
	if id <= 0 {
		logger.Warn("文章ID非法")
		return common.ParamErr
	}

	err := s.articleDomainService.DeleteArticle(ctx, id)
	if err != nil {
		logger.Error("管理员删除文章失败", zap.Uint64("id", id), zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	logger.Info("管理员删除文章成功", zap.Uint64("id", id), zap.Uint64("admin_id", adminID))
	return nil
}

// buildArticleUpdateDO 从更新请求构建领域对象（不含作者字段，作者由归属校验/既有行决定）
func buildArticleUpdateDO(req *dto.UpdateArticleReq) *do.ArticleDO {
	return &do.ArticleDO{
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
}

// authorRef 作者引用（user_id/admin_id 二选一），用作同页作者去重的 map key
type authorRef struct {
	userID  uint64
	adminID uint64
}

// resolveAuthor 按作者引用（user_id/admin_id 二选一，非零者为作者）实时关联作者信息。
// 作者不存在或查询失败时返回 nil（响应中为 null），不影响文章本身的读取。
func (s *ArticleAppService) resolveAuthor(ctx context.Context, userID, adminID uint64) *dto.AuthorInfo {
	switch {
	case userID != 0:
		userDO, err := s.userDomainService.FindUserByID(ctx, userID)
		if err != nil || userDO == nil {
			logger.Warn("加载文章作者信息失败（用户）", zap.Uint64("user_id", userID), zap.Error(err))
			return nil
		}
		return &dto.AuthorInfo{
			Type:      consts.AuthorTypeUser,
			Name:      userDO.NickName,
			Avatar:    userDO.Avatar,
			Signature: userDO.Signature,
		}
	case adminID != 0:
		adminDO, err := s.adminDomainService.FindAdminByID(ctx, adminID)
		if err != nil || adminDO == nil {
			logger.Warn("加载文章作者信息失败（管理员）", zap.Uint64("admin_id", adminID), zap.Error(err))
			return nil
		}
		return &dto.AuthorInfo{
			Type:      consts.AuthorTypeAdmin,
			Name:      adminDO.AdminName,
			Avatar:    adminDO.Avatar,
			Signature: adminDO.Remark,
		}
	default:
		return nil
	}
}

// resolveAuthorMemo 带 memo 的作者关联：同一页内相同作者只查询一次
func (s *ArticleAppService) resolveAuthorMemo(ctx context.Context, userID, adminID uint64, memo map[authorRef]*dto.AuthorInfo) *dto.AuthorInfo {
	ref := authorRef{userID: userID, adminID: adminID}
	if author, ok := memo[ref]; ok {
		return author
	}
	author := s.resolveAuthor(ctx, userID, adminID)
	memo[ref] = author
	return author
}
