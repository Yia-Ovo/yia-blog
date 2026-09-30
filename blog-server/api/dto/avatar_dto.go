package dto

// AvatarUploadResp 是已保存到本人资料的同源头像地址，不包含磁盘路径。
type AvatarUploadResp struct {
	Avatar string `json:"avatar"`
}
