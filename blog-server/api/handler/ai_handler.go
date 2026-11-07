package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"blog-server/domain/do"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AIHandler struct{ app *service.AIAppService }

func NewAIHandler(app *service.AIAppService) *AIHandler { return &AIHandler{app: app} }

func aiRequest(c *gin.Context) (context.Context, context.CancelFunc, uint64) {
	return aiSizedRequest(c, 32<<10)
}

func aiSizedRequest(c *gin.Context, maxBytes int64) (context.Context, context.CancelFunc, uint64) {
	c.Header("Cache-Control", "private, no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	ctx, cancel := context.WithTimeout(c.Request.Context(), consts.AIRequestTimeout)
	id, _ := c.Get(consts.AdminIDKey)
	adminID, _ := id.(uint64)
	return ctx, cancel, adminID
}
func aiWrite(c *gin.Context, data any, err error) {
	if err == nil {
		response.WriteResp(c, data, common.OK)
		return
	}
	errno, ok := err.(common.Errno)
	if !ok {
		errno = common.ServerErr
	}
	response.WriteResp(c, nil, errno)
}
func (h *AIHandler) Overview(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	data, err := h.app.Overview(ctx, id)
	aiWrite(c, data, err)
}
func (h *AIHandler) Conversation(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	var req dto.AIConversationReq
	if c.ShouldBindQuery(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	data, err := h.app.Conversation(ctx, id, req.ID)
	aiWrite(c, data, err)
}
func (h *AIHandler) Send(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	var req dto.AISendReq
	if c.ShouldBindJSON(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	data, err := h.app.Send(ctx, service.AISendCommand{AdminID: id, ConversationID: req.ConversationID, RequestID: req.RequestID, Content: req.Content, ContextMode: req.ContextMode, ImportID: req.ImportID})
	aiWrite(c, data, err)
}
func (h *AIHandler) SetMemory(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	var req dto.AIMemoryReq
	if c.ShouldBindJSON(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	aiWrite(c, nil, h.app.SetMemory(ctx, id, req.Memory, req.ContextMode))
}
func (h *AIHandler) DeleteConversation(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	var req dto.AIConversationReq
	if c.ShouldBindJSON(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	aiWrite(c, nil, h.app.DeleteConversation(ctx, id, req.ID))
}

func (h *AIHandler) ImportedThreads(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	data, err := h.app.ImportedThreads(ctx, id)
	aiWrite(c, data, err)
}

func (h *AIHandler) ImportedThread(c *gin.Context) {
	ctx, cancel, id := aiRequest(c)
	defer cancel()
	var req dto.AIImportReq
	if c.ShouldBindQuery(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	data, err := h.app.ImportedThread(ctx, id, req.ID, req.Before)
	aiWrite(c, data, err)
}

func (h *AIHandler) SyncImportedThread(c *gin.Context) {
	ctx, cancel, id := aiSizedRequest(c, consts.AIImportMaxBytes)
	defer cancel()
	var req dto.AISyncImportReq
	if c.ShouldBindJSON(&req) != nil {
		aiWrite(c, nil, common.ParamErr)
		return
	}
	value := do.AIImportedThread{AdminID: id, ID: req.ID, Title: req.Title, Workspace: req.Workspace, UpdatedAt: req.UpdatedAt}
	for _, m := range req.Messages {
		value.Messages = append(value.Messages, do.AIMessage{ID: m.ID, Role: m.Role, Content: m.Content, CreateAt: m.CreateAt})
	}
	aiWrite(c, nil, h.app.SyncImportedThread(ctx, value))
}
