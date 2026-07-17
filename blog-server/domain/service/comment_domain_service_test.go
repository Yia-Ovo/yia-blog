package service

import (
	"context"
	"strings"
	"testing"

	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
)

type fakeCommentMysqlRepo struct {
	repo.CommentMysqlRepo

	comment  *model.CommentModel
	comments map[uint64]*model.CommentModel

	created *model.CommentModel

	updateCalled  bool
	updateUserID  uint64
	updateContent string
	updateStatus  int
	updateRows    int64
	updateErr     error

	deleteCalled bool
	deleteUserID uint64
	deleteRows   int64
	deleteErr    error
}

func (f *fakeCommentMysqlRepo) CreateComment(_ context.Context, comment *model.CommentModel) error {
	copy := *comment
	f.created = &copy
	return nil
}

func (f *fakeCommentMysqlRepo) GetCommentByID(_ context.Context, commentID uint64) (*model.CommentModel, error) {
	if f.comments != nil {
		return f.comments[commentID], nil
	}
	return f.comment, nil
}

func (f *fakeCommentMysqlRepo) UpdateCommentByUser(
	_ context.Context,
	_ uint64,
	userID uint64,
	content string,
	status int,
) (int64, error) {
	f.updateCalled = true
	f.updateUserID = userID
	f.updateContent = content
	f.updateStatus = status
	return f.updateRows, f.updateErr
}

func (f *fakeCommentMysqlRepo) DeleteCommentByUser(
	_ context.Context,
	_ uint64,
	userID uint64,
) (int64, error) {
	f.deleteCalled = true
	f.deleteUserID = userID
	return f.deleteRows, f.deleteErr
}

func assertErrCode(t *testing.T, err error, want common.Errno) {
	t.Helper()
	got, ok := err.(common.Errno)
	if !ok {
		t.Fatalf("期望 common.Errno，实际 %T: %v", err, err)
	}
	if got.Code != want.Code {
		t.Fatalf("错误码不匹配：got=%d want=%d", got.Code, want.Code)
	}
}

type fakeCommentArticleRepo struct {
	repo.ArticleMysqlRepo
	missing bool
	err     error
}

func (f *fakeCommentArticleRepo) GetArticleById(_ context.Context, articleID uint64) (*model.ArticleModel, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.missing {
		return nil, nil
	}
	return &model.ArticleModel{ID: articleID}, nil
}

func newTestCommentDomainService(commentRepo repo.CommentMysqlRepo) *CommentDomainService {
	return NewCommentDomainService(commentRepo, &fakeCommentArticleRepo{})
}

func TestCreateCommentForcesServerManagedFields(t *testing.T) {
	fake := &fakeCommentMysqlRepo{}
	srv := newTestCommentDomainService(fake)

	err := srv.CreateComment(context.Background(), commentModelToDOForTest(&model.CommentModel{
		UserID:    7,
		AdminID:   99,
		ArticleID: 10,
		Nickname:  "user",
		Content:   "hello",
		Status:    consts.CommentStatusPassed,
		Likes:     88,
	}))
	if err != nil {
		t.Fatalf("创建评论失败: %v", err)
	}
	if fake.created == nil {
		t.Fatal("期望调用 CreateComment")
	}
	if fake.created.UserID != 7 || fake.created.AdminID != 0 {
		t.Fatalf("作者字段错误: user_id=%d admin_id=%d", fake.created.UserID, fake.created.AdminID)
	}
	if fake.created.Status != consts.CommentStatusPending || fake.created.Likes != 0 {
		t.Fatalf("服务端字段未重置: status=%d likes=%d", fake.created.Status, fake.created.Likes)
	}
	if fake.created.CreateAt.IsZero() || fake.created.UpdateAt.IsZero() {
		t.Fatal("创建和更新时间应由 DomainService 设置")
	}
}

func TestUpdateCommentByOwnerResetsStatus(t *testing.T) {
	fake := &fakeCommentMysqlRepo{
		comment:    &model.CommentModel{ID: 3, UserID: 7, ArticleID: 11, Content: "old", Status: consts.CommentStatusPassed},
		updateRows: 1,
	}
	srv := newTestCommentDomainService(fake)

	articleID, err := srv.UpdateComment(context.Background(), 7, 3, "new")
	if err != nil {
		t.Fatalf("更新评论失败: %v", err)
	}
	if articleID != 11 {
		t.Fatalf("返回文章ID错误: %d", articleID)
	}
	if !fake.updateCalled || fake.updateUserID != 7 || fake.updateContent != "new" {
		t.Fatalf("更新参数错误: %+v", fake)
	}
	if fake.updateStatus != consts.CommentStatusPending {
		t.Fatalf("编辑后应重置为待审核，实际 status=%d", fake.updateStatus)
	}
}

