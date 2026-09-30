package service

import (
	"blog-server/domain/repo"
	"context"
	"reflect"
	"testing"
	"time"
)

type artworkRepoStub struct {
	ids []string
	err error
}

func (r *artworkRepoStub) List(context.Context) ([]string, error) { return r.ids, r.err }
func (r *artworkRepoStub) Read(context.Context, string) ([]byte, error) {
	return nil, repo.ErrAppearanceImageNotFound
}

func TestDailyAppearanceMidnightAndStableCycle(t *testing.T) {
	ids := []string{"c.jpg", "a.jpg", "b.jpg"}
	service := NewAppearanceDomainService(&artworkRepoStub{ids: ids})
	ctx := context.Background()
	before := time.Date(2026, 9, 29, 15, 59, 59, 0, time.UTC)
	first, err := service.Today(ctx, before)
	if err != nil {
		t.Fatal(err)
	}
	if first.Date != "2026-09-29" || first.NextChangeAt.UTC() != before.Add(time.Second) {
		t.Fatalf("wrong boundary: %+v", first)
	}
	same, _ := service.Today(ctx, before.Add(-12*time.Hour))
	if same.ImageID != first.ImageID {
		t.Fatal("image changed within same Beijing day")
	}
	next, _ := service.Today(ctx, before.Add(time.Second))
	if next.Date != "2026-09-30" || next.ImageID == first.ImageID {
		t.Fatalf("did not rotate: %+v", next)
	}
	cycle, _ := service.Today(ctx, before.AddDate(0, 0, 3))
	if cycle.ImageID != first.ImageID {
		t.Fatal("cycle did not repeat after all images")
	}
	if !reflect.DeepEqual(ids, []string{"c.jpg", "a.jpg", "b.jpg"}) {
		t.Fatal("modified repository-owned list")
	}
	otherZone, _ := service.Today(ctx, before.In(time.FixedZone("visitor", -7*60*60)))
	if otherZone.ImageID != first.ImageID || otherZone.Date != first.Date {
		t.Fatal("caller timezone affected image")
	}
}

func TestDailyAppearanceLeapDayAndEmptyCatalog(t *testing.T) {
	service := NewAppearanceDomainService(&artworkRepoStub{ids: []string{"only.jpg"}})
	value, err := service.Today(context.Background(), time.Date(2028, 2, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if value.NextChangeAt.Format(time.RFC3339) != "2028-03-01T00:00:00+08:00" {
		t.Fatal(value.NextChangeAt)
	}
	empty := NewAppearanceDomainService(&artworkRepoStub{})
	if _, err := empty.Today(context.Background(), time.Now()); err == nil {
		t.Fatal("empty catalog was accepted")
	}
	failed := NewAppearanceDomainService(&artworkRepoStub{err: context.Canceled})
	if _, err := failed.Today(context.Background(), time.Now()); err != context.Canceled {
		t.Fatal("repository error was lost")
	}
}
