/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-25 14:32:37
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-16 19:38:37
 * @FilePath: \yia-blog\blog-server\app\service\comment_app_service.go
 * @Description: 评论应用服务 - 业务编排、缓存处理、流程调度
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package service

import (
	"blog-server/common"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"

	"go.uber.org/zap"
)

// CommentCreateCommand 发布评论用例的应用层输入。
// HTTP DTO 和 Gin Context 由 Handler 转换为该命令，AppService 不依赖 API 层类型。
type CommentCreateCommand struct {
	UserID    uint64 // 评论作者的登录用户ID，来自鉴权 Context
	ArticleID uint64 // 评论所属文章ID
	ParentID  uint64 // 0 表示顶级评论，非 0 表示回复对应评论
	Nickname  string // 作者昵称快照，来自登录用户资料
	Email     string // 作者邮箱快照，来自登录用户资料
	Avatar    string // 作者头像快照，来自登录用户资料
	Content   string // 评论正文
	IP        string // 客户端IP，由服务端从 HTTP 请求提取
	UserAgent string // 客户端 User-Agent，由服务端从 HTTP 请求提取
}

// CommentListQuery 评论分页查询用例的应用层输入。
// Handler 把 HTTP 查询 DTO 转换为该命令，AppService 不依赖 api/dto。
type CommentListQuery struct {
	ArticleID uint64  // 大于 0 时按所属文章筛选
	ParentID  *uint64 // nil 不筛选；指向 0 查顶级，指向 N 查某父评论的回复
	Status    *int    // nil 不筛选；0 待审核 / 1 已通过 / 2 已拒绝
	AdminID   uint64  // 大于 0 时按管理员作者筛选
	Nickname  string  // 昵称模糊搜索
	Content   string  // 内容模糊搜索
	Page      int     // 页码，默认 1
	Size      int     // 每页数量，默认 10
	OrderBy   string  // 排序字段，由 Repo 白名单校验
	OrderDir  string  // 排序方向 asc/desc
}

// CommentAppService 应用服务：仅做流程编排 + 缓存处理
type CommentAppService struct {
	commentDomainService *service.CommentDomainService // 领域服务（纯业务）
	commentRedisRepo     repo.CommentRedisRepo         // 缓存仓储（应用层编排）
}

// NewCommentAppService 构造函数：依赖外部注入，无强耦合
func NewCommentAppService(
	domainService *service.CommentDomainService,
	redisRepo repo.CommentRedisRepo,
) *CommentAppService {
	return &CommentAppService{
		commentDomainService: domainService,
		commentRedisRepo:     redisRepo,
	}
}

// CreateComment 发布评论：调用领域服务 + 清理缓存
func (c *CommentAppService) CreateComment(ctx context.Context, command *CommentCreateCommand) error {
	if command == nil {
		logger.Warn("[APP] 发布评论参数为空")
		return common.ParamErr
	}
	if command.UserID == 0 {
		logger.Warn("[APP] 发布评论未获取到登录用户")
		return common.LoginNotLoginErr
	}

	commentDO := &do.CommentDO{
		UserID:    command.UserID,
		ArticleID: command.ArticleID,
		ParentID:  command.ParentID,
		Nickname:  command.Nickname,
		Email:     command.Email,
		Avatar:    command.Avatar,
		Content:   command.Content,
		IP:        command.IP,
		UserAgent: command.UserAgent,
	}

	if err := c.commentDomainService.CreateComment(ctx, commentDO); err != nil {
		return err
	}

	_ = c.commentRedisRepo.DelArticleCommentCache(ctx, command.ArticleID)

	logger.Info("[APP] 创建评论，缓存已清理",
		zap.Uint64("user_id", command.UserID),
		zap.Uint64("article_id", command.ArticleID),
	)
	return nil
}

// ListCommentByPage 分页查询评论：把应用层查询命令转换为领域查询条件后调用领域服务
func (c *CommentAppService) ListCommentByPage(ctx context.Context, q *CommentListQuery) ([]do.CommentDO, int64, error) {
	if q == nil {
		logger.Warn("评论分页查询参数为空")
		return nil, 0, common.ParamErr
	}
	query := &do.CommentQuery{
		ArticleID: q.ArticleID,
		ParentID:  q.ParentID,
		Status:    q.Status,
		AdminID:   q.AdminID,
		Nickname:  q.Nickname,
		Content:   q.Content,
		Page:      q.Page,
		Size:      q.Size,
		OrderBy:   q.OrderBy,
		OrderDir:  q.OrderDir,
	}
	return c.commentDomainService.ListCommentByPage(ctx, query)
}

// GetCommentByID 根据ID获取评论详情
func (c *CommentAppService) GetCommentByID(ctx context.Context, commentID uint64) (*do.CommentDO, error) {
	return c.commentDomainService.GetCommentByID(ctx, commentID)
}

