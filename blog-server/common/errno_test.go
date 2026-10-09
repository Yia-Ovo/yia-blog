package common

import (
	"errors"
	"strings"
	"testing"
)

func TestErrnoSeparatesPublicMessageAndPrivateDetail(t *testing.T) {
	base := ParamErr
	detail := errors.New("private validator detail")
	for _, got := range []Errno{
		base.WithMsg("请检查输入").WithErr(detail),
		base.WithErr(detail).WithMsg("请检查输入"),
	} {
		if got.Code != base.Code || got.Msg != "请检查输入" || got.Error() != "请检查输入" || !strings.Contains(got.ErrMsg, detail.Error()) {
			t.Fatalf("unexpected error: %#v", got)
		}
	}
	if ParamErr != base {
		t.Fatal("shared error was mutated")
	}
	first := base.WithMsg("第一次")
	second := base.WithMsg("第二次")
	if first.Msg != "第一次" || second.Msg != "第二次" || first.ErrMsg != "" || second.ErrMsg != "" {
		t.Fatal("messages must remain independent")
	}
}
