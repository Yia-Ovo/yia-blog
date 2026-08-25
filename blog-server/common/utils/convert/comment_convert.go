// CommentDOToModel DO转Model
package convert

import (
	"blog-server/api/dto"
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

// CommentDOToVO DO 转公开展示 VO。
// 只映射访客可见字段，Email、IP、UserAgent、Status、UserID、AdminID、DeletedAt
// 一律不参与映射，保证公开响应不携带隐私和内部状态。
func CommentDOToVO(d *do.CommentDO) *dto.CommentVO {
	if d == nil {
		return nil
	}
	return &dto.CommentVO{
		ID:        d.ID,
		ArticleID: d.ArticleID,
		ParentID:  d.ParentID,
		Nickname:  d.Nickname,
		Avatar:    d.Avatar,
		Content:   d.Content,
		Likes:     d.Likes,
		CreateAt:  d.CreateAt,
	}
}

// CommentDOListToVOList DO 列表转公开展示 VO 列表。
func CommentDOListToVOList(list []do.CommentDO) []dto.CommentVO {
	res := make([]dto.CommentVO, 0, len(list))
	for i := range list {
		res = append(res, *CommentDOToVO(&list[i]))
	}
	return res
}
