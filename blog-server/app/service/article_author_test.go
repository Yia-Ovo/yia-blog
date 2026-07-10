package service

import (
	"context"
	"testing"

	"blog-server/api/dto"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	domainsvc "blog-server/domain/service"
	"blog-server/infra/model"

	"gorm.io/gorm"
)

// fakeUserRepo 内嵌接口，仅覆盖 FindUserByID（其余方法测试用不到）
type fakeUserRepo struct {
	repo.UserMysqlRepo
	user  *model.UserModel
	calls int
}

func (f *fakeUserRepo) FindUserByID(_ context.Context, _ uint64) (*model.UserModel, error) {
	f.calls++
	if f.user == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.user, nil
}

// fakeAdminRepo 内嵌接口，仅覆盖 FindAdminByID
type fakeAdminRepo struct {
	repo.AdminMysqlRepo
	admin *model.AdminModel
}

func (f *fakeAdminRepo) FindAdminByID(_ context.Context, _ uint64) (*model.AdminModel, error) {
	if f.admin == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.admin, nil
}

func newTestArticleAppService(u repo.UserMysqlRepo, a repo.AdminMysqlRepo) *ArticleAppService {
	// resolveAuthor 只依赖 user/admin 领域服务，文章领域服务与缓存置空即可
	return NewArticleAppService(nil, nil, domainsvc.NewUserDomainService(u), domainsvc.NewAdminDomainService(a))
}

func TestResolveAuthor_User(t *testing.T) {
	uRepo := &fakeUserRepo{user: &model.UserModel{ID: 5, NickName: "张三", Avatar: "http://a/5.png", Signature: "热爱写作"}}
	s := newTestArticleAppService(uRepo, &fakeAdminRepo{})

	got := s.resolveAuthor(context.Background(), 5, 0)
	if got == nil {
		t.Fatal("期望返回用户作者，实际为 nil")
	}
	if got.Type != consts.AuthorTypeUser || got.Name != "张三" || got.Avatar != "http://a/5.png" || got.Signature != "热爱写作" {
		t.Fatalf("用户作者信息映射错误: %+v", got)
	}
}

func TestResolveAuthor_Admin(t *testing.T) {
	aRepo := &fakeAdminRepo{admin: &model.AdminModel{ID: 3, AdminName: "管理员A", Avatar: "http://a/3.png", Remark: "站点维护"}}
	s := newTestArticleAppService(&fakeUserRepo{}, aRepo)

	got := s.resolveAuthor(context.Background(), 0, 3)
	if got == nil {
		t.Fatal("期望返回管理员作者，实际为 nil")
	}
	// 管理员作者：Name 取 AdminName，Signature 取 Remark
	if got.Type != consts.AuthorTypeAdmin || got.Name != "管理员A" || got.Signature != "站点维护" {
		t.Fatalf("管理员作者信息映射错误: %+v", got)
	}
}

func TestResolveAuthor_NoneAndMissing(t *testing.T) {
	s := newTestArticleAppService(&fakeUserRepo{}, &fakeAdminRepo{})

	// 无作者引用（user_id/admin_id 均为 0）→ nil
	if got := s.resolveAuthor(context.Background(), 0, 0); got != nil {
		t.Fatalf("无作者引用时应为 nil，实际 %+v", got)
	}
	// 作者已删除 / 查不到 → nil（不影响文章读取）
	if got := s.resolveAuthor(context.Background(), 99, 0); got != nil {
		t.Fatalf("作者查不到时应为 nil，实际 %+v", got)
	}
}

func TestResolveAuthorMemo_Dedup(t *testing.T) {
	uRepo := &fakeUserRepo{user: &model.UserModel{ID: 5, NickName: "张三"}}
	s := newTestArticleAppService(uRepo, &fakeAdminRepo{})

	memo := make(map[authorRef]*dto.AuthorInfo)
	_ = s.resolveAuthorMemo(context.Background(), 5, 0, memo)
	_ = s.resolveAuthorMemo(context.Background(), 5, 0, memo)

	if uRepo.calls != 1 {
		t.Fatalf("同一作者应只查询一次（memo 去重），实际查询 %d 次", uRepo.calls)
	}
}
