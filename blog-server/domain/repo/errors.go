/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-07-23 19:35:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-08-12 12:01:14
 * @FilePath: \yia-blog\blog-server\domain\repo\errors.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import "errors"

// ErrCacheMiss 表示缓存未命中。Repo 实现应在底层缓存（如 go-redis 的 redis.Nil）
// 未命中时返回它，避免把具体缓存驱动的错误类型泄漏到 App 层。
var ErrCacheMiss = errors.New("repo: cache miss")

// ErrDuplicateKey 表示唯一约束冲突，由 Repo 翻译驱动错误，Domain 决定业务错误码。
var ErrDuplicateKey = errors.New("repo: duplicate key")

// ErrNoRowsAffected 表示带业务条件的更新未命中任何行。Repo 实现应在 WHERE 叠加了
// 业务约束（如 AND role = ?）却影响 0 行时返回它，让 Domain 能区分「更新成功」与
// 「条件不满足」——目标不存在、已软删除，或在权限校验通过后被并发改动。
//
// 只有当更新列中含有每次必变的列（如 token_version、update_at）时，RowsAffected == 0
// 才等价于「WHERE 未匹配」；否则「匹配到但新旧值相同」也会返回 0，不可据此判失败。
var ErrNoRowsAffected = errors.New("repo: no rows affected")
