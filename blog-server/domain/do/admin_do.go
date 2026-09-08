package do

import "time"

// AdminDO 管理员领域对象
// 纯数据结构，不依赖数据库/框架，用于领域层传输
type AdminDO struct {
	ID           uint64     `json:"id"`
	AdminName    string     `json:"admin_name"`    // 管理员账号
	Password     string     `json:"-"`             // 密码脱敏，不序列化
	Avatar       string     `json:"avatar"`        // 头像
	Role         int        `json:"role"`          // 角色：0-超级管理员，1-普通管理员
	Status       int        `json:"status"`        // 状态：1-正常，0-禁用
	TokenVersion uint64     `json:"-"`             // 令牌版本号，仅用于鉴权失效
	LastLoginAt  *time.Time `json:"last_login_at"` // 最后登录时间
	CreateAt     time.Time  `json:"create_at"`     // 创建时间
	UpdateAt     time.Time  `json:"update_at"`     // 更新时间
	Email        string     `json:"email"`         // 邮箱
	Remark       string     `json:"remark"`        // 备注
}

// AdminInfoUpdate 管理员个人资料的部分更新。nil 表示保留原值；
// 头像、邮箱、备注可显式传空字符串清空，账号名必须非空。
// 角色、状态、密码和令牌版本由各自领域用例维护，不接受本用例修改。
type AdminInfoUpdate struct {
	AdminName *string
	Avatar    *string
	Email     *string
	Remark    *string
}

// FormatCreateAt 业务方法：格式化创建时间为字符串（给前端展示用）
func (a *AdminDO) FormatCreateAt() string {
	return a.CreateAt.Format("2006-01-02 15:04:05")
}

// FormatUpdateAt 业务方法：格式化更新时间
func (a *AdminDO) FormatUpdateAt() string {
	return a.UpdateAt.Format("2006-01-02 15:04:05")
}
