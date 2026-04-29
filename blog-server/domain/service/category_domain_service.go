package service

import (
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"

	"go.uber.org/zap"
)

type CategoryDomainService struct {
	categoryMysqlRepo repo.CategoryMysqlRepo
}

func NewCategoryDomainService(categoryRepo repo.CategoryMysqlRepo) *CategoryDomainService {
	return &CategoryDomainService{
		categoryMysqlRepo: categoryRepo,
	}
}

// CreateCategory 创建分类
// 入参：领域对象 DO，出参：业务错误
func (c *CategoryDomainService) CreateCategory(ctx context.Context, categoryDO *do.CategoryDO) error {
	if categoryDO.Name == "" {
		// 使用你封装的 logger
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
		// 错误日志（适配你的logger）
		logger.Error("创建分类数据库操作失败",
			zap.String("category_name", categoryDO.Name),
			zap.Error(err),
		)
		return err
	}

	// 成功日志
	logger.Info("创建分类成功", zap.String("category_name", categoryDO.Name))
	return nil
}

func (c *CategoryDomainService) ListCategoryByPage(ctx context.Context, query *do.CategoryQuery) ([]do.CategoryDO, int64, error) {
	// 业务校验：分页参数合法性
	if query.PageNum < 1 {
		query.PageNum = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 10
	}

	// 调用仓储层
	list, total, err := c.categoryMysqlRepo.ListCategoryByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询分类失败",
			zap.Int("page_num", query.PageNum),
			zap.Int("page_size", query.PageSize),
			zap.Error(err),
		)
		return nil, 0, common.DatabaseErr
	}

	logger.Info("分页查询分类成功",
		zap.Int64("total", total),
		zap.Int("page_num", query.PageNum),
	)
	return list, total, nil
}

func (c *CategoryDomainService) GetCategoryById(ctx context.Context, id uint64) (*do.CategoryDO, error) {
	// 参数校验
	if id <= 0 {
		logger.Warn("根据ID查询分类失败：ID非法", zap.Uint64("category_id", id))
		return nil, common.ParamErr
	}

	// 调用仓储层
	categoryDO, err := c.categoryMysqlRepo.GetCategoryById(ctx, id)
	if err != nil {
		logger.Error("根据ID查询分类失败",
			zap.Uint64("category_id", id),
			zap.Error(err),
		)
		return nil, common.DatabaseErr
	}

	// 数据不存在校验
	if categoryDO == nil {
		logger.Warn("分类不存在", zap.Uint64("category_id", id))
		return nil, common.CategoryNotFoundErr
	}

	logger.Info("根据ID查询分类成功", zap.Uint64("category_id", id))
	return categoryDO, nil
}

func (c *CategoryDomainService) UpdateCategory(ctx context.Context, categoryDO *do.CategoryDO) error {
	// 业务规则校验
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

	// DO 转换为 Model
	categoryModel := convert.CategoryDOToModel(categoryDO)

	// 调用仓储层
	err := c.categoryMysqlRepo.UpdateCategory(ctx, categoryModel)
	if err != nil {
		logger.Error("更新分类数据库操作失败",
			zap.Uint64("category_id", categoryDO.ID),
			zap.Error(err),
		)
		return common.DatabaseErr
	}

	logger.Info("更新分类成功", zap.Uint64("category_id", categoryDO.ID))
	return nil
}

func (c *CategoryDomainService) DeleteCategory(ctx context.Context, id uint64) error {
	// 参数校验
	if id <= 0 {
		logger.Warn("删除分类失败：ID非法", zap.Uint64("category_id", id))
		return common.ParamErr
	}

	// 调用仓储层
	err := c.categoryMysqlRepo.DeleteCategory(ctx, id)
	if err != nil {
		logger.Error("删除分类数据库操作失败",
			zap.Uint64("category_id", id),
			zap.Error(err),
		)
		return common.DatabaseErr
	}

	logger.Info("删除分类成功", zap.Uint64("category_id", id))
	return nil
}
