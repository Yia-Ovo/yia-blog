package router

import (
	"blog-server/api/handler"
	"github.com/gin-gonic/gin"
)

func (r *Router) weatherRouter(v1 *gin.RouterGroup) {
	v1.GET("/weather/current", handler.NewWeatherHandler(r.c.WeatherApp).Current)
}
