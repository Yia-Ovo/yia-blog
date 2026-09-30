package consts

import "time"

const (
	ArticleImageMaxUploadBytes    = 5 << 20
	ArticleImageMultipartOverhead = 64 << 10
	ArticleImageMaxDimension      = 8192
	ArticleImageMaxPixels         = 4_000_000
	ArticleImageOutputDimension   = 1920
	ArticleImageJPEGQuality       = 85
	ArticleImageMaxStoredBytes    = 16 << 20
	ArticleImageDefaultDirectory  = "uploads/articles"
	ArticleImageRoute             = "/api/blog/v1/article/images/"
	ArticleImageCacheControl      = "public, max-age=31536000, immutable"
	ArticleImageRequestTimeout    = 10 * time.Second
)
