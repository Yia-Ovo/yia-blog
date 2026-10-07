package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"math"
	"net/netip"
	"strings"
)

// WeatherDomainService 维护可选城市、公网 IP 与城市级位置合法性，不发 HTTP、不接触缓存。
type WeatherDomainService struct{}

func NewWeatherDomainService() *WeatherDomainService { return &WeatherDomainService{} }

var weatherCities = map[string]do.WeatherLocationDO{
	"nanning":   {City: "南宁", Region: "广西", Country: "China", Latitude: 22.82, Longitude: 108.37},
	"beijing":   {City: "北京", Region: "北京", Country: "China", Latitude: 39.90, Longitude: 116.40},
	"shanghai":  {City: "上海", Region: "上海", Country: "China", Latitude: 31.23, Longitude: 121.47},
	"guangzhou": {City: "广州", Region: "广东", Country: "China", Latitude: 23.13, Longitude: 113.26},
	"chengdu":   {City: "成都", Region: "四川", Country: "China", Latitude: 30.57, Longitude: 104.07},
	"tokyo":     {City: "Tokyo", Region: "Tokyo", Country: "Japan", Latitude: 35.68, Longitude: 139.69},
	"hanoi":     {City: "Hanoi", Region: "Hanoi", Country: "Vietnam", Latitude: 21.03, Longitude: 105.85},
	"london":    {City: "London", Region: "England", Country: "United Kingdom", Latitude: 51.51, Longitude: -0.13},
	"new-york":  {City: "New York", Region: "New York", Country: "United States", Latitude: 40.71, Longitude: -74.01},
}

var weatherNonPublicNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("64:ff9b::/96"),
}

// Select 手选城市优先。私网、回环、无效 IP 不会外发；没有可信位置时由前端提示手选。
func (s *WeatherDomainService) Select(city, ip string) (do.WeatherSelectionDO, error) {
	city = strings.TrimSpace(city)
	if city != "" {
		location, ok := weatherCities[city]
		if !ok {
			return do.WeatherSelectionDO{}, common.ParamErr
		}
		location.Source = consts.WeatherSourceManual
		return do.WeatherSelectionDO{CityID: city, Location: &location}, nil
	}
	address, err := netip.ParseAddr(ip)
	if err != nil || address.Zone() != "" {
		return do.WeatherSelectionDO{}, nil
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() {
		return do.WeatherSelectionDO{}, nil
	}
	for _, network := range weatherNonPublicNetworks {
		if network.Contains(address) {
			return do.WeatherSelectionDO{}, nil
		}
	}
	return do.WeatherSelectionDO{IP: address.String()}, nil
}

// NormalizeIPLocation 拒绝缺少城市或越界坐标，不能用零值假装获得了访问者所在地。
func (s *WeatherDomainService) NormalizeIPLocation(location do.WeatherLocationDO) (do.WeatherLocationDO, bool) {
	location.City = strings.TrimSpace(location.City)
	location.Region = strings.TrimSpace(location.Region)
	location.Country = strings.TrimSpace(location.Country)
	if location.City == "" || location.Country == "" || len(location.City) > 256 || len(location.Region) > 256 || len(location.Country) > 256 ||
		math.IsNaN(location.Latitude) || math.IsInf(location.Latitude, 0) || math.IsNaN(location.Longitude) || math.IsInf(location.Longitude, 0) ||
		location.Latitude < -90 || location.Latitude > 90 || location.Longitude < -180 || location.Longitude > 180 {
		return do.WeatherLocationDO{}, false
	}
	location.Source = consts.WeatherSourceIP
	return location, true
}
