/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 03:11:41
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-07 05:22:12
 * @FilePath: \yia-blog\blog-server\common\utils\convert\article_convert.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
// convert/article_convert.go
package convert

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
)

// ArticleModelToDO Model转DO
func ArticleModelToDO(m *model.ArticleModel) *do.ArticleDO {
	if m == nil {
		return nil
	}
	return &do.ArticleDO{
		ID:            m.ID,
		AdminID:       m.AdminID,
		Title:         m.Title,
		Cover:         m.Cover,
		Summary:       m.Summary,
		Content:       m.Content,
		CategoryID:    m.CategoryID,
		TagIDs:        m.TagIDs,
		Status:        m.Status,
		Sort:          m.Sort,
		Views:         m.Views,
		Likes:         m.Likes,
		Comments:      m.Comments,
		IsTop:         m.IsTop,
		CommentStatus: m.CommentStatus,
		PublishAt:     m.PublishAt,
		CreateAt:      m.CreateAt,
		UpdateAt:      m.UpdateAt,
	}
}

// ArticleDOToModel DO转Model
func ArticleDOToModel(d *do.ArticleDO) *model.ArticleModel {
	if d == nil {
		return nil
	}
	return &model.ArticleModel{
		ID:            d.ID,
		AdminID:       d.AdminID,
		Title:         d.Title,
		Cover:         d.Cover,
		Summary:       d.Summary,
		Content:       d.Content,
		CategoryID:    d.CategoryID,
		TagIDs:        d.TagIDs,
		Status:        d.Status,
		Sort:          d.Sort,
		Views:         d.Views,
		Likes:         d.Likes,
		Comments:      d.Comments,
		IsTop:         d.IsTop,
		CommentStatus: d.CommentStatus,
		PublishAt:     d.PublishAt,
	}
}

// ArticleModelListToDO 模型列表转DO列表
func ArticleModelListToDOList(list *[]model.ArticleModel) []do.ArticleDO {
	res := make([]do.ArticleDO, 0, len(*list))
	for _, v := range *list {
		res = append(res, *ArticleModelToDO(&v))
	}
	return res
}

// ArticleDOListToModelList DO列表转模型列表
func ArticleDOListToModelList(list *[]do.ArticleDO) []model.ArticleModel {
	res := make([]model.ArticleModel, 0, len(*list))
	for _, v := range *list {
		res = append(res, *ArticleDOToModel(&v))
	}
	return res
}
