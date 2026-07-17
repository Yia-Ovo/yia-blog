// CommentDOToModel DO转Model
package convert

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
)

func CommentDOToModel(do *do.CommentDO) *model.CommentModel {
	return &model.CommentModel{
		ID:        do.ID,
		UserID:    do.UserID,
		ArticleID: do.ArticleID,
		ParentID:  do.ParentID,
		AdminID:   do.AdminID,
		Nickname:  do.Nickname,
		Email:     do.Email,
		Avatar:    do.Avatar,
		Content:   do.Content,
		IP:        do.IP,
		UserAgent: do.UserAgent,
		Status:    do.Status,
		Likes:     do.Likes,
		CreateAt:  do.CreateAt,
		UpdateAt:  do.UpdateAt,
		DeletedAt: do.DeletedAt,
	}
}

// CommentModelToDO Model转DO
func CommentModelToDO(model *model.CommentModel) *do.CommentDO {
	return &do.CommentDO{
		ID:        model.ID,
		UserID:    model.UserID,
		ArticleID: model.ArticleID,
		ParentID:  model.ParentID,
		AdminID:   model.AdminID,
		Nickname:  model.Nickname,
		Email:     model.Email,
		Avatar:    model.Avatar,
		Content:   model.Content,
		IP:        model.IP,
		UserAgent: model.UserAgent,
		Status:    model.Status,
		Likes:     model.Likes,
		CreateAt:  model.CreateAt,
		UpdateAt:  model.UpdateAt,
		DeletedAt: model.DeletedAt,
	}
}

// CommentModelListToDOList 模型列表转DO列表
func CommentModelListToDOList(list []*model.CommentModel) []do.CommentDO {
	res := make([]do.CommentDO, 0, len(list))
	for _, v := range list {
		res = append(res, *CommentModelToDO(v))
	}
	return res
}
