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
	"strings"

	"github.com/gin-gonic/gin"
)

// ArticleImageHandler 只接受单个 multipart file，当前身份来自鉴权 Context。
type ArticleImageHandler struct {
	app *service.ArticleImageAppService
}

func NewArticleImageHandler(app *service.ArticleImageAppService) *ArticleImageHandler {
	return &ArticleImageHandler{app: app}
}

func articleImageHTTPError(c *gin.Context, err error) {
	errno := common.ServerErr
	var supplied common.Errno
	if errors.As(err, &supplied) {
		errno = supplied
	}
	response.WriteResp(c, nil, errno)
}

func readArticleImage(c *gin.Context) ([]byte, error) {
	invalid := common.ParamErr
	invalid.Msg = "请选择单张不超过 5 MB 的 JPG 或 PNG 图片"
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, consts.ArticleImageMaxUploadBytes+consts.ArticleImageMultipartOverhead)
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
	data, err := io.ReadAll(io.LimitReader(part, consts.ArticleImageMaxUploadBytes+1))
	if err != nil || len(data) == 0 || len(data) > consts.ArticleImageMaxUploadBytes {
		return nil, invalid
	}
	if _, err := reader.NextPart(); err != io.EOF {
		return nil, invalid
	}
	return data, nil
}

func (h *ArticleImageHandler) UploadAdmin(c *gin.Context) { h.upload(c, consts.AdminIDKey, true) }
func (h *ArticleImageHandler) UploadUser(c *gin.Context)  { h.upload(c, consts.UserIDKey, false) }

func (h *ArticleImageHandler) upload(c *gin.Context, identityKey string, admin bool) {
	value, ok := c.Get(identityKey)
	id, typed := value.(uint64)
	if !ok || !typed || id == 0 {
		articleImageHTTPError(c, common.LoginNotLoginErr)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), consts.ArticleImageRequestTimeout)
	defer cancel()
	data, err := readArticleImage(c)
	if err != nil {
		articleImageHTTPError(c, err)
		return
	}
	if admin {
		result, err := h.app.UploadAdmin(ctx, id, data)
		if err != nil {
			articleImageHTTPError(c, err)
			return
		}
		response.WriteResp(c, result, common.OK)
		return
	}
	result, err := h.app.UploadUser(ctx, id, data)
	if err != nil {
		articleImageHTTPError(c, err)
		return
	}
	response.WriteResp(c, result, common.OK)
}

func (h *ArticleImageHandler) Image(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), consts.ArticleImageRequestTimeout)
	defer cancel()
	image, err := h.app.Image(ctx, c.Param("name"))
	if err != nil {
		c.Header("Cache-Control", "no-store")
		articleImageHTTPError(c, err)
		return
	}
	etag := `"` + c.Param("name") + `"`
	c.Header("Cache-Control", consts.ArticleImageCacheControl)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("ETag", etag)
	c.Header("Content-Type", image.ContentType)
	// GET 的弱比较同时支持浏览器/代理传入的多个 ETag。
	for _, candidate := range strings.Split(c.GetHeader("If-None-Match"), ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == etag {
			c.Status(http.StatusNotModified)
			return
		}
	}
	c.Data(http.StatusOK, image.ContentType, image.Data)
}
