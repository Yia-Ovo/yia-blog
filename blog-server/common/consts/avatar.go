package consts

import "time"

const (
	AvatarMaxUploadBytes    = 5 << 20
	AvatarMultipartOverhead = 64 << 10
	AvatarMaxDimension      = 4096
	AvatarMaxPixels         = 4_000_000
	AvatarOutputDimension   = 256
	AvatarMaxStoredBytes    = 512 << 10
	AvatarDefaultDirectory  = "uploads/avatars"
	AvatarImageRoute        = "/api/blog/v1/avatar/images/"
	AvatarImageCacheControl = "public, max-age=31536000, immutable"
	AvatarRequestTimeout    = 10 * time.Second
)
