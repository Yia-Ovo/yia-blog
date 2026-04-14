package do

import "time"

type UserDO struct {
	ID          uint64     `json:"id"`
	NickName    string     `json:"nick_name"`
	Password    string     `json:"-"`
	Avatar      string     `json:"avatar"`
	Gender      int        `json:"gender"` // 性别：0-未知 1-男 2-女
	Status      int        `json:"status"`
	LastLoginAt *time.Time `json:"last_login_time"`
	CreateAt    time.Time  `json:"create_at"`
	UpdateAt    time.Time  `json:"update_at"`
	IsDeleted   bool       `json:"is_deleted"`
	Signature   string     `json:"signature"` // 个性签名
	Email       string     `json:"email"`
}

// FormatCreateAt 业务方法：格式化创建时间为字符串（给前端展示用）
func (u *UserDO) FormatCreateAt() string {
	return u.CreateAt.Format("2006-01-02 15:04:05")
}

// FormatUpdateAt 业务方法：格式化更新时间
func (u *UserDO) FormatUpdateAt() string {
	return u.UpdateAt.Format("2006-01-02 15:04:05")
}
