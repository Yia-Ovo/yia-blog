package router

import (
	"blog-server/api/handler"
	"github.com/gin-gonic/gin"
)

func (r *Router) avatarRouter(v1 *gin.RouterGroup) {
	h := handler.NewAvatarHandler(r.c.AvatarApp)
	v1.GET("/avatar/images/:name", h.Image)
}
