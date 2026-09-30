package dto

// ArticleImageUploadResp 返回可插入正文或封面的同源地址；上传不会创建或修改文章。
type ArticleImageUploadResp struct {
	URL         string `json:"url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ContentType string `json:"content_type"`
}
