package service

import (
	"context"
	"testing"

	"blog-server/api/dto"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
)

type categoryPageRepo struct {
	repo.CategoryMysqlRepo
	query do.CategoryQuery
}

func (r *categoryPageRepo) ListCategoryByPage(_ context.Context, query *do.CategoryQuery) ([]*model.CategoryModel, int64, error) {
	r.query = *query
	return nil, 11, nil
}

func TestCategoryPaginationUsesEffectiveQuery(t *testing.T) {
	for _, size := range []int{0, -1, 51} {
		r := &categoryPageRepo{}
		s := NewCategoryAppService(domainservice.NewCategoryDomainService(r), nil)
		page, err := s.ListCategoryByPage(context.Background(), &dto.CategoryPageQuery{PageSize: size})
		if err != nil {
			t.Fatal(err)
		}
		if page.PageNum != 1 || page.PageSize != 10 || page.Pages != 2 || page.PageNum != r.query.PageNum || page.PageSize != r.query.PageSize {
			t.Fatalf("response/query mismatch for size %d: response=%+v query=%+v", size, page, r.query)
		}
	}
}
