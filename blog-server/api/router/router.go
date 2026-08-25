/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-08-13 18:26:49
 * @FilePath: \yia-blog\blog-server\api\router\router.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package router

import (
	"blog-server/api/handler"
	"blog-server/config"
	"blog-server/ioc"
	"blog-server/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Router 只依赖容器，不再依赖一堆服务
type Router struct {
	FullPPROF bool
	rootPath  string
	conf      *config.GlobalConfig
	c         *ioc.Container // 唯一依赖：容器
	checkFunc func() error
}

// NewRouter 构造函数极简
func NewRouter(
	conf *config.GlobalConfig,
	c *ioc.Container,
	checkFunc func() error,
) *Router {
	return &Router{
		FullPPROF: conf.Server.EnablePprof,
		rootPath:  "/api/blog",
		conf:      conf,
		c:         c,
		checkFunc: checkFunc,
	}
}

func (r *Router) checkServer() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := r.checkFunc(); err != nil {
			ctx.JSON(500, gin.H{"message": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (r *Router) Register(engine *gin.Engine) {
	if r.conf.Server.EnablePprof {
		SetupPprof(engine, "/debug/pprof")
	}
	engine.Any("/ping", r.checkServer())

	root := engine.Group(r.rootPath)
	r.route(root)
}

func (r *Router) route(rg *gin.RouterGroup) {
	v1 := rg.Group("/v1")
	{
		r.userRouter(v1)
		r.adminRouter(v1)
		r.categoryRouter(v1)
		r.articleRouter(v1)
		r.commentRouter(v1)
	}
}

// ===================== 路由内部从容器取服务 =====================
func (r *Router) userRouter(v1 *gin.RouterGroup) {
	g := v1.Group("/user")
	h := handler.NewUserHandler(r.c.UserApp)

	g.POST("/register", h.Register)
	g.GET("/findUserByNickname", h.FindUserByNickname)
	g.POST("/login", h.Login)

	g.Use(middleware.DefaultUserAuth())
	{
		g.GET("/getInfo", h.GetInfo)
		g.POST("/updatePWD", h.UpdatePWD)
		g.POST("/logout", h.Logout)
		g.POST("/updateUserInfo", h.UpdateUserInfo) // 改成 POST + 登录后
	}
}

func (r *Router) adminRouter(v1 *gin.RouterGroup) {
	g := v1.Group("/admin")
	h := handler.NewAdminHandler(r.c.AdminApp)

	// 1. 公开接口（无需登录）
	g.POST("/login", h.Login)

	// 2. 登录接口（所有管理员可用）
	adminAuth := g.Group("/")
	adminAuth.Use(middleware.DefaultAdminAuth())
	{
		adminAuth.GET("/getInfo", h.GetInfo)
		adminAuth.POST("/updatePWD", h.UpdatePWD)
		adminAuth.POST("/logout", h.Logout)
		adminAuth.POST("/updateAdminInfo", h.UpdateAdminInfo)
	}

	// 3. 超管专属接口（双中间件：登录 + 超管角色）
	adminSuper := g.Group("/")
	adminSuper.Use(middleware.DefaultAdminAuth(), middleware.AdminSuperAuth())
	{
		adminSuper.POST("/create", h.CreateAdmin)                 // 创建管理员
		adminSuper.GET("/list", h.ListAdmin)                      // 管理员列表
		adminSuper.POST("/updateStatus", h.UpdateAdminStatus)     // 修改状态
		adminSuper.POST("/resetPWD", h.ResetAdminPWD)             // 重置密码
		adminSuper.POST("/updateAdminRole", h.UpdateAdminRole)    // 变更角色（可造超管，权限最高）
	}
}

func (r *Router) categoryRouter(v1 *gin.RouterGroup) {
	g := v1.Group("/category")
	h := handler.NewCategoryHandler(r.c.CategoryApp)

	g.GET("/listCategory", h.ListCategory)
	g.GET("/getCategoryById", h.GetCategoryById)

	g.Use(middleware.DefaultAdminAuth(), middleware.AdminSuperAuth())
	{
		g.POST("/createCategory", h.CreateCategory)
		g.POST("/updateCategory", h.UpdateCategory)
		g.POST("/deleteCategory", h.DeleteCategory)
	}
}

func (r *Router) articleRouter(v1 *gin.RouterGroup) {
	g := v1.Group("/article")
	h := handler.NewArticleHandler(r.c.ArticleApp)

	// 公开接口（无需登录）
	g.GET("/listArticle", h.ListArticle)
	g.GET("/getArticleById", h.GetArticleById)

	// 普通用户：管理自己的文章（用户 token，仅本人）
	userGroup := g.Group("/")
	userGroup.Use(middleware.DefaultUserAuth())
	{
		userGroup.POST("/createArticle", h.CreateArticle)
		userGroup.POST("/updateArticle", h.UpdateArticle)
		userGroup.POST("/deleteArticle", h.DeleteArticle)
	}

	// 管理员：管理全部文章（管理员 token，不限作者）
	adminGroup := g.Group("/")
	adminGroup.Use(middleware.DefaultAdminAuth())
	{
		adminGroup.POST("/adminCreateArticle", h.AdminCreateArticle)
		adminGroup.POST("/adminUpdateArticle", h.AdminUpdateArticle)
		adminGroup.POST("/adminDeleteArticle", h.AdminDeleteArticle)
	}
}

func (r *Router) commentRouter(v1 *gin.RouterGroup) {
	g := v1.Group("/comment")
	h := handler.NewCommentHandler(r.c.CommentApp)

	g.GET("/listComment", h.ListCommentByPage)
	g.GET("/getCommentById", h.GetCommentByID)
	g.GET("/getArticleCommentCount", h.GetArticleCommentCount)

	g.Use(middleware.DefaultUserAuth())
	{
		g.POST("/createComment", h.CreateComment)
		g.POST("/updateComment", h.UpdateComment)
		g.POST("/deleteComment", h.DeleteComment)
		g.POST("/likeComment", h.LikeComment)
		g.POST("/unlikeComment", h.DisLikeComment)
	}
}
