package handler

import (
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AppearanceHandler 提供每日外观元数据与受控图片资源。
type AppearanceHandler struct{ app *service.AppearanceAppService }

func NewAppearanceHandler(app *service.AppearanceAppService) *AppearanceHandler {
	return &AppearanceHandler{app: app}
}

func (h *AppearanceHandler) Today(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), consts.AppearanceRequestTimeout)
	defer cancel()
	c.Header("Cache-Control", "no-store")
	value, err := h.app.Today(ctx)
	if err != nil {
		response.WriteResp(c, nil, common.ServerErr)
		return
	}
	response.WriteResp(c, value, common.OK)
}

// Image 成功返回 JPEG 二进制；错误仍使用 API 统一信封。
func (h *AppearanceHandler) Image(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), consts.AppearanceRequestTimeout)
	defer cancel()
	data, err := h.app.Image(ctx, c.Param("id"))
	if err != nil {
		c.Header("Cache-Control", "no-store")
		errno, ok := err.(common.Errno)
		if !ok {
			errno = common.ServerErr
		}
		response.WriteResp(c, nil, errno)
		return
	}
	etag := `"` + c.Param("id") + `"`
	c.Header("Cache-Control", consts.AppearanceImageCacheControl)
	c.Header("ETag", etag)
	c.Header("X-Content-Type-Options", "nosniff")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "image/jpeg", data)
}
