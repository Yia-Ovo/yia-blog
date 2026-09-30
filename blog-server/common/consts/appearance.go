package consts

import "time"

const (
	AppearanceCatalogPath       = "appearance/catalog.json"
	AppearanceAssetPrefix       = "appearance/"
	AppearanceImageRoute        = "/api/blog/v1/appearance/images/"
	AppearanceTimezone          = "Asia/Shanghai"
	AppearanceUTCOffset         = 8 * 60 * 60
	AppearanceRequestTimeout    = 3 * time.Second
	AppearanceDaySeconds        = 24 * 60 * 60
	AppearanceImageCacheControl = "public, max-age=31536000, immutable"
)
