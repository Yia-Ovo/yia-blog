package repo

import (
	"context"
	"errors"
	"testing"

	domainrepo "blog-server/domain/repo"
)

func TestConcurrentCommentLikesPersistOncePerUser(t *testing.T) {
	for _, sameUser := range []bool{true, false} {
		r, state := newLikeSQLRepo(t)
		state.likes = 10 // 保留旧聚合数，新增关系只增加本次贡献。
		type result struct {
			changed bool
			err     error
		}
		results := make(chan result, 32)
		for i := 0; i < 32; i++ {
			userID := uint64(i + 1)
			if sameUser {
				userID = 7
			}
			go func(userID uint64) {
				changed, err := r.LikeComment(context.Background(), userID, 11)
				results <- result{changed, err}
			}(userID)
		}
		changed := 0
		for i := 0; i < 32; i++ {
			got := <-results
			if got.err != nil {
				t.Fatal(got.err)
			}
			if got.changed {
				changed++
			}
		}
		want := 32
		if sameUser {
			want = 1
		}
		if changed != want || state.likes != 10+int64(want) || len(state.records) != want {
			t.Fatalf("likes repeated/lost: changed=%d likes=%d records=%d", changed, state.likes, len(state.records))
		}
	}
}

func TestCommentLikeCounterFailureRollsBackRelationship(t *testing.T) {
	r, state := newLikeSQLRepo(t)
	state.failCounter = true
	if changed, err := r.LikeComment(context.Background(), 7, 11); err == nil || changed || len(state.records) != 0 || state.likes != 0 || state.rollbacks != 1 {
		t.Fatalf("partial write survived: changed=%v err=%v state=%+v", changed, err, state)
	}
	state.failCounter = false
	if changed, err := r.LikeComment(context.Background(), 7, 11); err != nil || !changed || state.likes != 1 || len(state.records) != 1 {
		t.Fatalf("retry failed: changed=%v err=%v", changed, err)
	}
}

func TestCommentUnlikeOnlyRemovesOwnRelationAndRollsBackOnFailure(t *testing.T) {
	r, state := newLikeSQLRepo(t)
	if _, err := r.LikeComment(context.Background(), 7, 11); err != nil {
		t.Fatal(err)
	}
	if changed, err := r.DisLikeComment(context.Background(), 8, 11); changed || err != nil || state.likes != 1 {
		t.Fatalf("other user changed count: changed=%v err=%v likes=%d", changed, err, state.likes)
	}
	state.failCounter = true
	if changed, err := r.DisLikeComment(context.Background(), 7, 11); changed || err == nil || state.likes != 1 || len(state.records) != 1 {
		t.Fatalf("failed unlike was partially committed: changed=%v err=%v", changed, err)
	}
	state.failCounter = false
	if changed, err := r.DisLikeComment(context.Background(), 7, 11); !changed || err != nil || state.likes != 0 || len(state.records) != 0 {
		t.Fatalf("unlike retry failed: changed=%v err=%v", changed, err)
	}
	if changed, err := r.DisLikeComment(context.Background(), 7, 11); changed || err != nil || state.likes != 0 {
		t.Fatalf("unlike repeated: changed=%v err=%v", changed, err)
	}
	state.records[[2]uint64{11, 7}] = true
	if _, err := r.DisLikeComment(context.Background(), 7, 11); err != nil || state.likes != 0 {
		t.Fatalf("count became negative: %d err=%v", state.likes, err)
	}
}

func TestCommentLikeRetryAfterLostCommitReplyDoesNotIncrementAgain(t *testing.T) {
	r, state := newLikeSQLRepo(t)
	state.loseCommitReply = true
	if changed, err := r.LikeComment(context.Background(), 7, 11); changed || err == nil {
		t.Fatalf("lost acknowledgement reported as success: %v %v", changed, err)
	}
	if state.likes != 1 || len(state.records) != 1 {
		t.Fatal("fixture did not model an applied commit")
	}
	if changed, err := r.LikeComment(context.Background(), 7, 11); changed || err != nil || state.likes != 1 || len(state.records) != 1 {
		t.Fatalf("retry duplicated committed like: changed=%v err=%v likes=%d", changed, err, state.likes)
	}
}

func TestCommentLikeSQLRejectsInvisibleOrMissingTarget(t *testing.T) {
	for _, missing := range []bool{true, false} {
		r, state := newLikeSQLRepo(t)
		state.present, state.public = !missing, false
		if changed, err := r.LikeComment(context.Background(), 7, 11); changed || !errors.Is(err, domainrepo.ErrNoRowsAffected) || len(state.records) != 0 {
			t.Fatalf("invisible target accepted: changed=%v err=%v", changed, err)
		}
	}
}
