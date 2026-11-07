package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *AIHandler) Workspace(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	data, err := h.app.Workspace(ctx, id)
	aiWrite(c, data, err)
}

func (h *AIHandler) CreateCodeJob(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	var req dto.AICodeJobReq
	if c.ShouldBindJSON(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	data, err := h.app.CreateCodeJob(ctx, service.CreateCodeJobCommand{AdminID: id, ID: req.ID, Prompt: req.Prompt, ContextMode: req.ContextMode})
	aiWrite(c, data, err)
}

func (h *AIHandler) PublishCodeJob(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	var req dto.AIPublishJobReq
	if c.ShouldBindJSON(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	aiWrite(c, nil, h.app.PublishCodeJob(ctx, id, req.ID, req.ReviewDigest))
}

// aiWorkerTransport 检查直连回环地址，不信任代理提供的 ClientIP；工作凭据只交给 App 校验。
func aiWorkerTransport(c *gin.Context) bool {
	if c.GetHeader("X-Forwarded-For") != "" || c.GetHeader("X-Real-IP") != "" || c.GetHeader("Forwarded") != "" {
		return false
	}
	if c.GetHeader("X-Forwarded-For") != "" || c.GetHeader("X-Real-IP") != "" || c.GetHeader("Forwarded") != "" {
		return false
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	ip := net.ParseIP(host)
	return err == nil && ip != nil && ip.IsLoopback()
}

func (h *AIHandler) ClaimCodeJob(c *gin.Context) {
	ctx, cancel, _ := aiRequest(c)
	defer cancel()
	if !aiWorkerTransport(c) {
		aiWrite(c, nil, common.PermissionErr)
		return
	}
	data, err := h.app.ClaimCodeJob(ctx, strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	aiWrite(c, data, err)
}

func (h *AIHandler) ReportCodeJob(c *gin.Context) {
	ctx, cancel, _ := aiSizedRequest(c, 1<<20)
	defer cancel()
	if !aiWorkerTransport(c) {
		aiWrite(c, nil, common.PermissionErr)
		return
	}
	var req dto.AIWorkerReportReq
	if c.ShouldBindJSON(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	aiWrite(c, nil, h.app.ReportCodeJob(ctx, strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "), service.CodeJobReportCommand{AdminID: req.AdminID, ID: req.ID, Lease: req.Lease, Phase: req.Phase, Summary: req.Summary, Diff: req.Diff, Checks: req.Checks, BaseRevision: req.BaseRevision, TestsPassed: req.TestsPassed}))
}
