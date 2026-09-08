package service

import (
	"context"
	"testing"

	"blog-server/common"
)

func TestArticleUpdateRejectsNilBeforeRepository(t *testing.T) {
	s := NewArticleDomainService(nil)
	if err := s.UpdateArticle(context.Background(), nil); err != common.ParamErr {
		t.Fatalf("admin update: %v", err)
	}
	if err := s.UpdateArticleByUser(context.Background(), 1, nil); err != common.ParamErr {
		t.Fatalf("user update: %v", err)
	}
}
