package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// ArticleManagementHandler 为本人和管理员读取提供独立 HTTP 入口。
type ArticleManagementHandler struct {
	app *service.ArticleManagementAppService
}

// NewArticleManagementHandler 注入管理读取应用服务。
func NewArticleManagementHandler(app *service.ArticleManagementAppService) *ArticleManagementHandler {
	return &ArticleManagementHandler{app: app}
}

// ListMyArticles 查询本人文章，包括草稿与下架内容。
func (h *ArticleManagementHandler) ListMyArticles(c *gin.Context) {
	h.list(c, consts.UserIDKey, h.app.ListMyArticles)
}

// ListAdminArticles 查询管理范围内文章。
func (h *ArticleManagementHandler) ListAdminArticles(c *gin.Context) {
	h.list(c, consts.AdminIDKey, h.app.ListAdminArticles)
}

// GetMyArticle 获取本人文章编辑数据。
func (h *ArticleManagementHandler) GetMyArticle(c *gin.Context) {
	h.detail(c, consts.UserIDKey, h.app.GetMyArticle)
}

// GetAdminArticle 获取管理员文章编辑数据。
func (h *ArticleManagementHandler) GetAdminArticle(c *gin.Context) {
	h.detail(c, consts.AdminIDKey, h.app.GetAdminArticle)
}

func (h *ArticleManagementHandler) list(c *gin.Context, identityKey string, load func(context.Context, uint64, *service.ArticleManagementQuery) (*dto.PageResult, error)) {
	actorID, ok := managementActorID(c, identityKey)
	if !ok {
		return
	}
	var req dto.ArticlePageQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	result, err := load(ctx, actorID, &service.ArticleManagementQuery{PageNum: req.PageNum, PageSize: req.PageSize, Title: req.Title, CategoryID: req.CategoryID, Status: req.Status, IsTop: req.IsTop})
	writeManagementResult(c, result, err)
}

func (h *ArticleManagementHandler) detail(c *gin.Context, identityKey string, load func(context.Context, uint64, uint64) (*dto.ManagedArticle, error)) {
	actorID, ok := managementActorID(c, identityKey)
	if !ok {
		return
	}
	var req dto.ArticleIDQueryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	result, err := load(ctx, actorID, req.ID)
	writeManagementResult(c, result, err)
}
