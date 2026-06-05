/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-03-30 18:08:41
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-05 17:28:07
 * @FilePath: \yia-blog\blog-server\common\utils\password\password.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package password

import "regexp"

// CheckPasswordFormat 统一密码校验
// oldPwd 旧密码（注册时传空串）
// newPwd 新密码
// return 错误信息，为空则表示校验通过
func CheckPasswordFormat(oldPwd, newPwd string) string {
	// 1. 密码为空 → 直接返回正确提示
	if newPwd == "" {
		return "密码不能为空"
	}

	// 2. 长度校验
	if len(newPwd) < 6 || len(newPwd) > 20 {
		return "密码长度必须为 6-20 位"
	}

	// 3. 必须包含字母 + 数字
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(newPwd)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(newPwd)
	if !hasLetter || !hasNumber {
		return "密码必须同时包含英文字母与数字"
	}

	// 4. 新旧密码不能相同
	if oldPwd != "" && oldPwd == newPwd {
		return "新密码不能与原密码相同"
	}

	return ""
}
