package dto

import "time"

// DailyAppearanceResp 只暴露当天图片 URL 和轮换时间，不暴露源文件名或本机路径。
// ServerTime 与 NextChangeAt 配合使用，使前端不依赖访客设备的日期设置。
type DailyAppearanceResp struct {
	Date         string    `json:"date"`
	ImageURL     string    `json:"image_url"`
	ServerTime   time.Time `json:"server_time"`
	NextChangeAt time.Time `json:"next_change_at"`
}
