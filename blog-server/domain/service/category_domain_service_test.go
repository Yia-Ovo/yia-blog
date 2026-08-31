package service

import (
	"context"
	"testing"

	"blog-server/common"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
)

// fakeCategoryMysqlRepo 只实现测试用到的方法，其余由内嵌接口占位（未实现的方法被调用会 panic，
// 正好暴露「领域层不该走到这一步」的情况）。
type fakeCategoryMysqlRepo struct {
	repo.CategoryMysqlRepo

	// created 记录最终下发到 Repo 的 Model，用于断言字段没有在转换途中被丢弃
	created      *model.CategoryModel
	createCalled bool
	createErr    error
}

func (f *fakeCategoryMysqlRepo) CreateCategory(_ context.Context, category *model.CategoryModel) error {
	f.createCalled = true
	f.created = category
	return f.createErr
}

// TestCreateCategory_PersistsSubmittedFields 钉死严重问题 11 的完整链路：
// 新建 DO 的 ID 为 0，领域层转换后交给 Repo 的 Model 必须携带真实的 Name/Sort/Status，
// 而不是被守卫替换成空 Model（那会让 category 表写入 name=” 的空行且返回成功）。
func TestCreateCategory_PersistsSubmittedFields(t *testing.T) {
	fake := &fakeCategoryMysqlRepo{}
	svc := NewCategoryDomainService(fake)

	err := svc.CreateCategory(context.Background(), &do.CategoryDO{Name: "tech", Sort: 5, Status: 1})
	if err != nil {
		t.Fatalf("CreateCategory 返回错误: %v", err)
	}

	if !fake.createCalled {
		t.Fatal("Repo.CreateCategory 未被调用")
	}
	if fake.created == nil {
		t.Fatal("下发到 Repo 的 Model 为 nil")
	}
	if fake.created.Name != "tech" {
		t.Errorf("下发的 Name = %q, 期望 %q（空字符串即严重问题 11 复发）", fake.created.Name, "tech")
	}
	if fake.created.Sort != 5 {
		t.Errorf("下发的 Sort = %d, 期望 %d", fake.created.Sort, 5)
	}
	if fake.created.Status != 1 {
		t.Errorf("下发的 Status = %d, 期望 %d", fake.created.Status, 1)
	}
}

// TestCreateCategory_RejectsNilDO 领域对象为空属于非法调用，应在触达 Repo 前就被拦下。
func TestCreateCategory_RejectsNilDO(t *testing.T) {
	fake := &fakeCategoryMysqlRepo{}
	svc := NewCategoryDomainService(fake)

	err := svc.CreateCategory(context.Background(), nil)

	if err != common.ParamErr {
		t.Errorf("err = %v, 期望 %v", err, common.ParamErr)
	}
	if fake.createCalled {
		t.Error("nil DO 不应触达 Repo")
	}
}

// TestCreateCategory_RejectsEmptyName 名称为空是既有的领域规则，补测防止回归。
func TestCreateCategory_RejectsEmptyName(t *testing.T) {
	fake := &fakeCategoryMysqlRepo{}
	svc := NewCategoryDomainService(fake)

	err := svc.CreateCategory(context.Background(), &do.CategoryDO{Name: "", Sort: 1, Status: 1})

	if err != common.ParamErr {
		t.Errorf("err = %v, 期望 %v", err, common.ParamErr)
	}
	if fake.createCalled {
		t.Error("空名称不应触达 Repo")
	}
}
