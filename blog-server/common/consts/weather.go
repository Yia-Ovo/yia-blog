package consts

import "time"

const (
	WeatherReady               = "ready"
	WeatherUnavailable         = "unavailable"
	WeatherSourceIP            = "ip"
	WeatherSourceManual        = "manual"
	WeatherLocationUnavailable = "location_unavailable"
	WeatherDataUnavailable     = "weather_unavailable"
	WeatherBusy                = "busy"
	WeatherRequestTimeout      = 3 * time.Second
	WeatherLoadTimeout         = 2800 * time.Millisecond
	WeatherHTTPTimeout         = 1800 * time.Millisecond
	WeatherCacheTTL            = 15 * time.Minute
	WeatherFailureTTL          = time.Minute
	WeatherCacheEntries        = 256
	WeatherMaxFlights          = 8
	WeatherLoadsPerMinute      = 30
	WeatherTotalLoadsPerDay    = 9000
	WeatherIPLoadsPerDay       = 900
	WeatherMaxResponseBytes    = 64 * 1024
)
