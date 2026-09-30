/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-16 09:13:28
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-02 18:42:21
 * @FilePath: \yia-blog\blog-server\ioc\container.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package ioc

import (
	appSrv "blog-server/app/service"
	"blog-server/assets"
	"blog-server/config"
	domainRepo "blog-server/domain/repo"
	domainSrv "blog-server/domain/service"
	infraRepo "blog-server/infra/repo"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var AppContainer *Container

// Container 真正的DI容器，所有依赖放在结构体里
type Container struct {
	AppearanceRepo          domainRepo.AppearanceRepo
	AppearanceDomain        *domainSrv.AppearanceDomainService
	AppearanceApp           *appSrv.AppearanceAppService
	CommentReviewRepo       domainRepo.CommentReviewRepo
	CommentModerationDomain *domainSrv.CommentModerationDomainService
	CommentModerationApp    *appSrv.CommentModerationAppService
	// 基础
	Config *config.GlobalConfig
	DB     *gorm.DB
	Redis  *redis.Client

	// Repo 仓储
	UserRepo      domainRepo.UserMysqlRepo
	UserRedis     domainRepo.UserRedisRepo
	AdminRepo     domainRepo.AdminMysqlRepo
	AdminRedis    domainRepo.AdminRedisRepo
	CategoryRepo  domainRepo.CategoryMysqlRepo
	CategoryRedis domainRepo.CategoryRedisRepo
	ArticleRepo   domainRepo.ArticleMysqlRepo
	ArticleRedis  domainRepo.ArticleRedisRepo
	CommentRepo   domainRepo.CommentMysqlRepo
	CommentRedis  domainRepo.CommentRedisRepo

	// Domain 领域服务
	CommentDomain *domainSrv.CommentDomainService
	// Admain 领域服务
	AdminDomain *domainSrv.AdminDomainService
	// User 领域服务
	UserDomain *domainSrv.UserDomainService
	// Category 领域服务
	CategoryDomain *domainSrv.CategoryDomainService
	// Article 领域服务
	ArticleDomain           *domainSrv.ArticleDomainService
	ArticleManagementDomain *domainSrv.ArticleManagementDomainService

	// App 应用服务
	UserApp              *appSrv.UserAppService
	AdminApp             *appSrv.AdminAppService
	CategoryApp          *appSrv.CategoryAppService
	ArticleApp           *appSrv.ArticleAppService
	CommentApp           *appSrv.CommentAppService
	ArticleManagementApp *appSrv.ArticleManagementAppService
}

// NewContainer 容器构造函数：自动依赖注入（DDD 标准）
func NewContainer(
	db *gorm.DB,
	rdb *redis.Client,
	conf *config.GlobalConfig,
) *Container {
	c := &Container{
		Config: conf,
		DB:     db,
		Redis:  rdb,
	}

	// ========== 初始化仓储 ==========
	c.UserRepo = infraRepo.NewUserMysqlRepo(db)
	c.UserRedis = infraRepo.NewUserRedisRepo(rdb, conf)

	c.AdminRepo = infraRepo.NewAdminMysqlRepo(db)
	c.AdminRedis = infraRepo.NewAdminRedisRepo(rdb, conf)

	c.CategoryRepo = infraRepo.NewCategoryMysqlRepo(db)
	c.CategoryRedis = infraRepo.NewCategoryRedisRepo(rdb, conf)

	c.ArticleRepo = infraRepo.NewArticleMysqlRepo(db)
	c.ArticleRedis = infraRepo.NewArticleRedisRepo(rdb, conf)

	commentRepo := infraRepo.NewCommentMysqlRepo(db)
	c.CommentRepo = commentRepo
	c.CommentReviewRepo = commentRepo
	c.CommentRedis = infraRepo.NewCommentRedisRepo(rdb, conf)

	// ========== 初始化领域服务 ==========
	c.CommentDomain = domainSrv.NewCommentDomainService(c.CommentRepo, c.ArticleRepo)
	c.AdminDomain = domainSrv.NewAdminDomainService(c.AdminRepo)
	c.UserDomain = domainSrv.NewUserDomainService(c.UserRepo)
	c.CategoryDomain = domainSrv.NewCategoryDomainService(c.CategoryRepo)
	c.ArticleDomain = domainSrv.NewArticleDomainService(c.ArticleRepo)
	c.ArticleManagementDomain = domainSrv.NewArticleManagementDomainService(c.ArticleDomain, c.UserDomain, c.AdminDomain)

	// ========== 初始化应用服务 ==========
	c.UserApp = appSrv.NewUserAppService(c.UserDomain, c.UserRedis)
	c.AdminApp = appSrv.NewAdminAppService(c.AdminDomain, c.AdminRedis)
	c.CategoryApp = appSrv.NewCategoryAppService(c.CategoryDomain, c.CategoryRedis)
	c.ArticleApp = appSrv.NewArticleAppService(c.ArticleDomain, c.ArticleRedis, c.UserDomain, c.AdminDomain)
	c.CommentApp = appSrv.NewCommentAppService(c.CommentDomain, c.CommentRedis)
	c.ArticleManagementApp = appSrv.NewArticleManagementAppService(c.ArticleManagementDomain)

	c.CommentModerationDomain = domainSrv.NewCommentModerationDomainService(c.CommentDomain, c.AdminDomain, c.CommentReviewRepo)
	c.CommentModerationApp = appSrv.NewCommentModerationAppService(c.CommentModerationDomain, c.CommentRedis)
	c.AppearanceRepo = infraRepo.NewAppearanceFileRepo(assets.DailyArtwork)
	c.AppearanceDomain = domainSrv.NewAppearanceDomainService(c.AppearanceRepo)
	c.AppearanceApp = appSrv.NewAppearanceAppService(c.AppearanceDomain)
	return c
}
