/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-16 09:13:28
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-26 00:25:53
 * @FilePath: \yia-blog\blog-server\ioc\container.go
 * @Description:IOC容器
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package ioc

import (
	appSrv "blog-server/app/service"
	"blog-server/config"
	domainRepo "blog-server/domain/repo"
	domainSrv "blog-server/domain/service"
	infraRepo "blog-server/infra/repo"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 全局单例：托管所有仓储实现
var (
	// 基础全局依赖
	Conf  *config.GlobalConfig
	DB    *gorm.DB
	Redis *redis.Client

	// ===================== 用户端仓储 =====================
	UserMysqlRepo domainRepo.UserMysqlRepo // MySQL接口
	UserRedisRepo domainRepo.UserRedisRepo // Redis接口
	// ===================== 管理员端仓储 =====================
	AdminMysqlRepo domainRepo.AdminMysqlRepo // MySQL接口
	AdminRedisRepo domainRepo.AdminRedisRepo // 管理员Redis接口

	// ===================== 分类端仓储 =====================
	CategoryMysqlRepo domainRepo.CategoryMysqlRepo // MySQL接口
	CategoryRedisRepo domainRepo.CategoryRedisRepo // Redis接口

	// ===================== 文章端仓储 =====================
	ArticleMysqlRepo domainRepo.ArticleMysqlRepo // MySQL接口
	ArticleRedisRepo domainRepo.ArticleRedisRepo // Redis接口

	// ===================== 评论端仓储 =====================
	CommentMysqlRepo domainRepo.CommentMysqlRepo // MySQL接口
	CommentRedisRepo domainRepo.CommentRedisRepo // Redis接口

	CommentDomainService *domainSrv.CommentDomainService // 评论领域服务
	CommentAppService    *appSrv.CommentAppService       // 评论应用服务
)

// Init 初始化IOC：绑定接口与实现
func Init(db *gorm.DB, rdb *redis.Client, conf *config.GlobalConfig) {
	// 1. 赋值基础依赖
	DB = db
	Redis = rdb
	Conf = conf

	// 2. 初始化基础设施实现
	//user
	UserMysqlRepo = infraRepo.NewUserMysqlRepo(db)
	UserRedisRepo = infraRepo.NewUserRedisRepo(rdb, conf)
	// admin
	AdminMysqlRepo = infraRepo.NewAdminMysqlRepo(db)
	AdminRedisRepo = infraRepo.NewAdminRedisRepo(rdb, conf)

	// category
	CategoryMysqlRepo = infraRepo.NewCategoryMysqlRepo(db)
	CategoryRedisRepo = infraRepo.NewCategoryRedisRepo(rdb, conf)

	// article
	ArticleMysqlRepo = infraRepo.NewArticleMysqlRepo(db)
	ArticleRedisRepo = infraRepo.NewArticleRedisRepo(rdb, conf)
	// comment
	CommentMysqlRepo = infraRepo.NewCommentMysqlRepo(db)
	CommentRedisRepo = infraRepo.NewCommentRedisRepo(rdb, conf)

	// 1. 初始化评论领域服务（仅依赖 MySQL 仓储，纯净DDD）
	CommentDomainService = domainSrv.NewCommentDomainService(CommentMysqlRepo)
	// 2. 初始化评论应用服务（依赖 领域服务 + Redis 仓储）
	CommentAppService = appSrv.NewCommentAppService(CommentDomainService, CommentRedisRepo)
}

// ===================== 对外获取方法 =====================
func GetUserMysqlRepo() domainRepo.UserMysqlRepo { return UserMysqlRepo }
func GetUserRedisRepo() domainRepo.UserRedisRepo { return UserRedisRepo }

func GetCategoryMysqlRepo() domainRepo.CategoryMysqlRepo { return CategoryMysqlRepo }
func GetCategoryRedisRepo() domainRepo.CategoryRedisRepo { return CategoryRedisRepo }

func GetAdminMysqlRepo() domainRepo.AdminMysqlRepo { return AdminMysqlRepo }
func GetAdminRedisRepo() domainRepo.AdminRedisRepo { return AdminRedisRepo }

func GetArticleMysqlRepo() domainRepo.ArticleMysqlRepo { return ArticleMysqlRepo }
func GetArticleRedisRepo() domainRepo.ArticleRedisRepo { return ArticleRedisRepo }

func GetCommentMysqlRepo() domainRepo.CommentMysqlRepo { return CommentMysqlRepo }
func GetCommentRedisRepo() domainRepo.CommentRedisRepo { return CommentRedisRepo }

func GetCommentDomainService() *domainSrv.CommentDomainService { return CommentDomainService }
func GetCommentAppService() *appSrv.CommentAppService          { return CommentAppService }
