package router

import (
	"blog-server/api/handler"
	"blog-server/middleware"
	"github.com/gin-gonic/gin"
)

func (r *Router) aiRouter(v1 *gin.RouterGroup) {
	// 测试容器和禁用 AI 的旧部署仍可注册其他业务路由。
	if r.c.AIApp == nil {
		return
	}
	g := v1.Group("/admin/ai", middleware.DefaultAdminAuth())
	h := handler.NewAIHandler(r.c.AIApp)
	g.GET("/overview", h.Overview)
	g.GET("/conversation", h.Conversation)
	// HTTP 入口先检查超管，Domain 仍重新核验现行账号和资料归属。
	private := g.Group("", middleware.AdminSuperAuth())
	private.GET("/imports", h.ImportedThreads)
	private.GET("/import", h.ImportedThread)
	private.POST("/import", h.SyncImportedThread)
	private.GET("/workspace", h.Workspace)
	private.POST("/job", h.CreateCodeJob)
	private.POST("/job/publish", h.PublishCodeJob)
	g.POST("/send", h.Send)
	g.POST("/memory", h.SetMemory)
	g.POST("/delete", h.DeleteConversation)
	worker := v1.Group("/internal/ai")
	worker.POST("/claim", h.ClaimCodeJob)
	worker.POST("/report", h.ReportCodeJob)
}
