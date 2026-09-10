package service

import (
	"context"
	"errors"
	"testing"

	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
)

type moderationTestRepo struct {
	repo.CommentMysqlRepo
	comment     *model.CommentModel
	query       *do.CommentQuery
	writes      int
	err         error
	beforeWrite func()
}

func (r *moderationTestRepo) GetCommentByID(context.Context, uint64) (*model.CommentModel, error) {
	if r.comment == nil {
		return nil, nil
	}
	copy := *r.comment
	return &copy, nil
}

func (r *moderationTestRepo) ListCommentByPage(_ context.Context, q *do.CommentQuery) ([]*model.CommentModel, int64, error) {
	copy := *q
	r.query = &copy
	return nil, 0, nil
}

func (r *moderationTestRepo) ReviewComment(_ context.Context, review *do.CommentReview) (int64, error) {
	r.writes++
	if r.err != nil {
		return 0, r.err
	}
	if r.beforeWrite != nil {
		r.beforeWrite()
	}
	if r.comment == nil || r.comment.Content != review.ExpectedContent || r.comment.Status != review.ExpectedStatus {
		return 0, nil
	}
	r.comment.Status = review.Status
	return 1, nil
}

func moderationTestService(r *moderationTestRepo, admin *model.AdminModel) *CommentModerationDomainService {
	return NewCommentModerationDomainService(NewCommentDomainService(r, nil), NewAdminDomainService(&contentTestAdmins{admin: admin}), r)
}

func TestCommentModerationTransitionsAndRetry(t *testing.T) {
	admin := &model.AdminModel{ID: 9, Status: consts.AdminStatusEnabled, Role: consts.AdminRoleNormal}
	for _, states := range [][2]int{{0, 1}, {0, 2}, {1, 2}, {2, 1}} {
		r := &moderationTestRepo{comment: &model.CommentModel{ID: 3, ArticleID: 11, Content: "reviewed", Status: states[0], Likes: 4, UserID: 7}}
		s := moderationTestService(r, admin)
		q := &do.CommentReview{CommentID: 3, ExpectedContent: "reviewed", ExpectedStatus: states[0], Status: states[1]}
		for attempt := 0; attempt < 2; attempt++ {
			articleID, err := s.ReviewComment(context.Background(), 9, q)
			if err != nil || articleID != 11 || r.comment.Status != states[1] || r.writes != 1 || r.comment.Content != "reviewed" || r.comment.Likes != 4 || r.comment.UserID != 7 {
				t.Fatalf("transition/retry failed %v: id=%d err=%v row=%+v writes=%d", states, articleID, err, r.comment, r.writes)
			}
		}
	}
}

func TestCommentModerationRejectsStaleOrUnauthorizedDecisions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		adminID  uint64
		admin    *model.AdminModel
		content  string
		expected int
		target   int
		want     error
	}{
		{"anonymous", 0, nil, "old", 0, 1, common.LoginNotLoginErr},
		{"missing admin", 9, nil, "old", 0, 1, common.AdminNotFoundErr},
		{"disabled", 9, &model.AdminModel{ID: 9, Status: 0, Role: 1}, "old", 0, 1, common.AdminAccountDisabledErr},
		{"invalid role", 9, &model.AdminModel{ID: 9, Status: 1, Role: 8}, "old", 0, 1, common.PermissionDeniedErr},
		{"changed body", 9, &model.AdminModel{ID: 9, Status: 1, Role: 1}, "NEW", 0, 1, common.ParamErr},
		{"changed state", 9, &model.AdminModel{ID: 9, Status: 1, Role: 1}, "old", 2, 1, common.ParamErr},
		{"invalid target", 9, &model.AdminModel{ID: 9, Status: 1, Role: 1}, "old", 0, 0, common.ParamErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &moderationTestRepo{comment: &model.CommentModel{ID: 3, ArticleID: 11, Content: "old", Status: 0}}
			_, err := moderationTestService(r, tc.admin).ReviewComment(context.Background(), tc.adminID, &do.CommentReview{CommentID: 3, ExpectedContent: tc.content, ExpectedStatus: tc.expected, Status: tc.target})
			if !contentErrorMatches(err, tc.want) || r.writes != 0 {
				t.Fatalf("invalid review wrote data: err=%v writes=%d", err, r.writes)
			}
		})
	}
}

func TestCommentModerationConditionalWriteHandlesRaces(t *testing.T) {
	admin := &model.AdminModel{ID: 9, Status: 1, Role: 0}
	for _, name := range []string{"edit", "delete", "same decision", "database failure"} {
		t.Run(name, func(t *testing.T) {
			r := &moderationTestRepo{comment: &model.CommentModel{ID: 3, ArticleID: 11, Content: "old", Status: 0}}
			want := error(common.ParamErr)
			switch name {
			case "edit":
				r.beforeWrite = func() { r.comment.Content = "Old" }
			case "delete":
				r.beforeWrite = func() { r.comment = nil }
				want = common.CommentNotFoundErr
			case "same decision":
				r.beforeWrite = func() { r.comment.Status = consts.CommentStatusPassed }
				want = nil
			case "database failure":
				r.err = errors.New("offline")
				want = common.DatabaseErr
			}
			_, err := moderationTestService(r, admin).ReviewComment(context.Background(), 9, &do.CommentReview{CommentID: 3, ExpectedContent: "old", ExpectedStatus: 0, Status: 1})
			if !contentErrorMatches(err, want) {
				t.Fatalf("unexpected race result: %v, want %v", err, want)
			}
			if name == "edit" && r.comment.Status != consts.CommentStatusPending {
				t.Fatal("unreviewed new content was approved")
			}
		})
	}
}

func TestAdminCommentListChecksRoleAndKeepsPendingFilter(t *testing.T) {
	r := &moderationTestRepo{}
	s := moderationTestService(r, &model.AdminModel{ID: 9, Status: 1, Role: 1})
	pending := consts.CommentStatusPending
	q := &do.CommentQuery{Status: &pending, WithDeleted: true, PublishedArticleOnly: true}
	if _, _, err := s.ListAdminComments(context.Background(), 9, q); err != nil || r.query.Status == nil || *r.query.Status != 0 || r.query.WithDeleted || r.query.PublishedArticleOnly || q.Page != 1 || q.Size != 10 {
		t.Fatalf("admin pending filter changed: %+v %v", r.query, err)
	}
	r.query = nil
	if _, _, err := s.ListAdminComments(context.Background(), 0, &do.CommentQuery{}); !contentErrorMatches(err, common.LoginNotLoginErr) || r.query != nil {
		t.Fatalf("anonymous list permitted: %v", err)
	}
	if _, _, err := s.ListAdminComments(context.Background(), 9, nil); !contentErrorMatches(err, common.ParamErr) {
		t.Fatalf("nil query = %v", err)
	}
}
