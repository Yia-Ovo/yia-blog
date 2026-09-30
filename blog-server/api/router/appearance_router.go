package router

import (
	"blog-server/api/handler"
	"github.com/gin-gonic/gin"
)

func (r *Router) appearanceRouter(v1 *gin.RouterGroup) {
	h := handler.NewAppearanceHandler(r.c.AppearanceApp)
	g := v1.Group("/appearance")
	g.GET("/today", h.Today)
	g.GET("/images/:id", h.Image)
}
