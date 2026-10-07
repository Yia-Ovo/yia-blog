//go:build !windows

package repo

import "os"

// publishArticleImageFile 在其他平台沿用同目录原子重命名。
func publishArticleImageFile(temporary, destination string) error {
	return os.Rename(temporary, destination)
}
