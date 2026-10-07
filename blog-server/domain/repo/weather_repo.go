package repo

import (
	"blog-server/domain/do"
	"context"
)

// WeatherProvider 是外部城市估算和模型天气数据端口；Domain 不依赖 HTTP 实现。
type WeatherProvider interface {
	LocateIP(context.Context, string) (do.WeatherLocationDO, error)
	Current(context.Context, do.WeatherLocationDO) (do.WeatherDO, error)
}
