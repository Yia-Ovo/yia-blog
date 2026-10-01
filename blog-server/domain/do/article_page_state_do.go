package do

// ArticlePageState 表示当前筛选范围内的有序分页身份及总数，不包含文章正文或作者资料。
// IDs 的顺序与公开列表一致；Total 是全部匹配文章数，不是当前页长度。
type ArticlePageState struct {
	IDs   []uint64
	Total int64
}
