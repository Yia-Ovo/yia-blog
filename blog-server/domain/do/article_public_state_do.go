package do

// ArticlePublicState 是公开可见性判断所需的最小数据库快照，不包含正文。
// 仓储只返回未软删除的记录；不存在或已软删除以 nil 表示。
type ArticlePublicState struct {
	ID     uint64
	Status int
}
