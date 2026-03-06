package response

import (
	"blog-server/common"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一响应结构体
type Resp struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	ErrMsg string `json:"err_msg"`
	Data   any    `json:"data"`
}

// 统一响应写入
func WriteResp(ctx *gin.Context, data any, errno common.Errno) {
	ctx.JSON(http.StatusOK, Resp{
		Code:   errno.Code,
		Msg:    errno.Msg,
		ErrMsg: errno.ErrMsg,
		Data:   data,
	})
}
