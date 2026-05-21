package do

import (
	"time"

	"gorm.io/gorm"
)

type CommentDO struct {
	ID        uint64
	ArticleID uint64
	ParentID  uint64
	AdminID   uint64
	Nickname  string
	Email     string
	Avatar    string
	Content   string
	IP        string
	UserAgent string
	Status    int
	Likes     int
	CreateAt  time.Time
	UpdateAt  time.Time
	DeletedAt gorm.DeletedAt
}

// CommentQuery 评论查询条件（前端传参 + 后端SQL筛选）
type CommentQuery struct {
	// 1. 精准筛选条件（等值查询）
	ArticleID uint64 `form:"articleID" json:"articleID"` // 必传：文章ID（查某篇文章的评论）
	ParentID  uint64 `form:"parentID" json:"parentID"`   // 父评论ID：0=顶级评论，非0=回复
	Status    int    `form:"status" json:"status"`       // 评论状态：0-待审核 1-已通过 2-已拒绝
	AdminID   uint64 `form:"adminID" json:"adminID"`     // 管理员ID：0=游客，>0=管理员评论

	// 2. 模糊搜索条件
	Nickname string `form:"nickname" json:"nickname"` // 搜索评论人昵称
	Content  string `form:"content" json:"content"`   // 搜索评论内容

	// 3. 分页参数（所有列表接口必带）
	Page int `form:"page" json:"page"` // 页码，默认 1
	Size int `form:"size" json:"size"` // 每页条数，默认 10

	// 4. 排序参数
	OrderBy  string `form:"orderBy" json:"orderBy"`   // 排序字段：create_at / likes / id
	OrderDir string `form:"orderDir" json:"orderDir"` // 排序方向：asc / desc

	// 5. 软删除控制
	WithDeleted bool `form:"withDeleted" json:"withDeleted"` // 是否包含已删除的评论（管理员用）
}
