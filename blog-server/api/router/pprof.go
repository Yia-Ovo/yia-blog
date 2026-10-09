package router

import (
	"net"
	"net/http"
	httppprof "net/http/pprof"
	runtimepprof "runtime/pprof"

	"github.com/gin-gonic/gin"
)

// SetupPprof 注册本机直接访问的诊断路由，不使用默认 HTTP mux。
func SetupPprof(r *gin.Engine, prefix string) {
	g := r.Group(prefix, localPprofOnly)
	g.GET("/", gin.WrapF(httppprof.Index))
	g.GET("/cmdline", gin.WrapF(httppprof.Cmdline))
	g.GET("/profile", gin.WrapF(httppprof.Profile))
	g.GET("/symbol", gin.WrapF(httppprof.Symbol))
	g.GET("/trace", gin.WrapF(httppprof.Trace))
	for _, profile := range runtimepprof.Profiles() {
		g.GET("/"+profile.Name(), gin.WrapH(httppprof.Handler(profile.Name())))
	}
}

// localPprofOnly 使用真实连接对端而非可伪造的代理 IP；拒绝代理和跨站浏览器请求。
func localPprofOnly(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	peer, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil || !isLoopbackIP(peer) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	host := c.Request.Host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	if host != "localhost" && !isLoopbackIP(host) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP", "Origin"} {
		if _, present := c.Request.Header[http.CanonicalHeaderKey(header)]; present {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
	if site := c.GetHeader("Sec-Fetch-Site"); site != "" && site != "none" && site != "same-origin" {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	c.Next()
}

func isLoopbackIP(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
