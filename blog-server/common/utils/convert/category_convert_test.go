package convert

import (
	"testing"
	"time"

	"blog-server/domain/do"
	"blog-server/infra/model"
)

// TestCategoryDOToModel_NewCategoryKeepsFields 钉死严重问题 11：
// 新建分类的 DO 必然 ID==0（主键待自增），转换必须完整保留 Name/Sort/Status，
// 不能因为 ID 为零值就被当成无效 DO 而返回空 Model —— 那会让 Create 写入 name=” 空行。
func TestCategoryDOToModel_NewCategoryKeepsFields(t *testing.T) {
	d := &do.CategoryDO{Name: "tech", Sort: 3, Status: 0}

	m := CategoryDOToModel(d)

	if m == nil {
		t.Fatal("CategoryDOToModel(新建DO) 返回 nil，创建路径将无数据可写")
	}
	if m.Name != "tech" {
		t.Errorf("Name = %q, 期望 %q", m.Name, "tech")
	}
	if m.Sort != 3 {
		t.Errorf("Sort = %d, 期望 %d", m.Sort, 3)
	}
	// Status=0（禁用）是客户端可提交的合法值，必须原样透传给持久化层
	if m.Status != 0 {
		t.Errorf("Status = %d, 期望 %d", m.Status, 0)
	}
	// ID 保持 0，交给数据库自增；显式断言避免日后误填占位主键
	if m.ID != 0 {
		t.Errorf("ID = %d, 新建时期望为 0（由数据库自增）", m.ID)
	}
	if m.DeletedAt.Valid {
		t.Error("新建分类不应带软删除标记")
	}
}

// TestCategoryDOToModel_Nil 约定：仅 nil 输入返回 nil，与 CategoryModelToDO 对称，由调用方判空。
func TestCategoryDOToModel_Nil(t *testing.T) {
	if m := CategoryDOToModel(nil); m != nil {
		t.Errorf("CategoryDOToModel(nil) = %+v, 期望 nil", m)
	}
}

// TestCategoryDOToModel_ExistingCategory 更新路径：ID 与时间字段原样带出，IsDeleted 映射为 DeletedAt。
func TestCategoryDOToModel_ExistingCategory(t *testing.T) {
	createAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	d := &do.CategoryDO{ID: 7, Name: "life", Sort: 1, Status: 1, CreateAt: createAt, IsDeleted: true}

	m := CategoryDOToModel(d)

	if m == nil {
		t.Fatal("CategoryDOToModel(已有分类) 返回 nil")
	}
	if m.ID != 7 || m.Name != "life" {
		t.Errorf("ID/Name = %d/%q, 期望 7/%q", m.ID, m.Name, "life")
	}
	if !m.CreateAt.Equal(createAt) {
		t.Errorf("CreateAt = %v, 期望 %v", m.CreateAt, createAt)
	}
	if !m.DeletedAt.Valid {
		t.Error("IsDeleted=true 时 DeletedAt 应为有效值")
	}
}

// TestCategoryModelListToDOList_SkipsNil 覆盖优化建议 14：
// CategoryModelToDO 对 nil / ID==0 元素返回 nil，列表转换必须跳过而非解引用。
func TestCategoryModelListToDOList_SkipsNil(t *testing.T) {
	list := []*model.CategoryModel{
		{ID: 1, Name: "tech"},
		// 防御：Repo 未来若产生 nil 元素不应 panic
		nil,
		// ID==0 被 CategoryModelToDO 判为无效，同样跳过
		{ID: 0},
		{ID: 2, Name: "life"},
	}

	res := CategoryModelListToDOList(list)

	if len(res) != 2 {
		t.Fatalf("len = %d, 期望 2（nil 与 ID==0 的元素应被跳过）", len(res))
	}
	if res[0].Name != "tech" || res[1].Name != "life" {
		t.Errorf("结果顺序/内容异常: %+v", res)
	}
}
