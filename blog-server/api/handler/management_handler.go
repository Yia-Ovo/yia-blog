package handler

import (
	"blog-server/common"
	"blog-server/common/gin/response"
	"errors"

	"github.com/gin-gonic/gin"
)

// managementActorID 只读取已由鉴权中间件建立的 HTTP 身份，不判断角色或资源归属。
func managementActorID(c *gin.Context, key string) (uint64, bool) {
	value, exists := c.Get(key)
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return 0, false
	}
	id, ok := value.(uint64)
	if !ok || id == 0 {
		response.WriteResp(c, nil, common.AuthErr)
		return 0, false
	}
	return id, true
}

func writeManagementResult(c *gin.Context, data any, err error) {
	if err == nil {
		response.WriteResp(c, data, common.OK)
		return
	}
	var errno common.Errno
	if !errors.As(err, &errno) {
		errno = common.ServerErr
	}
	response.WriteResp(c, nil, errno)
}
