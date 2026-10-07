package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"context"
	"net"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
)

// WeatherHandler 提供访问者城市天气，不改变其他 API 的全局代理设置。
type WeatherHandler struct{ app *service.WeatherAppService }

func NewWeatherHandler(app *service.WeatherAppService) *WeatherHandler {
	return &WeatherHandler{app: app}
}

func (h *WeatherHandler) Current(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	ctx, cancel := context.WithTimeout(c.Request.Context(), consts.WeatherRequestTimeout)
	defer cancel()
	var request dto.WeatherCurrentReq
	if err := c.ShouldBindQuery(&request); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	value, err := h.app.Current(ctx, service.WeatherCurrentCommand{City: request.City, ClientIP: weatherClientIP(c.Request.RemoteAddr, c.GetHeader("X-Forwarded-For"))})
	if err != nil {
		errno, ok := err.(common.Errno)
		if !ok {
			errno = common.ServerErr
		}
		response.WriteResp(c, nil, errno)
		return
	}
	response.WriteResp(c, value, common.OK)
}

// weatherClientIP 只信任本机反向代理。部署 Nginx 必须覆盖 XFF（当前模板为 $remote_addr）。
// 直接连接的公网客户端不能用 XFF 冒充位置；多跳从右往左遇到非回环即停止。
func weatherClientIP(remoteAddress, forwarded string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return ""
	}
	peer, err := netip.ParseAddr(host)
	if err != nil || peer.Zone() != "" {
		return ""
	}
	peer = peer.Unmap()
	if !peer.IsLoopback() || forwarded == "" {
		return peer.String()
	}
	if len(forwarded) > 1024 {
		return ""
	}
	parts := strings.Split(forwarded, ",")
	if len(parts) > 16 {
		return ""
	}
	for index := len(parts) - 1; index >= 0; index-- {
		address, err := netip.ParseAddr(strings.TrimSpace(parts[index]))
		if err != nil || address.Zone() != "" {
			return ""
		}
		address = address.Unmap()
		if !address.IsLoopback() {
			return address.String()
		}
	}
	return peer.String()
}
