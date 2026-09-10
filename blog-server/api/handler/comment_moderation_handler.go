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

// CommentModerationHandler 仅负责管理员审核请求的 HTTP 绑定和响应。
type CommentModerationHandler struct {
	app *service.CommentModerationAppService
}

// NewCommentModerationHandler 注入审核应用服务。
func NewCommentModerationHandler(app *service.CommentModerationAppService) *CommentModerationHandler {
	return &CommentModerationHandler{app: app}
}

// ListAdminComments 获取管理员评论列表。
func (h *CommentModerationHandler) ListAdminComments(c *gin.Context) {
	adminID, ok := managementActorID(c, consts.AdminIDKey)
	if !ok {
		return
	}
	var req dto.CommentModerationPageQueryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	result, err := h.app.ListAdminComments(ctx, adminID, &service.CommentModerationQuery{ArticleID: req.ArticleID, ParentID: req.ParentID, Nickname: req.Nickname, Content: req.Content, Status: req.Status, Page: req.Page, Size: req.Size, OrderBy: req.OrderBy, OrderDir: req.OrderDir})
	writeManagementResult(c, result, err)
}

// ReviewComment 提交通过/拒绝决定及管理员实际查看的内容快照。
func (h *CommentModerationHandler) ReviewComment(c *gin.Context) {
	adminID, ok := managementActorID(c, consts.AdminIDKey)
	if !ok {
		return
	}
	var req dto.CommentReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	err := h.app.ReviewComment(ctx, adminID, &service.CommentReviewCommand{ID: req.ID, ExpectedContent: *req.ExpectedContent, ExpectedStatus: *req.ExpectedStatus, Status: *req.Status})
	writeManagementResult(c, nil, err)
}
