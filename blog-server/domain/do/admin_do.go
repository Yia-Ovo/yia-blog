package do

import "time"

// AdminDO 管理员领域对象
// 纯数据结构，不依赖数据库/框架，用于领域层传输
type AdminDO struct {
	ID          uint64     `json:"id"`
	AdminName   string     `json:"admin_name"`    // 管理员账号
	Password    string     `json:"-"`             // 密码脱敏，不序列化
	Avatar      string     `json:"avatar"`        // 头像
	Role        int        `json:"role"`          // 角色：0-超级管理员，1-普通管理员
	Status      int        `json:"status"`        // 状态：1-正常，0-禁用
	LastLoginAt *time.Time `json:"last_login_at"` // 最后登录时间
	CreateAt    time.Time  `json:"create_at"`     // 创建时间
	UpdateAt    time.Time  `json:"update_at"`     // 更新时间
	Email       string     `json:"email"`         // 邮箱
	Remark      string     `json:"remark"`        // 备注
}