func TestUpdateCommentRejectsOtherUser(t *testing.T) {
	fake := &fakeCommentMysqlRepo{
		comment: &model.CommentModel{ID: 3, UserID: 7, ArticleID: 11, Content: "old"},
	}
	srv := newTestCommentDomainService(fake)

	_, err := srv.UpdateComment(context.Background(), 8, 3, "new")
	assertErrCode(t, err, common.PermissionDeniedErr)
	if fake.updateCalled {
		t.Fatal("越权更新不应调用 Repo")
	}
}

func TestDeleteCommentChecksOwnership(t *testing.T) {
	fake := &fakeCommentMysqlRepo{
		comment:    &model.CommentModel{ID: 3, UserID: 7, ArticleID: 11},
		deleteRows: 1,
	}
	srv := newTestCommentDomainService(fake)

	if _, err := srv.DeleteComment(context.Background(), 8, 3); err == nil {
		t.Fatal("其他用户删除评论应被拒绝")
	} else {
		assertErrCode(t, err, common.PermissionDeniedErr)
	}
	if fake.deleteCalled {
		t.Fatal("越权删除不应调用 Repo")
	}

	articleID, err := srv.DeleteComment(context.Background(), 7, 3)
	if err != nil {
		t.Fatalf("作者删除评论失败: %v", err)
	}
	if articleID != 11 || !fake.deleteCalled || fake.deleteUserID != 7 {
		t.Fatalf("删除结果错误: article_id=%d fake=%+v", articleID, fake)
	}
}

func TestGetCommentByIDReturnsNotFound(t *testing.T) {
	srv := newTestCommentDomainService(&fakeCommentMysqlRepo{})

	_, err := srv.GetCommentByID(context.Background(), 99)
	assertErrCode(t, err, common.CommentNotFoundErr)
}

func TestCreateCommentRejectsMissingArticle(t *testing.T) {
	commentRepo := &fakeCommentMysqlRepo{}
	srv := NewCommentDomainService(commentRepo, &fakeCommentArticleRepo{missing: true})

	err := srv.CreateComment(context.Background(), &do.CommentDO{
		UserID:    7,
		ArticleID: 99,
		Nickname:  "user",
		Content:   "hello",
	})
	assertErrCode(t, err, common.ArticleNotFoundErr)
	if commentRepo.created != nil {
		t.Fatal("文章不存在时不应创建评论")
	}
}

func TestCreateCommentValidatesParentArticle(t *testing.T) {
	tests := []struct {
		name    string
		parent  *model.CommentModel
		wantErr common.Errno
	}{
		{name: "父评论不存在", wantErr: common.CommentNotFoundErr},
		{name: "父评论属于其他文章", parent: &model.CommentModel{ID: 5, ArticleID: 20}, wantErr: common.ParamErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRepo := &fakeCommentMysqlRepo{comments: map[uint64]*model.CommentModel{5: tt.parent}}
			srv := newTestCommentDomainService(commentRepo)

			err := srv.CreateComment(context.Background(), &do.CommentDO{
				UserID:    7,
				ArticleID: 10,
				ParentID:  5,
				Nickname:  "user",
				Content:   "hello",
			})
			assertErrCode(t, err, tt.wantErr)
			if commentRepo.created != nil {
				t.Fatal("父评论关系非法时不应创建评论")
			}
		})
	}
}

func TestCommentContentValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr common.Errno
	}{
		{name: "纯空白内容", content: "   ", wantErr: common.ErrCommentContentEmpty},
		{name: "超过长度上限", content: strings.Repeat("评", maxCommentContentLength+1), wantErr: common.ParamErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRepo := &fakeCommentMysqlRepo{}
			srv := newTestCommentDomainService(commentRepo)

			err := srv.CreateComment(context.Background(), &do.CommentDO{
				UserID:    7,
				ArticleID: 10,
				Nickname:  "user",
				Content:   tt.content,
			})
			assertErrCode(t, err, tt.wantErr)
			if commentRepo.created != nil {
				t.Fatal("内容非法时不应创建评论")
			}
		})
	}
}

func commentModelToDOForTest(comment *model.CommentModel) *do.CommentDO {
	return &do.CommentDO{
		ID:        comment.ID,
		UserID:    comment.UserID,
		ArticleID: comment.ArticleID,
		ParentID:  comment.ParentID,
		AdminID:   comment.AdminID,
		Nickname:  comment.Nickname,
		Email:     comment.Email,
		Avatar:    comment.Avatar,
		Content:   comment.Content,
		IP:        comment.IP,
		UserAgent: comment.UserAgent,
		Status:    comment.Status,
		Likes:     comment.Likes,
	}
}
