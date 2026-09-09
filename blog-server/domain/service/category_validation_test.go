package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"blog-server/common"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
)

func TestCategoryValidationUsesBusinessErrors(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, tc := range []struct {
			name   string
			value  string
			sort   int
			status int
			code   int
		}{
			{"empty name", "", 0, 1, common.ErrCategoryNameEmpty.Code},
			{"long name", strings.Repeat("类", 65), 0, 1, common.ErrCategoryNameTooLong.Code},
			{"negative sort", "tech", -1, 1, common.ErrCategorySortInvalid.Code},
			{"invalid status", "tech", 0, 2, common.ParamErr.Code},
			{"64 unicode characters", strings.Repeat("类", 64), 0, 0, 0},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				r := &fakeCategoryMysqlRepo{existing: &model.CategoryModel{ID: 1}}
				s := NewCategoryDomainService(r)
				var err error
				if operation == "create" {
					err = s.CreateCategory(context.Background(), &do.CategoryCreate{Name: tc.value, Sort: tc.sort, Status: &tc.status})
				} else {
					err = s.UpdateCategory(context.Background(), &do.CategoryUpdate{ID: 1, Name: tc.value, Sort: &tc.sort, Status: &tc.status})
				}
				if tc.code == 0 {
					if err != nil {
						t.Fatalf("valid category rejected: %v", err)
					}
					return
				}
				var errno common.Errno
				if !errors.As(err, &errno) || errno.Code != tc.code {
					t.Fatalf("error = %v, want code %d", err, tc.code)
				}
				if r.createCalled || r.updateCalled {
					t.Fatal("invalid category reached a write")
				}
			})
		}
	}
}

func TestCategoryDuplicateNameIsNotDatabaseFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{"duplicate", fmt.Errorf("insert: %w", repo.ErrDuplicateKey), common.CategoryExistErr},
		{"database failure", errors.New("database unavailable"), common.DatabaseErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &fakeCategoryMysqlRepo{
				existing: &model.CategoryModel{ID: 1}, createErr: tc.err, updateErr: tc.err,
			}
			s := NewCategoryDomainService(r)
			if err := s.CreateCategory(context.Background(), &do.CategoryCreate{Name: "tech"}); !errors.Is(err, tc.want) {
				t.Fatalf("create returned %v, want %v", err, tc.want)
			}
			if err := s.UpdateCategory(context.Background(), &do.CategoryUpdate{ID: 1, Name: "tech"}); !errors.Is(err, tc.want) {
				t.Fatalf("update returned %v, want %v", err, tc.want)
			}
		})
	}
}
