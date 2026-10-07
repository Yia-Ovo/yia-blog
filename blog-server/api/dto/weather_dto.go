package dto

import "time"

// WeatherCurrentReq 仅允许可选预设城市 slug；IP 和坐标只能由服务端确定。
type WeatherCurrentReq struct {
	City string `form:"city" binding:"max=32"`
}

type WeatherLocationResp struct {
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
	Source  string `json:"source"`
}

// WeatherDataResp 的单位固定为 ℃、%、km/h；UpdatedAt 为模型时间。
type WeatherDataResp struct {
	Temperature float64   `json:"temperature"`
	Code        int       `json:"code"`
	IsDay       bool      `json:"is_day"`
	Humidity    float64   `json:"humidity"`
	WindSpeed   float64   `json:"wind_speed"`
	Minimum     float64   `json:"minimum"`
	Maximum     float64   `json:"maximum"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WeatherCurrentResp 天气不可用仍是成功响应，不阻断首页其他区域；不暴露 IP 或坐标。
type WeatherCurrentResp struct {
	Status   string               `json:"status"`
	Location *WeatherLocationResp `json:"location"`
	Weather  *WeatherDataResp     `json:"weather"`
	Stale    bool                 `json:"stale"`
	Reason   string               `json:"reason,omitempty"`
}
