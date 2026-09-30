package handler

import (
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AvatarHandler 接收单个 multipart file，身份只来自鉴权 Context。
type AvatarHandler struct{ app *service.AvatarAppService }

func NewAvatarHandler(app *service.AvatarAppService) *AvatarHandler { return &AvatarHandler{app: app} }

func avatarHTTPError(c *gin.Context, err error) {
	errno := common.ServerErr
	var supplied common.Errno
	if errors.As(err, &supplied) {
		errno = supplied
	}
	response.WriteResp(c, nil, errno)
}

func readAvatar(c *gin.Context) ([]byte, error) {
	invalid := common.ParamErr
	invalid.Msg = "请选择单张不超过 5 MB 的 JPG 或 PNG 图片"
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, consts.AvatarMaxUploadBytes+consts.AvatarMultipartOverhead)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return nil, invalid
	}
	part, err := reader.NextPart()
	if err != nil {
		return nil, invalid
	}
	defer part.Close()
	if part.FormName() != "file" || part.FileName() == "" {
		return nil, invalid
	}
	data, err := io.ReadAll(io.LimitReader(part, consts.AvatarMaxUploadBytes+1))
	if err != nil || len(data) == 0 || len(data) > consts.AvatarMaxUploadBytes {
		return nil, invalid
	}
	if _, err := reader.NextPart(); err != io.EOF {
		return nil, invalid
	}
	return data, nil
}

func (h *AvatarHandler) UploadAdmin(c *gin.Context) { h.upload(c, consts.AdminIDKey, true) }
func (h *AvatarHandler) UploadUser(c *gin.Context)  { h.upload(c, consts.UserIDKey, false) }

func (h *AvatarHandler) upload(c *gin.Context, identityKey string, admin bool) {
	value, ok := c.Get(identityKey)
	id, typed := value.(uint64)
	if !ok || !typed || id == 0 {
		avatarHTTPError(c, common.LoginNotLoginErr)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), consts.AvatarRequestTimeout)
	defer cancel()
	data, err := readAvatar(c)
	if err != nil {
		avatarHTTPError(c, err)
		return
	}
	if admin {
		result, err := h.app.UploadAdmin(ctx, id, data)
		if err != nil {
			avatarHTTPError(c, err)
			return
		}
		response.WriteResp(c, result, common.OK)
		return
	}
	result, err := h.app.UploadUser(ctx, id, data)
	if err != nil {
		avatarHTTPError(c, err)
		return
	}
	response.WriteResp(c, result, common.OK)
}

func (h *AvatarHandler) Image(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), consts.AvatarRequestTimeout)
	defer cancel()
	data, err := h.app.Image(ctx, c.Param("name"))
	if err != nil {
		c.Header("Cache-Control", "no-store")
		avatarHTTPError(c, err)
		return
	}
	etag := `"` + c.Param("name") + `"`
	c.Header("Cache-Control", consts.AvatarImageCacheControl)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("ETag", etag)
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "image/png", data)
}
