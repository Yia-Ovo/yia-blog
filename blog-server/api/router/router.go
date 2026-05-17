/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-17 12:50:52
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
	"blog-server/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Router struct {
	FullPPROF bool
	rootPath  string
	conf      *config.GlobalConfig
	checkFunc func() error // 健康检查函数
	// 注入用户应用服务
	userAppSrv     *service.UserAppService
	categoryAppSrv *service.CategoryAppService
	articleAppSrv  *service.ArticleAppService
	adminAppSrv    *service.AdminAppService
}

func NewRouter(conf *config.GlobalConfig, userAppSrv *service.UserAppService, categoryAppSrv *service.CategoryAppService, articleAppSrv *service.ArticleAppService, adminAppService *service.AdminAppService, checkFunc func() error) *Router {
	return &Router{
		FullPPROF:      conf.Server.EnablePprof, // 是否开启pprof
		rootPath:       "/api/blog",
		conf:           conf,
		checkFunc:      checkFunc,
		userAppSrv:     userAppSrv,
		categoryAppSrv: categoryAppSrv,
		articleAppSrv:  articleAppSrv,
		adminAppSrv:    adminAppService,
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

	userGroup.Use(middleware.DefaultUserAuth()) // 使用中间件进行用户认证
	{
		userGroup.GET("/getInfo", userHandler.GetInfo)      // GetInfo接口
		userGroup.POST("/updatePWD", userHandler.UpdatePWD) // UpdatePWD接口
		userGroup.POST("/logout", userHandler.Logout)       // UpdatePWD接口
	}
	// {
	// 直接把中间件写在路由里，确保和Handler共用同一个ctx
	// userGroup.GET("/getInfo", middleware.DefaultUserAuth(), userHandler.GetInfo)
	// }
}

// ---------------- 管理员路由 ----------------
func (r *Router) adminRouter(v1 *gin.RouterGroup) {
	// 管理员路由组：/api/blog/v1/admin
	adminGroup := v1.Group("/admin")
	// 创建管理员处理函数实例
	adminHandler := handler.NewAdminHandler(r.adminAppSrv)

	// 公开接口：无需管理员认证（仅登录接口）
	adminGroup.POST("/login", adminHandler.Login)

	// 私有接口：需要管理员登录认证（使用AdminAuth中间件）
	adminGroup.Use(middleware.DefaultAdminAuth())
	{
		adminGroup.GET("/getInfo", adminHandler.GetInfo)                  // 获取管理员信息
		adminGroup.POST("/updatePWD", adminHandler.UpdatePWD)             // 修改密码
		adminGroup.POST("/logout", adminHandler.Logout)                   // 登出
		adminGroup.POST("/updateAdminInfo", adminHandler.UpdateAdminInfo) // 更新信息
	}
}

// ---------------- 分类路由 ----------------
func (r *Router) categoryRouter(v1 *gin.RouterGroup) {
	// 分类子路由组：/api/blog/v1/category
	categoryGroup := v1.Group("/category")
	// 创建分类处理函数实例
	categoryHandler := handler.NewCategoryHandler(r.categoryAppSrv)

	// 公开接口：无需登录认证（和用户模块公开接口写法一致）
	categoryGroup.GET("/listCategory", categoryHandler.ListCategory)       // 分页查询分类
	categoryGroup.GET("/getCategoryById", categoryHandler.GetCategoryById) // 根据ID查询分类

	// 私有接口：需要用户登录认证（和用户模块写法完全一致）
	categoryGroup.Use(middleware.DefaultUserAuth())
	{
		categoryGroup.POST("/createCategory", categoryHandler.CreateCategory) // 创建分类
		categoryGroup.POST("/updateCategory", categoryHandler.UpdateCategory) // 更新分类
		categoryGroup.POST("/deleteCategory", categoryHandler.DeleteCategory) // 删除分类
	}
}

// ---------------- 文章路由  ----------------
func (r *Router) articleRouter(v1 *gin.RouterGroup) {
	// 文章子路由组：/api/blog/v1/article
	articleGroup := v1.Group("/article")
	// 创建文章处理函数实例
	articleHandler := handler.NewArticleHandler(r.articleAppSrv)

	// 公开接口：无需登录认证（列表、详情查询）
	articleGroup.GET("/listArticle", articleHandler.ListArticle)       // 分页查询文章
	articleGroup.GET("/getArticleById", articleHandler.GetArticleById) // 根据ID查询文章详情

	// 私有接口：需要用户登录认证（增删改）
	articleGroup.Use(middleware.DefaultUserAuth())
	{
		articleGroup.POST("/createArticle", articleHandler.CreateArticle) // 创建文章
		articleGroup.POST("/updateArticle", articleHandler.UpdateArticle) // 更新文章
		articleGroup.POST("/deleteArticle", articleHandler.DeleteArticle) // 删除文章
	}
}
