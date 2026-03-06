/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-09 02:03:09
 * @FilePath: \yia-blog\blog-server\api\router\router.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package router

import (
	"blog-server/api/handler"
	"blog-server/app/service"
	"blog-server/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Router struct {
	FullPPROF bool
	rootPath  string
	conf      *config.GlobalConfig
	checkFunc func() error // 健康检查函数
	// 注入用户应用服务
	userAppSrv *service.UserAppService
}

func NewRouter(conf *config.GlobalConfig, userAppSrv *service.UserAppService, checkFunc func() error) *Router {
	return &Router{
		FullPPROF:  conf.Server.EnablePprof, // 是否开启pprof
		rootPath:   "/api/blog",
		conf:       conf,
		checkFunc:  checkFunc,
		userAppSrv: userAppSrv,
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
	// 版本化路由组：/api/blog/v1
	v1 := rg.Group("/v1")
	{
		r.userRouter(v1) // 新增：注册用户路由
	}
}

// registerUserRouter 注册用户相关路由：模块化拆分，不污染核心代码
func (r *Router) userRouter(v1 *gin.RouterGroup) {
	// 用户子路由组：/api/blog/v1/user
	userGroup := v1.Group("/user")
	// 创建用户处理函数实例
	userHandler := handler.NewUserHandler(r.userAppSrv)
	// 绑定POST请求：/api/blog/v1/user/register → userHandler.Register
	userGroup.POST("/register", userHandler.Register)
	// 绑定POST请求：/api/blog/v1/user/findUserByNickname → userHandler.FindUserByNickname
	userGroup.GET("/findUserByNickname", userHandler.FindUserByNickname)
	userGroup.GET("/updateUserInfo", userHandler.UpdateUserInfo)
	userGroup.POST("/login", userHandler.Login)
}
