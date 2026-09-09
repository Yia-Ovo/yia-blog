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

	// existing 是 GetCategoryById 的返回值，nil 模拟「分类不存在」（Repo 约定返回 (nil, nil)）
	existing *model.CategoryModel

	// updated 记录最终下发到 Repo 的更新入参，用于断言未提交的字段没有在途中被压平成 0
	updated      *do.CategoryUpdate
	updateCalled bool
	updateErr    error
}

func (f *fakeCategoryMysqlRepo) CreateCategory(_ context.Context, category *model.CategoryModel) error {
	f.createCalled = true
	f.created = category
	return f.createErr
}

func (f *fakeCategoryMysqlRepo) GetCategoryById(_ context.Context, _ uint64) (*model.CategoryModel, error) {
	return f.existing, nil
}

func (f *fakeCategoryMysqlRepo) UpdateCategory(_ context.Context, upd *do.CategoryUpdate) error {
	f.updateCalled = true
	f.updated = upd
	return f.updateErr
}

// TestCreateCategory_PersistsSubmittedFields 钉死严重问题 11 的完整链路：
// 新建 DO 的 ID 为 0，领域层转换后交给 Repo 的 Model 必须携带真实的 Name/Sort/Status，
// 而不是被守卫替换成空 Model（那会让 category 表写入 name=” 的空行且返回成功）。
func TestCreateCategory_PersistsSubmittedFields(t *testing.T) {
	fake := &fakeCategoryMysqlRepo{}
	svc := NewCategoryDomainService(fake)

	status := 1
	err := svc.CreateCategory(context.Background(), &do.CategoryCreate{Name: "tech", Sort: 5, Status: &status})
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

	err := svc.CreateCategory(context.Background(), &do.CategoryCreate{Name: "", Sort: 1})

	if err != common.ErrCategoryNameEmpty {
		t.Errorf("err = %v, 期望 %v", err, common.ErrCategoryNameEmpty)
	}
	if fake.createCalled {
		t.Error("空名称不应触达 Repo")
	}
}

// TestUpdateCategory_OmittedFieldsStayNil 钉死一般问题 20：
// 客户端只提交 {id, name} 改个名字时，Sort/Status 必须一路保持 nil 直到 Repo，
// 由 Repo 把它们排除出 UPDATE 白名单。任何一层把 nil 压平成 0，
// 该分类都会被静默写成 status=0（禁用）+ sort=0——这是必然触发而非偶发。
func TestUpdateCategory_OmittedFieldsStayNil(t *testing.T) {
	fake := &fakeCategoryMysqlRepo{
		existing: &model.CategoryModel{ID: 7, Name: "old", Sort: 5, Status: 1},
	}
	svc := NewCategoryDomainService(fake)

	err := svc.UpdateCategory(context.Background(), &do.CategoryUpdate{ID: 7, Name: "new"})
	if err != nil {
		t.Fatalf("UpdateCategory 返回错误: %v", err)
	}

	if !fake.updateCalled {
		t.Fatal("Repo.UpdateCategory 未被调用")
	}
	if fake.updated == nil {
		t.Fatal("下发到 Repo 的更新入参为 nil")
	}
	if fake.updated.Name != "new" {
		t.Errorf("下发的 Name = %q, 期望 %q", fake.updated.Name, "new")
	}
	if fake.updated.Sort != nil {
		t.Errorf("下发的 Sort = %d, 期望 nil（非 nil 即一般问题 20 复发：只改名字会把排序重置为 0）", *fake.updated.Sort)
	}
	if fake.updated.Status != nil {
		t.Errorf("下发的 Status = %d, 期望 nil（非 nil 即一般问题 20 复发：只改名字会禁用该分类）", *fake.updated.Status)
	}
}

// TestUpdateCategory_ExplicitZeroIsPreserved 与上一条互为镜像：
// 客户端显式提交 sort=0（排最前）/ status=0（禁用）时，指针必须非 nil 且值为 0。
// 这正是不能用「零值即未提交」判断的原因——0 对这两列都是合法业务值，
// 若按空值语义丢弃，「禁用分类」将永远失效。
func TestUpdateCategory_ExplicitZeroIsPreserved(t *testing.T) {
	zero := 0
	fake := &fakeCategoryMysqlRepo{
		existing: &model.CategoryModel{ID: 7, Name: "old", Sort: 5, Status: 1},
	}
	svc := NewCategoryDomainService(fake)

	err := svc.UpdateCategory(context.Background(), &do.CategoryUpdate{
		ID: 7, Name: "new", Sort: &zero, Status: &zero,
	})
	if err != nil {
		t.Fatalf("UpdateCategory 返回错误: %v", err)
	}

	if fake.updated == nil {
		t.Fatal("下发到 Repo 的更新入参为 nil")
	}
	if fake.updated.Sort == nil {
		t.Fatal("下发的 Sort 为 nil，显式提交的 0 被当成「未提交」丢弃了")
	}
	if *fake.updated.Sort != 0 {
		t.Errorf("下发的 Sort = %d, 期望 0", *fake.updated.Sort)
	}
	if fake.updated.Status == nil {
		t.Fatal("下发的 Status 为 nil，显式提交的 0 被当成「未提交」丢弃了——禁用分类将永远失效")
	}
	if *fake.updated.Status != 0 {
		t.Errorf("下发的 Status = %d, 期望 0", *fake.updated.Status)
	}
}

// TestUpdateCategory_NotFound 更新不存在的分类应返回 13001 而非通用 DatabaseErr，
// 且不得触达 Repo 的写入。存在性由领域层显式校验，因为 MySQL 的 RowsAffected==0
// 无法区分「记录不存在」与「值未变化」。
func TestUpdateCategory_NotFound(t *testing.T) {
	fake := &fakeCategoryMysqlRepo{existing: nil} // Repo 约定：查不到返回 (nil, nil)
	svc := NewCategoryDomainService(fake)

	err := svc.UpdateCategory(context.Background(), &do.CategoryUpdate{ID: 7, Name: "new"})

	if err != common.CategoryNotFoundErr {
		t.Errorf("err = %v, 期望 %v", err, common.CategoryNotFoundErr)
	}
	if fake.updateCalled {
		t.Error("不存在的分类不应触达 Repo 的更新")
	}
}

// TestUpdateCategory_RejectsEmptyName 名称为空在更新路径同样应被领域层拦下。
func TestUpdateCategory_RejectsEmptyName(t *testing.T) {
	fake := &fakeCategoryMysqlRepo{
		existing: &model.CategoryModel{ID: 7, Name: "old", Sort: 5, Status: 1},
	}
	svc := NewCategoryDomainService(fake)

	err := svc.UpdateCategory(context.Background(), &do.CategoryUpdate{ID: 7, Name: ""})

	if err != common.ErrCategoryNameEmpty {
		t.Errorf("err = %v, 期望 %v", err, common.ErrCategoryNameEmpty)
	}
	if fake.updateCalled {
		t.Error("空名称不应触达 Repo")
	}
}
