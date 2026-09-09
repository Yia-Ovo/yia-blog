package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"blog-server/common"
	"blog-server/domain/do"
)

func TestUserProfileValidationCannotBeBypassed(t *testing.T) {
	empty, tooLong, badEmail, invalidGender := "", strings.Repeat("名", 17), "invalid", 3
	for _, update := range []*do.UserInfoUpdate{
		nil, {NickName: &empty}, {NickName: &tooLong}, {Email: &badEmail}, {Gender: &invalidGender},
	} {
		err := NewUserDomainService(nil).UpdateUserInfo(context.Background(), 7, update)
		var errno common.Errno
		if !errors.As(err, &errno) || errno.Code != common.ParamErr.Code {
			t.Fatalf("invalid profile reached persistence: update=%+v err=%v", update, err)
		}
	}
}
