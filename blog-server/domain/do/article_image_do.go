package do

// ArticleImage 是已规范化的文章图片；保留原比例，Data 不包含上传文件名或元信息。
type ArticleImage struct {
	Data        []byte
	ContentType string
	Width       int
	Height      int
}
