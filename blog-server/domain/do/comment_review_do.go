package do

// CommentReview 描述一次审核决定。预期正文及状态来自管理员实际查看的内容，
// 用于拒绝并发编辑后的陈旧决定；不修改作者、正文或点赞计数。
type CommentReview struct {
	CommentID       uint64
	ExpectedContent string
	ExpectedStatus  int
	Status          int
}
