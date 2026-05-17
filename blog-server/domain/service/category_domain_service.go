package service

import (
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
	"errors"

	"go.uber.org/zap"
)

type CategoryDomainService struct {
	categoryMysqlRepo repo.CategoryMysqlRepo
	categoryRedisRepo repo.CategoryRedisRepo
}

func NewCategoryDomainService(
	categoryRepo repo.CategoryMysqlRepo,
	categoryRedisRepo repo.CategoryRedisRepo,
) *CategoryDomainService {
	return &CategoryDomainService{
		categoryMysqlRepo: categoryRepo,
		categoryRedisRepo: categoryRedisRepo,
	}
}

// CreateCategory 创建分类
func (c *CategoryDomainService) CreateCategory(ctx context.Context, categoryDO *do.CategoryDO) error {
	if categoryDO.Name == "" {
		logger.Warn("创建分类失败：名称不能为空", zap.String("category_name", categoryDO.Name))
		return common.ErrCategoryNameEmpty
	}
	if len(categoryDO.Name) > 64 {
		logger.Warn("创建分类失败：名称超长",
			zap.String("category_name", categoryDO.Name),
			zap.Uint64("length", uint64(len(categoryDO.Name))),
		)
		return common.ErrCategoryNameTooLong
	}
	if categoryDO.Sort < 0 {
		logger.Warn("创建分类失败：排序值非法", zap.Int("sort", categoryDO.Sort))
		return common.ErrCategorySortInvalid
	}

	categoryModel := convert.CategoryDOToModel(categoryDO)

	err := c.categoryMysqlRepo.CreateCategory(ctx, categoryModel)
	if err != nil {
		logger.Error("创建分类数据库操作失败",
			zap.String("category_name", categoryDO.Name),
			zap.Error(err),
		)
		return err
	}

	_ = c.categoryRedisRepo.DelCategoryListFromRedis(ctx)
	logger.Info("创建分类成功，缓存已刷新", zap.String("category_name", categoryDO.Name))
	return nil
}

func (c *CategoryDomainService) ListCategoryByPage(ctx context.Context, query *do.CategoryQuery) ([]do.CategoryDO, int64, error) {
	if query.PageNum < 1 {
		query.PageNum = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 10
	}

	cacheList, err := c.categoryRedisRepo.GetCategoryListFromRedis(ctx)
	if err == nil {
		logger.Info("分类列表命中缓存", zap.Int("cache_count", len(cacheList)))
		return convert.CategoryModelListToDOList(cacheList), int64(len(cacheList)), nil
	}
	// 缓存不存在/异常，降级查询数据库
	if !errors.Is(err, common.CacheNotExist) {
		logger.Warn("分类列表缓存异常，降级查询DB", zap.Error(err))
	}

	// 调用仓储层查询DB
	list, total, err := c.categoryMysqlRepo.ListCategoryByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询分类失败",
			zap.Int("page_num", query.PageNum),
			zap.Int("page_size", query.PageSize),
			zap.Error(err),
		)
		return nil, 0, common.DatabaseErr
	}

	modelList := convert.CategoryDOListToModelList(list)
	_ = c.categoryRedisRepo.SetCategoryListToRedis(ctx, modelList)

	logger.Info("分页查询分类成功", zap.Int64("total", total))
	return list, total, nil
}

func (c *CategoryDomainService) GetCategoryById(ctx context.Context, id uint64) (*do.CategoryDO, error) {
	if id <= 0 {
		logger.Warn("根据ID查询分类失败：ID非法", zap.Uint64("category_id", id))
		return nil, common.ParamErr
	}

	categoryModel, err := c.categoryRedisRepo.GetCategoryFromRedis(ctx, id)
	if err == nil {
		logger.Info("分类详情命中缓存", zap.Uint64("category_id", id))
		return convert.CategoryModelToDO(categoryModel), nil
	}
	// 缓存不存在/异常，降级查询数据库
	if !errors.Is(err, common.CacheNotExist) {
		logger.Warn("分类详情缓存异常，降级查询DB", zap.Uint64("category_id", id), zap.Error(err))
	}

	// 调用仓储层查询DB
	categoryDO, err := c.categoryMysqlRepo.GetCategoryById(ctx, id)
	if err != nil {
		logger.Error("根据ID查询分类失败", zap.Uint64("category_id", id), zap.Error(err))
		return nil, common.DatabaseErr
	}

	if categoryDO == nil {
		// 数据不存在：写入空值缓存，防缓存穿透
		_ = c.categoryRedisRepo.SetCategoryToRedis(ctx, id, &model.CategoryModel{})
		logger.Warn("分类不存在，写入空值缓存", zap.Uint64("category_id", id))
		return nil, common.CategoryNotFoundErr
	}

	// 数据存在：写入正常缓存
	model := convert.CategoryDOToModel(categoryDO)
	_ = c.categoryRedisRepo.SetCategoryToRedis(ctx, id, model)

	logger.Info("根据ID查询分类成功", zap.Uint64("category_id", id))
	return categoryDO, nil
}

func (c *CategoryDomainService) UpdateCategory(ctx context.Context, categoryDO *do.CategoryDO) error {
	if categoryDO.ID <= 0 {
		logger.Warn("更新分类失败：ID不能为空", zap.Uint64("category_id", categoryDO.ID))
		return common.ParamErr
	}
	if categoryDO.Name == "" {
		logger.Warn("更新分类失败：名称不能为空", zap.Uint64("category_id", categoryDO.ID))
		return common.ParamErr
	}
	if len(categoryDO.Name) > 64 {
		logger.Warn("更新分类失败：名称超长", zap.Uint64("category_id", categoryDO.ID))
		return common.ParamErr
	}
	if categoryDO.Sort < 0 {
		logger.Warn("更新分类失败：排序值非法", zap.Uint64("category_id", categoryDO.ID))
		return common.ParamErr
	}

	categoryModel := convert.CategoryDOToModel(categoryDO)

	err := c.categoryMysqlRepo.UpdateCategory(ctx, categoryModel)
	if err != nil {
		logger.Error("更新分类数据库操作失败", zap.Uint64("category_id", categoryDO.ID), zap.Error(err))
		return common.DatabaseErr
	}

	_ = c.categoryRedisRepo.DelCategoryFromRedis(ctx, categoryDO.ID)
	_ = c.categoryRedisRepo.DelCategoryListFromRedis(ctx)
	logger.Info("更新分类成功，缓存已删除", zap.Uint64("category_id", categoryDO.ID))
	return nil
}

func (c *CategoryDomainService) DeleteCategory(ctx context.Context, id uint64) error {
	if id <= 0 {
		logger.Warn("删除分类失败：ID非法", zap.Uint64("category_id", id))
		return common.ParamErr
	}

	err := c.categoryMysqlRepo.DeleteCategory(ctx, id)
	if err != nil {
		logger.Error("删除分类数据库操作失败", zap.Uint64("category_id", id), zap.Error(err))
		return common.DatabaseErr
	}

	_ = c.categoryRedisRepo.DelCategoryFromRedis(ctx, id)
	_ = c.categoryRedisRepo.DelCategoryListFromRedis(ctx)
	logger.Info("删除分类成功，缓存已删除", zap.Uint64("category_id", id))
	return nil
}
