package do

import "time"

// WeatherLocationDO 表示城市级估算位置；坐标只在服务端请求天气时使用，不返回客户端。
type WeatherLocationDO struct {
	City, Region, Country string
	Latitude, Longitude   float64
	Source                string
}

// WeatherDO 表示模型天气，温度为摄氏度、湿度为百分比、风速为 km/h。
// UpdatedAt 是模型数据时间，不是本站拉取时间，也不是气象站实测时间。
type WeatherDO struct {
	Temperature      float64
	Code             int
	IsDay            bool
	Humidity         float64
	WindSpeed        float64
	Minimum, Maximum float64
	UpdatedAt        time.Time
}

// WeatherSelectionDO 的 IP 仅由 HTTP 请求的可信来源产生，客户端不能指定任意 IP 或坐标。
type WeatherSelectionDO struct {
	CityID   string
	IP       string
	Location *WeatherLocationDO
}