// UpdateComment 编辑评论：调用领域服务 + 清理缓存
func (c *CommentAppService) UpdateComment(ctx context.Context, operatorID uint64, commentID uint64, content string) error {
	articleID, err := c.commentDomainService.UpdateComment(
		ctx,
		operatorID,
		commentID,
		content,
	)
	if err != nil {
		return err
	}

	_ = c.commentRedisRepo.DelArticleCommentCache(ctx, articleID)

	logger.Info("[APP] 更新评论，缓存已清理",
		zap.Uint64("article_id", articleID),
		zap.Uint64("comment_id", commentID),
	)
	return nil
}

// DeleteComment 删除评论：调用领域服务 + 清理缓存
func (c *CommentAppService) DeleteComment(
	ctx context.Context,
	operatorID uint64,
	commentID uint64,
) error {
	articleID, err := c.commentDomainService.DeleteComment(
		ctx,
		operatorID,
		commentID,
	)
	if err != nil {
		return err
	}

	_ = c.commentRedisRepo.DelArticleCommentCache(ctx, articleID)
	logger.Info("[APP] 删除评论，缓存已清理",
		zap.Uint64("article_id", articleID),
		zap.Uint64("comment_id", commentID),
	)
	return nil
}

// LikeComment 评论点赞：应用层处理Redis防重 + 调用领域服务DB操作
func (c *CommentAppService) LikeComment(ctx context.Context, userID uint64, commentID uint64) error {
	// 参数校验
	if userID <= 0 || commentID <= 0 {
		logger.Warn("评论点赞参数非法", zap.Uint64("user_id", userID), zap.Uint64("comment_id", commentID))
		return common.ParamErr
	}

	// 1. 应用层编排：Redis校验重复点赞（防重逻辑）
	exist, err := c.commentRedisRepo.CheckUserLikeComment(ctx, userID, commentID)
	if err != nil {
		logger.Warn("检查点赞记录失败，降级执行", zap.Error(err))
	}
	if exist {
		return common.ErrCommentAlreadyLiked
	}

	// 2. 调用领域服务：执行数据库点赞（纯业务）
	err = c.commentDomainService.LikeComment(ctx, commentID)
	if err != nil {
		return err
	}

	// 3. 应用层编排：更新Redis缓存
	_ = c.commentRedisRepo.IncrCommentLikeCount(ctx, commentID)
	_ = c.commentRedisRepo.AddUserLikeRecord(ctx, userID, commentID)
	logger.Info("评论点赞成功", zap.Uint64("user_id", userID), zap.Uint64("comment_id", commentID))
	return nil
}

// DisLikeComment 取消点赞：应用层处理Redis + 调用领域服务DB操作
func (c *CommentAppService) DisLikeComment(ctx context.Context, userID uint64, commentID uint64) error {
	// 参数校验
	if userID <= 0 || commentID <= 0 {
		logger.Warn("取消点赞参数非法", zap.Uint64("user_id", userID), zap.Uint64("comment_id", commentID))
		return common.ParamErr
	}

	// 1. 应用层编排：Redis校验是否已点赞
	exist, err := c.commentRedisRepo.CheckUserLikeComment(ctx, userID, commentID)
	if err != nil {
		logger.Warn("检查点赞记录失败，降级执行", zap.Error(err))
	}
	if !exist {
		return common.ErrCommentNotLiked
	}

	// 2. 调用领域服务：执行数据库取消点赞
	err = c.commentDomainService.DisLikeComment(ctx, commentID)
	if err != nil {
		return err
	}

	// 3. 应用层编排：更新Redis缓存
	_ = c.commentRedisRepo.DecrCommentLikeCount(ctx, commentID)
	_ = c.commentRedisRepo.DelUserLikeRecord(ctx, userID, commentID)
	logger.Info("取消点赞成功", zap.Uint64("user_id", userID), zap.Uint64("comment_id", commentID))
	return nil
}

// GetArticleCommentCount 获取文章评论总数（Redis缓存优先，防击穿）
func (c *CommentAppService) GetArticleCommentCount(ctx context.Context, articleID uint64) (int, error) {
	if articleID <= 0 {
		logger.Warn("获取文章评论数失败：文章ID非法", zap.Uint64("article_id", articleID))
		return 0, common.ParamErr
	}

	// 1. 优先读缓存
	count, err := c.commentRedisRepo.GetArticleCommentCount(ctx, articleID)
	if err == nil {
		return count, nil
	}

	// 2. 缓存未命中，查询数据库
	logger.Info("文章评论数缓存未命中，查询数据库", zap.Uint64("article_id", articleID))
	query := &do.CommentQuery{ArticleID: articleID, Page: 1, Size: 1}
	_, total, err := c.commentDomainService.ListCommentByPage(ctx, query)
	if err != nil {
		logger.Error("查询文章评论总数失败", zap.Uint64("article_id", articleID), zap.Error(err))
		return 0, err
	}

	// 3. 回写缓存
	_ = c.commentRedisRepo.SetArticleCommentCount(ctx, articleID, int(total))
	return int(total), nil
}
