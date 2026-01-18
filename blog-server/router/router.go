package router

import (
	"blog-server/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

type IRouter interface {
	Register(engine *gin.Engine)
}

type Router struct {
	FullPPROF bool
	rootPath  string
	conf      *config.Config
	checkFunc func() error // 健康检查函数
}

func NewRouter(conf *config.Config, checkFunc func() error) *Router {
	return &Router{
		FullPPROF: conf.Server.EnablePprof, // 是否开启pprof
		rootPath:  "/api/blog",
		conf:      conf,
		checkFunc: checkFunc,
	}
}

// 健康检查处理函数,用于实现服务器的健康检查端点。客户端可以请求这个端点来检查服务器是否正常运行。
func (r *Router) checkServer() func(*gin.Context) {
	return func(ctx *gin.Context) {
		if err := r.checkFunc(); err != nil {
			ctx.JSON(500, gin.H{
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (r *Router) Register(engine *gin.Engine) {
	if r.conf.Server.EnablePprof {
		SetupPprof(engine, "/debug/pprof")
	}
	engine.Any("/ping", r.checkServer()) // 健康检查

	root := engine.Group(r.rootPath)
	// 注册路由
	r.route(root)
}

func (r *Router) route(rg *gin.RouterGroup) {

}
