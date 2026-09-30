package convert

import (
	"blog-server/infra/model"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestUserModelToDOPreservesDeletionForDomainRules(t *testing.T) {
	deletedAt := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	for _, valid := range []bool{false, true} {
		model := &model.UserModel{ID: 7, DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: valid}}
		got := UserModelToDO(model)
		if got == nil || got.IsDeleted != valid {
			t.Fatalf("soft-delete state lost: model=%+v do=%+v", model.DeletedAt, got)
		}
	}
}
