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
	"blog-server/common/consts"
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
	WeatherApp              *appSrv.WeatherAppService
	ArticleImageRepo        domainRepo.ArticleImageRepo
	ArticleImageDomain      *domainSrv.ArticleImageDomainService
	ArticleImageApp         *appSrv.ArticleImageAppService
	AvatarRepo              domainRepo.AvatarRepo
	AvatarDomain            *domainSrv.AvatarDomainService
	AvatarApp               *appSrv.AvatarAppService
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
	// ReadCacheRedis 由容器创建并交给入口关闭，使用独立短超时池隔离鉴权 Redis。
	ReadCacheRedis *redis.Client

	// Repo 仓储
	UserRepo      domainRepo.UserMysqlRepo
	UserRedis     domainRepo.UserRedisRepo
	AdminRepo     domainRepo.AdminMysqlRepo
	AdminRedis    domainRepo.AdminRedisRepo
	CategoryRepo  domainRepo.CategoryMysqlRepo
	CategoryRedis domainRepo.ReadCacheRepo
	ArticleRepo   domainRepo.ArticleMysqlRepo
	ArticleRedis  domainRepo.ReadCacheRepo
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
		Config:         conf,
		DB:             db,
		Redis:          rdb,
		ReadCacheRedis: newReadCacheClient(rdb),
	}

	// ========== 初始化仓储 ==========
	c.UserRepo = infraRepo.NewUserMysqlRepo(db)
	c.UserRedis = infraRepo.NewUserRedisRepo(rdb, conf)

	c.AdminRepo = infraRepo.NewAdminMysqlRepo(db)
	c.AdminRedis = infraRepo.NewAdminRedisRepo(rdb, conf)

	c.CategoryRepo = infraRepo.NewCategoryMysqlRepo(db)
	c.CategoryRedis = infraRepo.NewReadCacheRedisRepo(c.ReadCacheRedis, consts.ReadCacheCategoryNamespace)

	c.ArticleRepo = infraRepo.NewArticleMysqlRepo(db)
	c.ArticleRedis = infraRepo.NewReadCacheRedisRepo(c.ReadCacheRedis, consts.ReadCacheArticleNamespace)

	commentRepo := infraRepo.NewCommentMysqlRepo(db)
	c.CommentRepo = commentRepo
	c.CommentReviewRepo = commentRepo
	c.CommentRedis = infraRepo.NewCommentRedisRepo(rdb, conf)

	// ========== 初始化领域服务 ==========
	c.CommentDomain = domainSrv.NewCommentDomainService(c.CommentRepo, c.ArticleRepo)
	c.AdminDomain = domainSrv.NewAdminDomainService(c.AdminRepo)
	c.UserDomain = domainSrv.NewUserDomainService(c.UserRepo)
	c.AvatarRepo = infraRepo.NewAvatarFileRepo(conf.Storage.AvatarDir)
	c.AvatarDomain = domainSrv.NewAvatarDomainService(c.AdminDomain, c.UserDomain)
	c.AvatarApp = appSrv.NewAvatarAppService(c.AvatarDomain, c.AvatarRepo, c.AdminRedis, c.UserRedis)
	c.ArticleImageRepo = infraRepo.NewArticleImageFileRepo(conf.Storage.ArticleImageDir)
	c.ArticleImageDomain = domainSrv.NewArticleImageDomainService(c.AdminDomain, c.UserDomain)
	c.ArticleImageApp = appSrv.NewArticleImageAppService(c.ArticleImageDomain, c.ArticleImageRepo)
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
	c.WeatherApp = appSrv.NewWeatherAppService(domainSrv.NewWeatherDomainService(), infraRepo.NewWeatherHTTPRepo())
	return c
}
