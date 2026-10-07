package repo

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"time"
)

var errWeatherProvider = errors.New("weather provider unavailable")

// WeatherHTTPRepo 仅访问两个固定 HTTPS 上游；不允许用户指定主机、URL、IP 或坐标。
type WeatherHTTPRepo struct {
	client *http.Client
	now    func() time.Time
}

func NewWeatherHTTPRepo() *WeatherHTTPRepo {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = consts.WeatherMaxFlights
	transport.ResponseHeaderTimeout = consts.WeatherHTTPTimeout
	return &WeatherHTTPRepo{
		client: &http.Client{Timeout: consts.WeatherHTTPTimeout, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return errWeatherProvider }},
		now:    time.Now,
	}
}

// readJSON 限制响应体并要求一个完整 JSON；上游错误信息不透传给客户端或写入日志。
func (r *WeatherHTTPRepo) readJSON(ctx context.Context, endpoint string, result any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return errWeatherProvider
	}
	request.Header.Set("Accept", "application/json")
	response, err := r.client.Do(request)
	if err != nil {
		return errWeatherProvider
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errWeatherProvider
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errWeatherProvider
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, consts.WeatherMaxResponseBytes+1))
	if err != nil || len(body) > consts.WeatherMaxResponseBytes {
		return errWeatherProvider
	}
	if err := json.Unmarshal(body, result); err != nil {
		return errWeatherProvider
	}
	return nil
}

func (r *WeatherHTTPRepo) LocateIP(ctx context.Context, ip string) (do.WeatherLocationDO, error) {
	address, err := netip.ParseAddr(ip)
	if err != nil || address.Zone() != "" || !address.IsGlobalUnicast() || address.IsLoopback() || address.IsPrivate() {
		return do.WeatherLocationDO{}, errWeatherProvider
	}
	query := url.Values{"fields": {"success,city,region,country,latitude,longitude"}}
	var value struct {
		Success   bool     `json:"success"`
		City      string   `json:"city"`
		Region    string   `json:"region"`
		Country   string   `json:"country"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	}
	if err := r.readJSON(ctx, "https://ipwho.is/"+url.PathEscape(address.Unmap().String())+"?"+query.Encode(), &value); err != nil ||
		!value.Success || value.Latitude == nil || value.Longitude == nil {
		return do.WeatherLocationDO{}, errWeatherProvider
	}
	return do.WeatherLocationDO{City: value.City, Region: value.Region, Country: value.Country, Latitude: *value.Latitude, Longitude: *value.Longitude}, nil
}

func (r *WeatherHTTPRepo) Current(ctx context.Context, location do.WeatherLocationDO) (do.WeatherDO, error) {
	if !weatherNumber(location.Latitude, -90, 90) || !weatherNumber(location.Longitude, -180, 180) {
		return do.WeatherDO{}, errWeatherProvider
	}
	query := url.Values{
		"latitude":         {strconv.FormatFloat(location.Latitude, 'f', 2, 64)},
		"longitude":        {strconv.FormatFloat(location.Longitude, 'f', 2, 64)},
		"current":          {"temperature_2m,relative_humidity_2m,is_day,weather_code,wind_speed_10m"},
		"daily":            {"temperature_2m_max,temperature_2m_min"},
		"temperature_unit": {"celsius"}, "wind_speed_unit": {"kmh"},
		"timeformat": {"unixtime"}, "timezone": {"auto"}, "forecast_days": {"1"},
	}
	var value struct {
		Current struct {
			Time        *int64   `json:"time"`
			Temperature *float64 `json:"temperature_2m"`
			Humidity    *float64 `json:"relative_humidity_2m"`
			IsDay       *int     `json:"is_day"`
			Code        *int     `json:"weather_code"`
			Wind        *float64 `json:"wind_speed_10m"`
		} `json:"current"`
		CurrentUnits map[string]string `json:"current_units"`
		Daily        struct {
			Time    []int64    `json:"time"`
			Minimum []*float64 `json:"temperature_2m_min"`
			Maximum []*float64 `json:"temperature_2m_max"`
		} `json:"daily"`
		DailyUnits map[string]string `json:"daily_units"`
	}
	if err := r.readJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+query.Encode(), &value); err != nil {
		return do.WeatherDO{}, err
	}
	c, d := value.Current, value.Daily
	if c.Time == nil || c.Temperature == nil || c.Humidity == nil || c.IsDay == nil || c.Code == nil || c.Wind == nil ||
		len(d.Time) != 1 || len(d.Minimum) != 1 || len(d.Maximum) != 1 || d.Minimum[0] == nil || d.Maximum[0] == nil ||
		value.CurrentUnits["temperature_2m"] != "°C" || value.CurrentUnits["relative_humidity_2m"] != "%" ||
		value.CurrentUnits["wind_speed_10m"] != "km/h" || value.DailyUnits["temperature_2m_min"] != "°C" || value.DailyUnits["temperature_2m_max"] != "°C" {
		return do.WeatherDO{}, errWeatherProvider
	}
	updated := time.Unix(*c.Time, 0).UTC()
	now := r.now()
	if !weatherNumber(*c.Temperature, -100, 70) || !weatherNumber(*c.Humidity, 0, 100) || !weatherNumber(*c.Wind, 0, 500) ||
		!weatherNumber(*d.Minimum[0], -100, 70) || !weatherNumber(*d.Maximum[0], -100, 70) || *d.Minimum[0] > *d.Maximum[0] ||
		(*c.IsDay != 0 && *c.IsDay != 1) || !validWeatherCode(*c.Code) ||
		updated.Before(now.Add(-2*time.Hour)) || updated.After(now.Add(time.Hour)) ||
		d.Time[0] > *c.Time || d.Time[0] <= *c.Time-26*60*60 {
		return do.WeatherDO{}, errWeatherProvider
	}
	return do.WeatherDO{Temperature: *c.Temperature, Code: *c.Code, IsDay: *c.IsDay == 1, Humidity: *c.Humidity, WindSpeed: *c.Wind,
		Minimum: *d.Minimum[0], Maximum: *d.Maximum[0], UpdatedAt: updated}, nil
}

func weatherNumber(value, minimum, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= maximum
}

func validWeatherCode(code int) bool {
	switch code {
	case 0, 1, 2, 3, 45, 48, 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 71, 73, 75, 77, 80, 81, 82, 85, 86, 95, 96, 99:
		return true
	default:
		return false
	}
}
