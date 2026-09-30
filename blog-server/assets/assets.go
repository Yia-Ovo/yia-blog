// Package assets 提供随服务发布的每日主题图片，不依赖开发机素材目录。
package assets

import "embed"

// DailyArtwork 仅包含已压缩图片与显式目录，原始素材和其他桌面文件不进入服务。
//
//go:embed appearance/*.jpg appearance/catalog.json
var DailyArtwork embed.FS
