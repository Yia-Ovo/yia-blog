package repo

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
)

type CommentMysqlRepo interface {
	// CreateComment 创建评论
	CreateComment(ctx context.Context, commentModel *model.CommentModel) error
	// ListCommentByPage 分页查询评论列表 + 返回总数
	ListCommentByPage(ctx context.Context, query *do.CommentQuery) (list []*model.CommentModel, total int64, err error)
	// UpdateComment 更新评论
	UpdateComment(ctx context.Context, commentModel *model.CommentModel) error
	// DeleteComment 删除评论(软删除)
	DeleteComment(ctx context.Context, commentID uint64) error
	// LikeComment 评论点赞(数据库+1)
	LikeComment(ctx context.Context, commentID uint64) error
	// DisLikeComment 评论取消点赞(数据库-1)
	DisLikeComment(ctx context.Context, commentID uint64) error
	// GetCommentByID 根据ID查询评论(基础方法，建议加上)
	GetCommentByID(ctx context.Context, commentID uint64) (*model.CommentModel, error)
}

// CommentRedisRepo 评论Redis仓储接口
type CommentRedisRepo interface {
	// --------------------- 评论点赞相关 ---------------------
	// GetCommentLikeCount 获取评论点赞数
	GetCommentLikeCount(ctx context.Context, commentID uint64) (int, error)
	// SetCommentLikeCount 设置评论点赞数
	SetCommentLikeCount(ctx context.Context, commentID uint64, count int) error
	// IncrCommentLikeCount 评论点赞数+1
	IncrCommentLikeCount(ctx context.Context, commentID uint64) error
	// DecrCommentLikeCount 评论点赞数-1
	DecrCommentLikeCount(ctx context.Context, commentID uint64) error

	// --------------------- 用户点赞去重 ---------------------
	// CheckUserLikeComment 检查用户是否已点赞该评论
	CheckUserLikeComment(ctx context.Context, userID uint64, commentID uint64) (bool, error)
	// AddUserLikeRecord 添加用户点赞记录
	AddUserLikeRecord(ctx context.Context, userID uint64, commentID uint64) error
	// DelUserLikeRecord 删除用户点赞记录
	DelUserLikeRecord(ctx context.Context, userID uint64, commentID uint64) error

	// --------------------- 评论列表/统计缓存 ---------------------
	// SetArticleCommentCount 设置文章的评论总数缓存
	SetArticleCommentCount(ctx context.Context, articleID uint64, count int) error
	// GetArticleCommentCount 获取文章的评论总数缓存
	GetArticleCommentCount(ctx context.Context, articleID uint64) (int, error)
	// DelArticleCommentCache 删除文章评论相关缓存(更新/删除评论时清理)
	DelArticleCommentCache(ctx context.Context, articleID uint64) error
}
