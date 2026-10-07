package repo

import "syscall"

// publishArticleImageFile 使用不覆盖目标的 MoveFile，避免替换已有文件时与 Windows 读句柄冲突。
// syscall.Rename 在 Windows 调用 MoveFile；os.Rename 会设置 REPLACE_EXISTING。
func publishArticleImageFile(temporary, destination string) error {
	return syscall.Rename(temporary, destination)
}
