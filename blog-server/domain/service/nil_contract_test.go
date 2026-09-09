package service

import (
	"context"
	"testing"

	"blog-server/common"
	"blog-server/domain/do"
)

// 本文件配套 convert 层的统一 nil 契约（见 convert/nil_contract_test.go）：
// convert 对 nil 入参返回 nil 后，领域层必须在第一次解引用之前把 nil 拦下，
// 否则 nil 会先在字段读取处 panic，或作为 nil Model 流到持久化层。
//
// 三个用例都刻意传 nil repo 构造领域服务：判空发生在触达 Repo 之前，
// 所以正常路径不会用到 repo；一旦有人把判空移到 Repo 调用之后，
// 这里会 panic 而不是静默通过——这正是我们想要的失败信号。

// TestCreateArticle_RejectsNilDO 创建文章：nil 领域对象应在解引用 Title 之前被拦下。
func TestCreateArticle_RejectsNilDO(t *testing.T) {
	svc := NewArticleDomainService(nil)

	err := svc.CreateArticle(context.Background(), nil)

	if err != common.ParamErr {
		t.Errorf("err = %v, 期望 %v", err, common.ParamErr)
	}
}

// TestUpdateUserInfo_RejectsNilUpdateData 更新用户信息：nil 更新数据应在按字段覆盖之前被拦下。
func TestUpdateUserInfo_RejectsNilUpdateData(t *testing.T) {
	svc := NewUserDomainService(nil)

	err := svc.UpdateUserInfo(context.Background(), 1, nil)

	if err != common.ParamErr {
		t.Errorf("err = %v, 期望 %v", err, common.ParamErr)
	}
}

// TestUpdateAdminInfo_RejectsNilData 更新管理员信息：nil 更新数据应在按字段覆盖之前被拦下。
func TestUpdateAdminInfo_RejectsNilData(t *testing.T) {
	svc := NewAdminDomainService(nil)

	err := svc.UpdateAdminInfo(context.Background(), 1, nil)

	if err != common.ParamErr {
		t.Errorf("err = %v, 期望 %v", err, common.ParamErr)
	}
}

// TestCreateArticle_NonNilDOPassesNilCheck 反向断言：合法 DO 不应被 nil 守卫误拦。
// 空标题走的是既有的领域校验（ErrArticleTitleEmpty），不是 ParamErr——
// 借此确认新增的判空没有把正常入参一起挡掉。
func TestCreateArticle_NonNilDOPassesNilCheck(t *testing.T) {
	svc := NewArticleDomainService(nil)

	err := svc.CreateArticle(context.Background(), &do.ArticleCreate{Title: ""})

	if err == common.ParamErr {
		t.Error("合法（非 nil）DO 被 nil 守卫误拦成 ParamErr")
	}
	if err != common.ErrArticleTitleEmpty {
		t.Errorf("err = %v, 期望 %v", err, common.ErrArticleTitleEmpty)
	}
}
