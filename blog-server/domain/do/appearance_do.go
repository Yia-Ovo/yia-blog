package do

import "time"

// DailyAppearanceDO 表示北京时间一个自然日内所有访客共享的图片。
// ImageID 来自受控素材目录；NextChangeAt 是下一个自然日开始时间。
type DailyAppearanceDO struct {
	ImageID      string
	Date         string
	NextChangeAt time.Time
}
