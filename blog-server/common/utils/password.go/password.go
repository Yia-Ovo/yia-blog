package password

import (
	"errors"
	"regexp"
)

// CheckPasswordFormat 密码格式校验
func CheckPasswordFormat(newPwd string) error {
	// 长度校验
	if len(newPwd) < 8 || len(newPwd) > 20 {
		return errors.New("密码长度必须为8-20位")
	}
	// 复杂度校验
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(newPwd)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(newPwd)
	if !hasLetter || !hasNumber {
		return errors.New("密码必须包含字母和数字")
	}
	return nil
}
