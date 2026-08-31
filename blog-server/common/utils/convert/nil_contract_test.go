package convert

import (
	"testing"

	"blog-server/domain/do"
	"blog-server/infra/model"
)

// TestDOToModel_NilContract 钉死 convert 层统一的 nil 契约：
// 所有 *DOToModel 对 nil 入参必须返回 nil，由调用方在第一次解引用前判空。
//
// 历史背景：UserDOToModel 曾对 nil 返回 &model.UserModel{}（空对象），
// AdminDOToModel / CommentDOToModel 则完全没有守卫（传 nil 直接 panic）。
// 「返回空对象」是严重问题 11 的传播机制——它让无效输入伪装成合法对象一路传到
// Create/Updates，静默写入空行而不报错；返回 nil 才能让错误尽早暴露。
//
// 用闭包逐个断言而非把结果装进 interface{}：*T 类型的 nil 装箱后 != nil
// （Go 的 typed nil 陷阱），装箱比较会让本测试永远通过。
func TestDOToModel_NilContract(t *testing.T) {
	cases := []struct {
		name       string
		returnsNil func() bool
	}{
		{"UserDOToModel", func() bool { return UserDOToModel(nil) == nil }},
		{"AdminDOToModel", func() bool { return AdminDOToModel(nil) == nil }},
		{"CommentDOToModel", func() bool { return CommentDOToModel(nil) == nil }},
		{"ArticleDOToModel", func() bool { return ArticleDOToModel(nil) == nil }},
		{"CategoryDOToModel", func() bool { return CategoryDOToModel(nil) == nil }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !c.returnsNil() {
				t.Errorf("%s(nil) 未返回 nil；返回空对象会让无效输入伪装成合法 Model 流向持久化层", c.name)
			}
		})
	}
}

// TestDOToModel_NonNilInputKeepsModel 反向断言：合法入参必须拿到非 nil Model。
// 防止「统一 nil 契约」被误写成无条件返回 nil，那会让所有写入路径静默失效。
func TestDOToModel_NonNilInputKeepsModel(t *testing.T) {
	if m := UserDOToModel(&do.UserDO{NickName: "yia"}); m == nil || m.NickName != "yia" {
		t.Errorf("UserDOToModel 丢失字段: %+v", m)
	}
	if m := AdminDOToModel(&do.AdminDO{AdminName: "root"}); m == nil || m.AdminName != "root" {
		t.Errorf("AdminDOToModel 丢失字段: %+v", m)
	}
	if m := CommentDOToModel(&do.CommentDO{Nickname: "tom"}); m == nil || m.Nickname != "tom" {
		t.Errorf("CommentDOToModel 丢失字段: %+v", m)
	}
	if m := ArticleDOToModel(&do.ArticleDO{Title: "go"}); m == nil || m.Title != "go" {
		t.Errorf("ArticleDOToModel 丢失字段: %+v", m)
	}
}

// TestArticleModelListToDOList_SkipsNil 列表转换必须跳过 nil 元素而非无条件解引用。
// 与 CategoryModelListToDOList 的处理保持一致（见优化建议 14）。
func TestArticleModelListToDOList_SkipsNil(t *testing.T) {
	list := []*model.ArticleModel{
		{ID: 1, Title: "go"},
		// 防御：Repo 未来若产生 nil 元素不应 panic
		nil,
		{ID: 2, Title: "gorm"},
	}

	res := ArticleModelListToDOList(list)

	if len(res) != 2 {
		t.Fatalf("len = %d, 期望 2（nil 元素应被跳过）", len(res))
	}
	if res[0].Title != "go" || res[1].Title != "gorm" {
		t.Errorf("结果顺序/内容异常: %+v", res)
	}
}
